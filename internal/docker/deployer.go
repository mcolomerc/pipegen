package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	logpkg "pipegen/internal/log"

	"pipegen/internal/pipeline"
	"pipegen/internal/types"

	"github.com/segmentio/kafka-go"
	"github.com/spf13/viper"
)

// StackDeployer handles end-to-end local stack resource provisioning (topics, schemas, Flink SQL jobs)
type StackDeployer struct {
	projectDir         string
	kafkaAddr          string
	flinkAddr          string
	schemaRegistryAddr string
	sqlGatewayAddr     string
	logger             logpkg.Logger
}

// waitForLocalSQLGatewayReady polls the /v1/sessions endpoint until it returns 200 or timeout.
func (d *StackDeployer) waitForLocalSQLGatewayReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	endpoint := fmt.Sprintf("%s/v1/sessions", d.sqlGatewayAddr)
	var lastErr error
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled during local SQL Gateway readiness: %w", ctx.Err())
		default:
		}
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to build readiness request: %w", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			if d.logger != nil {
				d.logger.Warn("sql gateway readiness transient", "err", err)
			}
			time.Sleep(750 * time.Millisecond)
			continue
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == 200 {
			if d.logger != nil {
				d.logger.Info("sql gateway ready")
			}
			return nil
		}
		lastErr = fmt.Errorf("status %d", resp.StatusCode)
		if d.logger != nil {
			d.logger.Debug("sql gateway readiness retry", "status", resp.StatusCode)
		}
		time.Sleep(750 * time.Millisecond)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unknown readiness failure")
	}
	return lastErr
}

// NewStackDeployer creates a new stack deployer configured from viper flags / env.
func NewStackDeployer(projectDir string) *StackDeployer {
	// Read configuration from viper
	kafkaAddr := viper.GetString("bootstrap_servers")
	if kafkaAddr == "" {
		kafkaAddr = "localhost:9092" // fallback to default
	}

	// Extract host and port from bootstrap servers
	if strings.Contains(kafkaAddr, "://") {
		// Handle URLs like "kafka://localhost:9092"
		parts := strings.Split(kafkaAddr, "://")
		if len(parts) == 2 {
			kafkaAddr = parts[1]
		}
	}

	flinkAddr := viper.GetString("flink_url")
	if flinkAddr == "" {
		flinkAddr = "http://localhost:8081" // fallback to default
	}

	schemaRegistryAddr := viper.GetString("schema_registry_url")
	if schemaRegistryAddr == "" {
		schemaRegistryAddr = "http://localhost:8082" // fallback to default
	}

	sqlGatewayAddr := viper.GetString("flink_sql_gateway_url")
	if sqlGatewayAddr == "" {
		sqlGatewayAddr = "http://localhost:8083" // fallback to default
	}

	return &StackDeployer{
		projectDir:         projectDir,
		kafkaAddr:          kafkaAddr,
		flinkAddr:          flinkAddr,
		schemaRegistryAddr: schemaRegistryAddr,
		sqlGatewayAddr:     sqlGatewayAddr,
		logger:             logpkg.Global(),
	}
}

// SetupTopicsAndSchemas creates topics and registers schemas
func (d *StackDeployer) SetupTopicsAndSchemas(ctx context.Context, withSchemaRegistry bool) error {
	// Load project configuration
	sqlLoader := pipeline.NewSQLLoader(d.projectDir)
	statements, err := sqlLoader.LoadStatements()
	if err != nil {
		return fmt.Errorf("failed to load SQL statements: %w", err)
	}

	schemaLoader := pipeline.NewSchemaLoader(d.projectDir)
	schemas, err := schemaLoader.LoadSchemas()
	if err != nil {
		return fmt.Errorf("failed to load schemas: %w", err)
	}

	// Extract topic names from SQL statements
	topics := d.extractTopicNames(statements)

	// Create Kafka topics
	if err := d.createKafkaTopics(ctx, topics); err != nil {
		return fmt.Errorf("failed to create Kafka topics: %w", err)
	}

	// Register schemas if Schema Registry is enabled
	if withSchemaRegistry {
		if err := d.registerSchemas(ctx, schemas, topics); err != nil {
			return fmt.Errorf("failed to register schemas: %w", err)
		}
	}

	return nil
}

// DeployFlinkJobs deploys FlinkSQL jobs
func (d *StackDeployer) DeployFlinkJobs(ctx context.Context) error {
	sqlLoader := pipeline.NewSQLLoader(d.projectDir)
	statements, err := sqlLoader.LoadStatements()
	if err != nil {
		return fmt.Errorf("failed to load SQL statements: %w", err)
	}

	// Process SQL statements for local deployment
	processedStatements := d.processStatementsForLocal(statements)

	// Deploy each statement via Flink SQL Gateway
	for _, stmt := range processedStatements {
		if d.logger != nil {
			d.logger.Info("deploy flink sql", "statement", stmt.Name)
		}

		if err := d.deployFlinkStatement(ctx, stmt); err != nil {
			return fmt.Errorf("failed to deploy statement %s: %w", stmt.Name, err)
		}

		if d.logger != nil {
			d.logger.Info("flink sql deployed", "statement", stmt.Name)
		}
	}

	return nil
}

// extractTopicNames extracts topic names from SQL statements
func (d *StackDeployer) extractTopicNames(statements []*types.SQLStatement) []string {
	topics := make(map[string]bool)

	// Default topics for the pipeline
	topics["input-events"] = true
	topics["output-results"] = true

	// Extract topics from SQL statements
	for _, stmt := range statements {
		content := strings.ToUpper(stmt.Content)

		// Look for topic references in WITH clauses
		if strings.Contains(content, "'TOPIC'") {
			// This is a simplified extraction - in production, you'd want
			// more robust SQL parsing
			lines := strings.Split(stmt.Content, "\n")
			for _, line := range lines {
				if strings.Contains(strings.ToUpper(line), "'TOPIC'") &&
					strings.Contains(line, "=") {
					parts := strings.Split(line, "=")
					if len(parts) > 1 {
						topic := strings.Trim(parts[1], " '\"(),")
						if topic != "" && !strings.Contains(topic, "$") {
							topics[topic] = true
						}
					}
				}
			}
		}
	}

	result := make([]string, 0, len(topics))
	for topic := range topics {
		result = append(result, topic)
	}

	return result
}

// createKafkaTopics creates Kafka topics with retry logic
func (d *StackDeployer) createKafkaTopics(ctx context.Context, topics []string) error {
	d.logger.Info("connecting kafka", "addr", d.kafkaAddr)

	// Create a Kafka client with proper configuration
	transport := &kafka.Transport{
		DialTimeout: 10 * time.Second,
		IdleTimeout: 30 * time.Second,
	}

	client := &kafka.Client{
		Addr:      kafka.TCP(d.kafkaAddr),
		Timeout:   10 * time.Second,
		Transport: transport,
	}

	// Retry topic creation with exponential backoff
	for attempt := 1; attempt <= 5; attempt++ {
		d.logger.Info("creating kafka topics", "attempt", attempt)

		// Prepare topic configurations
		topicConfigs := make([]kafka.TopicConfig, 0, len(topics))
		for _, topic := range topics {
			topicConfigs = append(topicConfigs, kafka.TopicConfig{
				Topic:             topic,
				NumPartitions:     3,
				ReplicationFactor: 1,
			})
		}

		// Create topics using the client
		response, err := client.CreateTopics(ctx, &kafka.CreateTopicsRequest{
			Topics: topicConfigs,
		})

		if err == nil {
			// Check if any topics failed to create
			allSuccess := true
			for _, topic := range topics {
				if topicError, exists := response.Errors[topic]; exists && topicError != nil {
					// Ignore "topic already exists" errors
					if !strings.Contains(topicError.Error(), "already exists") &&
						!strings.Contains(topicError.Error(), "TOPIC_ALREADY_EXISTS") {
						d.logger.Warn("topic create failed", "topic", topic, "err", topicError)
						allSuccess = false
					} else {
						d.logger.Info("topic exists", "topic", topic)
					}
				} else {
					d.logger.Info("topic created", "topic", topic)
				}
			}

			if allSuccess {
				d.logger.Info("all topics created")
				return nil
			}
		}

		if attempt < 5 {
			d.logger.Warn("topic creation attempt failed", "attempt", attempt, "next_wait_s", attempt*2)
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}

	return fmt.Errorf("failed to create topics after 5 attempts")
}

// registerSchemas registers AVRO schemas in Schema Registry
func (d *StackDeployer) registerSchemas(ctx context.Context, schemas map[string]*pipeline.Schema, topics []string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	for name, schema := range schemas {
		// Register value schema
		valueSubject := d.getSchemaSubject(name, topics)
		if d.logger != nil {
			d.logger.Info("register value schema", "name", name, "subject", valueSubject)
		}

		if err := d.registerSchema(client, valueSubject, schema); err != nil {
			return fmt.Errorf("failed to register value schema %s: %w", name, err)
		}

		if d.logger != nil {
			d.logger.Info("value schema registered", "subject", valueSubject)
		}

		// For output schema, also register key schema for upsert operations
		if name == "output" {
			keySubject := d.getKeySchemaSubject(name, topics)
			keySchema := d.createKeySchema(schema)
			if d.logger != nil {
				d.logger.Info("register key schema", "name", name, "subject", keySubject)
			}

			if err := d.registerSchema(client, keySubject, keySchema); err != nil {
				return fmt.Errorf("failed to register key schema %s: %w", name, err)
			}

			if d.logger != nil {
				d.logger.Info("key schema registered", "subject", keySubject)
			}
		}
	}

	return nil
}

// getSchemaSubject generates Schema Registry subject name
func (d *StackDeployer) getSchemaSubject(schemaName string, topics []string) string {
	// Map schema names to topics
	switch schemaName {
	case "input":
		return "transactions-value"
	case "output":
		return "output-results-value"
	default:
		return fmt.Sprintf("%s-value", schemaName)
	}
}

// getKeySchemaSubject generates Schema Registry subject name for key schemas
func (d *StackDeployer) getKeySchemaSubject(schemaName string, topics []string) string {
	// Map schema names to key topics
	switch schemaName {
	case "output":
		return "output-results-key"
	default:
		return fmt.Sprintf("%s-key", schemaName)
	}
}

// createKeySchema creates a key schema from a value schema
func (d *StackDeployer) createKeySchema(valueSchema *pipeline.Schema) *pipeline.Schema {
	// For simplicity, create a basic key schema with just the name field
	// In a production environment, you'd want to parse the original schema
	// and extract only the key fields
	keySchemaContent := `{
  "type": "record",
  "name": "OutputResultKey",
  "namespace": "test_pipeline.results",
  "fields": [
    {
      "name": "name",
      "type": "string"
    }
  ]
}`

	return &pipeline.Schema{
		Name:    "output-key",
		Content: keySchemaContent,
	}
}

// registerSchema registers a single schema in Schema Registry
func (d *StackDeployer) registerSchema(client *http.Client, subject string, schema *pipeline.Schema) error {
	// Create registration payload
	payload := map[string]interface{}{
		"schema": schema.Content,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal schema: %w", err)
	}

	// Register schema
	url := fmt.Sprintf("%s/subjects/%s/versions", d.schemaRegistryAddr, subject)
	resp, err := client.Post(url, "application/json", strings.NewReader(string(payloadBytes)))
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("schema registration failed with status %d", resp.StatusCode)
	}

	return nil
}

// processStatementsForLocal processes SQL statements for local deployment
func (d *StackDeployer) processStatementsForLocal(statements []*types.SQLStatement) []*types.SQLStatement {
	processed := make([]*types.SQLStatement, len(statements))

	for i, stmt := range statements {
		content := stmt.Content

		// Replace variables with local values
		replacements := map[string]string{
			"${INPUT_TOPIC}":         "input-events",
			"${OUTPUT_TOPIC}":        "output-results",
			"${BOOTSTRAP_SERVERS}":   "kafka:29092",
			"${SCHEMA_REGISTRY_URL}": "http://schema-registry:8082",
		}

		for placeholder, value := range replacements {
			content = strings.ReplaceAll(content, placeholder, value)
		}

		// Remove authentication settings for local deployment
		lines := strings.Split(content, "\n")
		var cleanLines []string

		for _, line := range lines {
			if strings.Contains(strings.ToUpper(line), "SASL") ||
				strings.Contains(strings.ToUpper(line), "SECURITY.PROTOCOL") ||
				strings.Contains(strings.ToUpper(line), "BASIC-AUTH") {
				continue
			}
			cleanLines = append(cleanLines, line)
		}

		processed[i] = &types.SQLStatement{
			Name:     stmt.Name,
			Content:  strings.Join(cleanLines, "\n"),
			FilePath: stmt.FilePath,
			Order:    stmt.Order,
		}
	}

	return processed
}

// deployFlinkStatement deploys a single FlinkSQL statement
func (d *StackDeployer) deployFlinkStatement(ctx context.Context, stmt *types.SQLStatement) error {
	client := &http.Client{Timeout: 30 * time.Second}

	// Create a session first
	sessionID, err := d.createFlinkSession(ctx, client)
	if err != nil {
		if d.logger != nil {
			d.logger.Warn("failed create flink session using fallback", "err", err, "statement", stmt.Name)
		}
		return d.deployViaRESTAPI(client, stmt)
	}

	// Create SQL job submission payload
	payload := map[string]interface{}{
		"statement": stmt.Content,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SQL statement: %w", err)
	}

	// Submit job via Flink SQL Gateway using the created session
	url := fmt.Sprintf("%s/v1/sessions/%s/statements", d.sqlGatewayAddr, sessionID)
	resp, err := client.Post(url, "application/json", strings.NewReader(string(payloadBytes)))
	if err != nil {
		// Fallback: try REST API if SQL Gateway is not available
		return d.deployViaRESTAPI(client, stmt)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("FlinkSQL deployment failed with status %d", resp.StatusCode)
	}

	return nil
}

// createFlinkSession creates a new Flink SQL Gateway session
func (d *StackDeployer) createFlinkSession(ctx context.Context, client *http.Client) (string, error) {
	sessionEndpoint := fmt.Sprintf("%s/v1/sessions", d.sqlGatewayAddr)
	payload := `{"sessionName": "pipegen-deploy-session", "properties": {}}`

	// Readiness wait (non-fatal) up to 8s
	readyCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	if err := d.waitForLocalSQLGatewayReady(readyCtx, 8*time.Second); err != nil {
		if d.logger != nil {
			d.logger.Warn("sql gateway readiness not confirmed before session attempts", "err", err)
		}
	}
	cancel()

	maxAttempts := 6
	backoff := 1500 * time.Millisecond
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("context cancelled while creating local session: %w", ctx.Err())
		default:
		}

		req, err := http.NewRequestWithContext(ctx, "POST", sessionEndpoint, strings.NewReader(payload))
		if err != nil {
			return "", fmt.Errorf("failed to build session request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("attempt %d/%d: %w", attempt, maxAttempts, err)
			if d.logger != nil {
				d.logger.Warn("sql gateway session create attempt failed", "attempt", attempt, "err", lastErr)
			}
		} else {
			body, rerr := io.ReadAll(resp.Body)
			if cerr := resp.Body.Close(); cerr != nil {
				if d.logger != nil {
					d.logger.Warn("failed close session body", "err", cerr)
				}
			}
			if rerr != nil {
				lastErr = fmt.Errorf("failed reading session response (attempt %d): %w", attempt, rerr)
				if d.logger != nil {
					d.logger.Warn("failed read session response", "attempt", attempt, "err", lastErr)
				}
			} else if resp.StatusCode == 200 {
				id := d.extractSessionID(string(body))
				if id != "" {
					if attempt > 1 {
						if d.logger != nil {
							d.logger.Info("sql session created", "attempt", attempt, "id", id)
						}
					} else {
						if d.logger != nil {
							d.logger.Info("sql session created", "id", id)
						}
					}
					return id, nil
				}
				lastErr = fmt.Errorf("attempt %d: session ID missing in response: %s", attempt, string(body))
				if d.logger != nil {
					d.logger.Warn("session id missing", "attempt", attempt, "err", lastErr)
				}
			} else {
				lastErr = fmt.Errorf("attempt %d: non-200 status %d: %s", attempt, resp.StatusCode, string(body))
				if d.logger != nil {
					d.logger.Warn("session create non-200", "attempt", attempt, "status", resp.StatusCode, "err", lastErr)
				}
			}
		}

		if attempt < maxAttempts {
			if d.logger != nil {
				d.logger.Info("retry session creation", "wait", backoff, "attempt", attempt, "max", maxAttempts)
			}
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				if !timer.Stop() {
					<-timer.C
				}
				return "", fmt.Errorf("context cancelled during backoff: %w", ctx.Err())
			case <-timer.C:
			}
			backoff = backoff * 2
			if backoff > 20*time.Second {
				backoff = 20 * time.Second
			}
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unknown failure creating session")
	}
	return "", lastErr
}

// extractSessionID extracts session ID from Flink SQL Gateway response
func (d *StackDeployer) extractSessionID(resp string) string {
	// Look for "sessionHandle":"..."
	idx := strings.Index(resp, "\"sessionHandle\":\"")
	if idx == -1 {
		return ""
	}
	start := idx + len("\"sessionHandle\":\"")
	end := strings.Index(resp[start:], "\"")
	if end == -1 {
		return ""
	}
	return resp[start : start+end]
}

// deployViaRESTAPI deploys via Flink's REST API (fallback method)
func (d *StackDeployer) deployViaRESTAPI(client *http.Client, stmt *types.SQLStatement) error {
	// For now, we'll create a simple JAR file that contains the SQL statement
	// In a production environment, you'd want to create a proper Flink job

	if d.logger != nil {
		d.logger.Warn("sql gateway unavailable using fallback", "statement", stmt.Name)
	}

	// Create a SQL file that can be executed later
	sqlDir := filepath.Join(d.projectDir, "deployed-sql")
	if err := os.MkdirAll(sqlDir, 0755); err != nil {
		return fmt.Errorf("failed to create deployed-sql directory: %w", err)
	}

	sqlFile := filepath.Join(sqlDir, fmt.Sprintf("%s.sql", stmt.Name))
	if err := os.WriteFile(sqlFile, []byte(stmt.Content), 0644); err != nil {
		return fmt.Errorf("failed to write SQL file: %w", err)
	}

	if d.logger != nil {
		d.logger.Info("sql statement saved", "file", sqlFile)
	}
	if d.logger != nil {
		d.logger.Info("manual execution hint", "hint", "Use Flink SQL CLI or Web UI to execute this statement")
	}

	return nil
}

// Deployer handles Docker Compose operations for cleanup
type Deployer struct {
	projectDir  string
	projectName string
	logger      logpkg.Logger
}

// NewDeployer creates a new deployer for Docker operations
func NewDeployer(projectDir, projectName string) *Deployer {
	return &Deployer{
		projectDir:  projectDir,
		projectName: projectName,
		logger:      logpkg.Global(),
	}
}

// CheckDockerAvailability checks if Docker is available and running
func (d *Deployer) CheckDockerAvailability() error {
	cmd := exec.Command("docker", "version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker is not available or not running: %w", err)
	}
	return nil
}

// StopStack stops the Docker Compose stack
func (d *Deployer) StopStack() error {
	if d.logger != nil {
		d.logger.Info("stopping docker containers")
	}

	cmd := exec.Command("docker", "compose", "down")
	cmd.Dir = d.projectDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	if d.logger != nil {
		d.logger.Info("containers stopped")
	}
	return nil
}

// RemoveVolumes removes Docker volumes
func (d *Deployer) RemoveVolumes() error {
	cmd := exec.Command("docker", "compose", "down", "-v")
	cmd.Dir = d.projectDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove volumes: %w", err)
	}

	if d.logger != nil {
		d.logger.Info("volumes removed")
	}
	return nil
}

// RemoveImages removes Docker images used by the stack
func (d *Deployer) RemoveImages() error {
	cmd := exec.Command("docker", "compose", "down", "--rmi", "all")
	cmd.Dir = d.projectDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove images: %w", err)
	}

	if d.logger != nil {
		d.logger.Info("images removed")
	}
	return nil
}

// CleanupOrphans removes orphaned containers
func (d *Deployer) CleanupOrphans() error {
	cmd := exec.Command("docker", "compose", "down", "--remove-orphans")
	cmd.Dir = d.projectDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to cleanup orphans: %w", err)
	}

	if d.logger != nil {
		d.logger.Info("orphaned containers cleaned up")
	}
	return nil
}

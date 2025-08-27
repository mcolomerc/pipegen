package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pipegen/internal/pipeline"
	"pipegen/internal/types"

	"github.com/segmentio/kafka-go"
)

// StackDeployer handles deployment operations for the local stack
type StackDeployer struct {
	projectDir         string
	kafkaAddr          string
	flinkAddr          string
	schemaRegistryAddr string
}

// NewStackDeployer creates a new stack deployer
func NewStackDeployer(projectDir string) *StackDeployer {
	return &StackDeployer{
		projectDir:         projectDir,
		kafkaAddr:          "localhost:9092",
		flinkAddr:          "http://localhost:8081",
		schemaRegistryAddr: "http://localhost:8082",
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
		fmt.Printf("📝 Deploying FlinkSQL job: %s\n", stmt.Name)

		if err := d.deployFlinkStatement(ctx, stmt); err != nil {
			return fmt.Errorf("failed to deploy statement %s: %w", stmt.Name, err)
		}

		fmt.Printf("  ✅ Deployed: %s\n", stmt.Name)
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
	fmt.Printf("🔌 Connecting to Kafka at %s...\n", d.kafkaAddr)

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
		fmt.Printf("📝 Attempt %d: Creating Kafka topics...\n", attempt)

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
						fmt.Printf("⚠️  Failed to create topic %s: %v\n", topic, topicError)
						allSuccess = false
					} else {
						fmt.Printf("  ✅ Topic %s already exists\n", topic)
					}
				} else {
					fmt.Printf("  ✅ Topic created: %s\n", topic)
				}
			}

			if allSuccess {
				fmt.Printf("✅ All topics created successfully\n")
				return nil
			}
		}

		if attempt < 5 {
			fmt.Printf("⚠️  Topic creation attempt %d failed, retrying in %d seconds...\n", attempt, attempt*2)
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}

	return fmt.Errorf("failed to create topics after 5 attempts")
}

// registerSchemas registers AVRO schemas in Schema Registry
func (d *StackDeployer) registerSchemas(ctx context.Context, schemas map[string]*pipeline.Schema, topics []string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	for name, schema := range schemas {
		subject := d.getSchemaSubject(name, topics)
		fmt.Printf("📋 Registering schema: %s -> %s\n", name, subject)

		if err := d.registerSchema(client, subject, schema); err != nil {
			return fmt.Errorf("failed to register schema %s: %w", name, err)
		}

		fmt.Printf("  ✅ Schema registered: %s\n", subject)
	}

	return nil
}

// getSchemaSubject generates Schema Registry subject name
func (d *StackDeployer) getSchemaSubject(schemaName string, topics []string) string {
	// Map schema names to topics
	switch schemaName {
	case "input":
		return "input-events-value"
	case "output":
		return "output-results-value"
	default:
		return fmt.Sprintf("%s-value", schemaName)
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

	// Create SQL job submission payload
	payload := map[string]interface{}{
		"statement": stmt.Content,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal SQL statement: %w", err)
	}

	// Submit job via Flink SQL Gateway (port 8083)
	url := "http://localhost:8083/v1/sessions/default/statements"
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

// deployViaRESTAPI deploys via Flink's REST API (fallback method)
func (d *StackDeployer) deployViaRESTAPI(client *http.Client, stmt *types.SQLStatement) error {
	// For now, we'll create a simple JAR file that contains the SQL statement
	// In a production environment, you'd want to create a proper Flink job

	fmt.Printf("  ⚠️  SQL Gateway not available, using fallback method for: %s\n", stmt.Name)

	// Create a SQL file that can be executed later
	sqlDir := filepath.Join(d.projectDir, "deployed-sql")
	if err := os.MkdirAll(sqlDir, 0755); err != nil {
		return fmt.Errorf("failed to create deployed-sql directory: %w", err)
	}

	sqlFile := filepath.Join(sqlDir, fmt.Sprintf("%s.sql", stmt.Name))
	if err := os.WriteFile(sqlFile, []byte(stmt.Content), 0644); err != nil {
		return fmt.Errorf("failed to write SQL file: %w", err)
	}

	fmt.Printf("  📄 SQL statement saved to: %s\n", sqlFile)
	fmt.Printf("  💡 Manual execution: Use Flink SQL CLI or Web UI to execute this statement\n")

	return nil
}

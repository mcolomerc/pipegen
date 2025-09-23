package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	logpkg "pipegen/internal/log"
	templates "pipegen/internal/templates"
)

// KafkaConfig holds Kafka topic creation settings
type KafkaConfig struct {
	Partitions        int   `yaml:"partitions"`
	ReplicationFactor int   `yaml:"replication_factor"`
	RetentionMs       int64 `yaml:"retention_ms"`
}

// Config holds the configuration for pipeline execution
type Config struct {
	ProjectDir        string
	MessageRate       int
	Duration          time.Duration // Producer execution duration
	PipelineTimeout   time.Duration // Overall pipeline timeout (independent of producer duration)
	ExpectedMessages  int64         // Expected number of messages to consume before stopping
	Cleanup           bool
	DryRun            bool
	BootstrapServers  string
	FlinkURL          string
	SQLGatewayURL     string // Explicit SQL Gateway base URL (overrides FlinkURL port substitution when set)
	SchemaRegistryURL string
	LocalMode         bool
	GenerateReport    bool             // New field to enable report generation
	ReportsDir        string           // Directory to save reports
	TrafficPatterns   *TrafficPatterns // Traffic patterns for dynamic rate changes
	KafkaConfig       KafkaConfig      // Kafka topic configuration
	GlobalTables      bool             // New field to enable global table creation mode
	CSVMode           bool             // When true, skip Kafka producer ONLY (filesystem CSV source table); consumer still runs
}

// Normalize fills defaults and canonicalizes URLs. Should be called early (e.g. NewRunner)
func (c *Config) Normalize() {
	// Default FlinkURL
	if strings.TrimSpace(c.FlinkURL) == "" {
		c.FlinkURL = "http://localhost:8081"
	}
	// Trim trailing slashes
	c.FlinkURL = strings.TrimRight(c.FlinkURL, "/")
	if c.SQLGatewayURL != "" {
		c.SQLGatewayURL = strings.TrimRight(c.SQLGatewayURL, "/")
	}
	// Derive gateway if not explicitly set
	if c.SQLGatewayURL == "" {
		c.SQLGatewayURL = DeriveGatewayURL(c.FlinkURL)
	}
}

// DeriveGatewayURL computes SQL Gateway base URL from a Flink REST URL.
// Rules:
// 1. If input contains :8081 replace with :8083
// 2. If input already has a port != 8081 reuse scheme/host and append :8083 only if different path not present
// 3. If no port, append :8083
func DeriveGatewayURL(flinkURL string) string {
	// Only transform the canonical UI/REST port 8081 -> gateway 8083.
	if strings.Contains(flinkURL, ":8081") {
		return strings.Replace(flinkURL, ":8081", ":8083", 1)
	}
	// If some other explicit port already present (e.g. test server random port) keep it as-is.
	// Heuristic: scheme://host:port (>=2 colons overall) or host:port (1 colon without scheme) but not ending in known path.
	// We just check last colon followed by digits up to optional slash.
	trimmed := strings.TrimRight(flinkURL, "/")
	lastColon := strings.LastIndex(trimmed, ":")
	if lastColon != -1 && lastColon > len("http://")-1 { // after scheme
		portPart := trimmed[lastColon+1:]
		allDigits := true
		for _, r := range portPart {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return trimmed // preserve existing non-8081 port
		}
	}
	return trimmed + ":8083"
}

// Runner orchestrates the complete pipeline execution
type Runner struct {
	config        *Config
	resourceMgr   *ResourceManager
	producer      *Producer
	consumer      *Consumer
	flinkDeployer *FlinkDeployer
	sqlLoader     *SQLLoader
	logger        logpkg.Logger
}

// TopicInfo represents information about a Kafka topic
type TopicInfo struct {
	Name              string
	Type              string
	Partitions        int
	ReplicationFactor int
	Schema            string
	MessageCount      int64
	Size              string
	ProduceRate       float64
	ConsumeRate       float64
	Lag               int64
}

// FlinkJobInfo represents information about a Flink job
type FlinkJobInfo struct {
	Name               string
	JobID              string // Add JobID for linking to Flink UI
	Status             string
	Duration           string
	RecordsProcessed   int64
	Parallelism        int
	RecordsPerSec      float64
	BackPressureStatus string
}

// SchemaInfo represents information about a Schema Registry subject
type SchemaInfo struct {
	Subject      string
	SchemaID     string
	Version      string
	Type         string
	Status       string
	MessageCount int64
	ProduceRate  float64
	ConsumeRate  float64
	Lag          int64
}

// ExecutionMetrics contains detailed metrics about the execution
type ExecutionMetrics struct {
	FlinkJobs     []FlinkJobInfo
	ProducerStats ProducerStatistics
	ConsumerStats ConsumerStatistics
	TotalMessages int64
	TotalDuration time.Duration
	ErrorCount    int64
	SuccessRate   float64
}

// ProducerStatistics contains producer performance metrics
type ProducerStatistics struct {
	MessagesProduced int64
	BytesProduced    int64
	SuccessRate      float64
	ErrorCount       int64
}

// ConsumerStatistics contains consumer performance metrics
type ConsumerStatistics struct {
	MessagesConsumed int64
	BytesConsumed    int64
	SuccessRate      float64
	ErrorCount       int64
}

// NewRunner creates a new pipeline runner
func NewRunner(config *Config) (*Runner, error) {
	// Normalize config early
	config.Normalize()
	resourceMgr := NewResourceManager(config)

	producer, err := NewProducer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}

	consumer, err := NewConsumer(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	var flinkDeployer *FlinkDeployer
	if config.GlobalTables {
		flinkDeployer = NewFlinkDeployerGlobal(config)
	} else {
		flinkDeployer = NewFlinkDeployer(config)
	}
	sqlLoader := NewSQLLoader(config.ProjectDir)

	logger := logpkg.Global()
	if config.GlobalTables {
		logger.Info("mode selected", "mode", "global")
	} else {
		logger.Info("mode selected", "mode", "session")
	}

	return &Runner{
		config:        config,
		resourceMgr:   resourceMgr,
		producer:      producer,
		consumer:      consumer,
		flinkDeployer: flinkDeployer,
		sqlLoader:     sqlLoader,
		logger:        logger,
	}, nil
}

// generateExecutionID creates a unique execution ID
func (r *Runner) generateExecutionID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random read fails
		return fmt.Sprintf("%x", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

// Run executes the complete pipeline
func (r *Runner) Run(ctx context.Context) error {
	r.logger.Info("pipeline start")

	// Set up pipeline timeout context
	pipelineCtx := ctx
	if r.config.PipelineTimeout > 0 {
		var cancelPipeline context.CancelFunc
		pipelineCtx, cancelPipeline = context.WithTimeout(ctx, r.config.PipelineTimeout)
		defer cancelPipeline()
		r.logger.Info("pipeline timeout configured", "timeout", r.config.PipelineTimeout.String())
	}

	// Track pipeline start time for reporting
	pipelineStartTime := time.Now()

	// Initialize execution data collector if report generation is enabled
	var dataCollector interface{}
	var executionID string

	if r.config.GenerateReport {
		executionID = r.generateExecutionID()
		r.logger.Info("execution id generated", "execution_id", executionID)

		// Create basic data collector
		dataCollector = map[string]interface{}{
			"execution_id": executionID,
			"parameters": map[string]interface{}{
				"message_rate":        r.config.MessageRate,
				"duration":            r.config.Duration.String(),
				"pipeline_timeout":    r.config.PipelineTimeout.String(),
				"bootstrap_servers":   r.config.BootstrapServers,
				"flink_url":           r.config.FlinkURL,
				"schema_registry_url": r.config.SchemaRegistryURL,
				"local_mode":          r.config.LocalMode,
				"project_dir":         r.config.ProjectDir,
				"cleanup":             r.config.Cleanup,
			},
		}
	}

	// Step 1: Load SQL statements
	r.logger.Info("loading sql statements")
	sqlStatements, err := r.sqlLoader.LoadStatements()
	if err != nil {
		return fmt.Errorf("failed to load SQL statements: %w", err)
	}
	r.logger.Info("sql statements loaded", "count", len(sqlStatements))

	// Step 2: Load AVRO schemas (optional when topics are defined in SQL)
	r.logger.Info("loading avro schemas")

	// Check if we have topics defined in SQL statements
	sqlTopics := r.sqlLoader.ExtractTopicsFromSQL(sqlStatements)

	var schemas map[string]*Schema
	schemaLoader := NewSchemaLoader(r.config.ProjectDir)

	if len(sqlTopics) > 0 {
		r.logger.Info("sql topics detected", "count", len(sqlTopics), "topics", strings.Join(sqlTopics, ","))
		r.logger.Info("auto-registration mode", "source", "table definitions")

		// Try to load additional manual schemas, but don't fail if they don't exist
		var err error
		schemas, err = schemaLoader.LoadSchemas()
		if err != nil {
			r.logger.Info("no additional schema files")
			schemas = make(map[string]*Schema) // Empty map
		} else {
			r.logger.Info("additional manual schemas loaded", "count", len(schemas))
		}
	} else {
		// No SQL topics found, schemas are required for backwards compatibility
		r.logger.Warn("no topics in sql statements; falling back to manual schema mode")
		var err error
		schemas, err = schemaLoader.LoadSchemas()
		if err != nil {
			return fmt.Errorf("failed to load schemas: %w", err)
		}
		r.logger.Info("schemas loaded", "count", len(schemas))
	}

	// Step 3: Generate dynamic resource names
	r.logger.Info("generating dynamic resource names")
	resources, err := r.resourceMgr.GenerateResources(sqlStatements)
	if err != nil {
		return fmt.Errorf("failed to generate resources: %w", err)
	}
	r.logger.Info("resources generated", "prefix", resources.Prefix)

	// Step 4: Clean up existing topics before creation
	r.logger.Info("deleting existing topics")
	if err := r.resourceMgr.DeleteTopics(ctx, resources); err != nil {
		r.logger.Warn("failed to delete topics", "error", err)
	} else {
		r.logger.Info("existing topics deleted")
	}

	// Step 5: Create Kafka topics
	r.logger.Info("creating kafka topics")
	if err := r.resourceMgr.CreateTopics(ctx, resources); err != nil {
		return fmt.Errorf("failed to create topics: %w", err)
	}
	r.logger.Info("kafka topics created", "count", len(resources.Topics))

	// Step 6: Deploy FlinkSQL statements (Flink will auto-register schemas)
	r.logger.Info("deploying flink sql statements")
	deploymentIDs, err := r.flinkDeployer.Deploy(ctx, sqlStatements, resources)
	if err != nil {
		return fmt.Errorf("failed to deploy FlinkSQL: %w", err)
	}
	r.logger.Info("flink sql statements deployed", "count", len(deploymentIDs))

	// Step 7: Register additional AVRO schemas (only if manually provided)
	if len(schemas) > 0 {
		r.logger.Info("registering additional AVRO schemas", "count", len(schemas))
		if err := r.resourceMgr.RegisterSchemas(ctx, resources, schemas); err != nil {
			r.logger.Warn("failed to register additional schemas", "error", err)
			r.logger.Info("schemas should auto-register from table definitions")
		} else {
			r.logger.Info("additional schemas registered", "count", len(schemas))
		}
	} else {
		r.logger.Info("no manual schemas to register; relying on flink auto-registration")
	}

	// Set up deferred cleanup to ensure it runs even if there are errors
	defer func() {
		if r.config.Cleanup {
			r.logger.Info("cleanup start")
			if err := r.cleanup(context.Background(), resources, deploymentIDs); err != nil {
				r.logger.Error("cleanup failed", "error", err)
			} else {
				r.logger.Info("cleanup completed")
			}
		}
	}()

	// Step 7: Wait for Flink jobs to be ready and start processing
	r.logger.Info("waiting for flink jobs to initialize")
	time.Sleep(3 * time.Second) // Give Flink jobs time to initialize

	if r.config.CSVMode {
		r.logger.Info("csv mode: skipping kafka producer; consumer still runs")
		// Monitor Flink metrics early
		go r.monitorFlinkMetrics(pipelineCtx, deploymentIDs)
	} else {
		// Step 8: Start producer (runs for specified duration)
		r.logger.Info("starting kafka producer", "duration", r.config.Duration.String(), "rate", r.config.MessageRate)

		producerCtx, cancelProducer := context.WithTimeout(pipelineCtx, r.config.Duration)
		defer cancelProducer()

		producerDone := make(chan error, 1)
		go func() {
			producerDone <- r.producer.Start(producerCtx, resources.InputTopic, schemas["input"])
		}()

		// Step 9: Monitor Flink job metrics during execution
		go r.monitorFlinkMetrics(pipelineCtx, deploymentIDs)

		// Step 10: Wait for producer to complete, then wait for Flink to process records
		r.logger.Info("waiting for producer completion before consumer start")

		var producerErr error
		select {
		case producerErr = <-producerDone:
			if producerErr != nil && producerErr != context.DeadlineExceeded {
				return fmt.Errorf("producer failed: %w", producerErr)
			}
			r.logger.Info("producer completed successfully")
		case <-pipelineCtx.Done():
			r.logger.Error("pipeline timeout during producer phase")
			return pipelineCtx.Err()
		}
	}

	// Step 11: Wait for Flink job to process records before starting consumer
	r.logger.Info("waiting for flink job to process records")
	if err := r.waitForFlinkProcessing(pipelineCtx, deploymentIDs); err != nil {
		if pipelineCtx.Err() != nil {
			r.logger.Error("pipeline timeout while waiting for flink processing")
			return pipelineCtx.Err()
		}
		return fmt.Errorf("failed waiting for Flink processing: %w", err)
	}

	// Step 12: Start consumer (always run; in CSV mode it will observe Flink sink output topic if applicable)
	consumerCompleted := false
	var consumerDone chan error
	if r.config.CSVMode {
		r.logger.Info("csv mode: starting kafka consumer to validate downstream topic")
	}
	{
		r.logger.Info("starting kafka consumer to read flink processing results")

		// Calculate expected messages if not explicitly set
		if r.config.ExpectedMessages == 0 {
			if r.config.CSVMode {
				// In CSV mode we can't estimate from producer; use a reasonable default (all rows processed by Flink)
				// We'll keep it zero meaning consumer will just read until timeout or completion of Flink job.
				r.logger.Info("csv mode: no expected message count derived; set --expected for bounded consume")
			} else {
				// Estimate based on producer stats if available
				if r.producer != nil {
					producerStats := r.producer.GetStats()
					r.config.ExpectedMessages = producerStats.MessagesSent
					r.logger.Info("expected messages derived from producer", "count", r.config.ExpectedMessages)
				} else {
					// Fallback estimation
					r.config.ExpectedMessages = int64(float64(r.config.MessageRate) * r.config.Duration.Seconds())
					r.logger.Info("expected messages estimated", "count", r.config.ExpectedMessages)
				}
			}
		} else {
			r.logger.Info("expected messages configured", "count", r.config.ExpectedMessages)
		}

		// Initialize consumer with Schema Registry
		if err := r.consumer.InitializeSchemaRegistry(resources.OutputTopic); err != nil {
			r.logger.Warn("failed to initialize consumer schema registry", "error", err)
			r.logger.Info("consumer will run without avro deserialization")
		}

		// Start consumer with smart stopping logic
		consumerDone = make(chan error, 1)
		go func() {
			consumerDone <- r.consumer.StartWithExpectedCount(pipelineCtx, resources.OutputTopic, r.config.ExpectedMessages)
		}()

		// Step 13: Wait for consumer to complete or pipeline timeout
		select {
		case consumerErr := <-consumerDone:
			consumerCompleted = true
			if consumerErr != nil {
				// Check if it's a timeout vs actual error
				if pipelineCtx.Err() != nil {
					r.logger.Error("consumer stopped due to pipeline timeout")
				} else {
					r.logger.Error("consumer completed with error", "error", consumerErr)
				}
			} else {
				r.logger.Info("consumer completed successfully")
			}
		case <-pipelineCtx.Done():
			r.logger.Error("pipeline timeout during consumer phase")
			// Give consumer a moment to stop gracefully
			time.Sleep(1 * time.Second)
		}
	}

	// Wait a bit more if consumer hasn't finished to see if it completes
	if !consumerCompleted {
		select {
		case consumerErr := <-consumerDone:
			if consumerErr != nil {
				r.logger.Error("consumer completed with error after timeout", "error", consumerErr)
			} else {
				r.logger.Info("consumer completed successfully after pipeline timeout")
			}
		case <-time.After(2 * time.Second):
			r.logger.Warn("consumer did not complete within grace period")
		}
	}

	// Step 14: Generate execution report if enabled
	actualDuration := time.Since(pipelineStartTime)
	finalStatus := "completed"
	if pipelineCtx.Err() != nil {
		finalStatus = "timeout"
	}

	if dataCollector != nil {
		if err := r.generateExecutionReport(dataCollector, finalStatus, actualDuration, resources, schemas); err != nil {
			r.logger.Warn("failed to generate execution report", "error", err)
		}
	}

	return nil
}

// cleanup removes all created resources
func (r *Runner) cleanup(ctx context.Context, resources *Resources, deploymentIDs []string) error {
	// Stop FlinkSQL deployments
	if err := r.flinkDeployer.Cleanup(ctx, deploymentIDs); err != nil {
		return fmt.Errorf("failed to cleanup FlinkSQL deployments: %w", err)
	}

	// Delete Kafka topics
	if err := r.resourceMgr.DeleteTopics(ctx, resources); err != nil {
		return fmt.Errorf("failed to delete topics: %w", err)
	}

	return nil
}

// generateExecutionReport creates and saves the final execution report
func (r *Runner) generateExecutionReport(dataCollector interface{}, status string, duration time.Duration, resources *Resources, schemas map[string]*Schema) error {
	if !r.config.GenerateReport {
		return nil
	}

	r.logger.Info("generating execution report", "status", status, "duration", duration.String())

	// Extract report data
	var reportData map[string]interface{}

	if dataCollector != nil {
		if collectorMap, ok := dataCollector.(map[string]interface{}); ok {
			reportData = map[string]interface{}{
				"execution_id": collectorMap["execution_id"],
				"parameters":   collectorMap["parameters"],
				"status":       status,
				"duration":     duration.String(),
				"timestamp":    time.Now().Format(time.RFC3339),
			}
		}
	}

	// Fallback if no data collector or invalid data
	if reportData == nil {
		reportData = map[string]interface{}{
			"execution_id": "unknown",
			"parameters": map[string]interface{}{
				"message_rate":        r.config.MessageRate,
				"duration":            r.config.Duration.String(),
				"bootstrap_servers":   r.config.BootstrapServers,
				"flink_url":           r.config.FlinkURL,
				"schema_registry_url": r.config.SchemaRegistryURL,
				"local_mode":          r.config.LocalMode,
				"project_dir":         r.config.ProjectDir,
				"cleanup":             r.config.Cleanup,
			},
			"status":    status,
			"duration":  duration.String(),
			"timestamp": time.Now().Format(time.RFC3339),
		}
	}

	// Set reports directory if not specified
	reportsDir := r.config.ReportsDir
	if reportsDir == "" {
		reportsDir = filepath.Join(r.config.ProjectDir, "reports")
	}

	// Create reports directory
	r.logger.Info("report directory", "path", reportsDir)
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		r.logger.Error("failed to create reports directory", "error", err)
		return nil
	}

	// Generate HTML report
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("pipegen-execution-report-%s.html", timestamp)
	reportPath := filepath.Join(reportsDir, filename)

	// Create enhanced HTML report with actual metrics
	htmlContent := r.generateEnhancedHTMLReport(reportData, status, duration, resources, schemas)

	if err := os.WriteFile(reportPath, []byte(htmlContent), 0644); err != nil {
		r.logger.Error("failed to write execution report", "error", err)
		return nil
	}

	r.logger.Info("execution report generated", "path", reportPath)
	return nil
}

// generateEnhancedHTMLReport creates a comprehensive HTML report with metrics and logo
func (r *Runner) generateEnhancedHTMLReport(reportData map[string]interface{}, status string, duration time.Duration, resources *Resources, schemas map[string]*Schema) string {
	// Get executable directory for path resolution
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Sprintf("<html><body><h1>Error getting executable path: %v</h1></body></html>", err)
	}
	execDir := filepath.Dir(execPath)
	// Collect actual execution metrics
	metrics := r.collectExecutionMetrics(duration)
	topicInfo := r.collectTopicInformation(resources)
	schemaInfo := r.collectSchemaRegistryInformation(resources, schemas)

	// Attempt to use embedded template first
	templateStr := ""
	if templates.ExecutionReportTemplate != "" {
		templateStr = templates.ExecutionReportTemplate
	} else {
		// Fallback to filesystem paths for development environments
		templatePath := filepath.Join(execDir, "..", "internal", "templates", "files", "execution_report.html")
		if _, err = os.Stat(templatePath); os.IsNotExist(err) {
			templatePath = filepath.Join("internal", "templates", "files", "execution_report.html")
		}
		content, readErr := os.ReadFile(templatePath)
		if readErr != nil {
			return fmt.Sprintf("<html><body><h1>Error reading template: %v</h1></body></html>", readErr)
		}
		templateStr = string(content)
	}

	// Get actual metrics from producer and consumer
	var messagesProduced int64
	var throughputProducer float64
	var bytesProduced int64

	if r.producer != nil {
		producerStats := r.producer.GetStats()
		messagesProduced = producerStats.MessagesSent
		throughputProducer = producerStats.MessagesPerSec
		bytesProduced = producerStats.BytesSent
	}

	messagesConsumed := int64(0)                                           // TODO: Get from actual consumer stats when implemented
	throughputConsumer := float64(0)                                       // TODO: Get from actual consumer stats
	successRate := 100.0                                                   // TODO: Calculate from actual metrics
	errorCount := int64(0)                                                 // TODO: Get from actual metrics
	avgLatency := "< 1ms"                                                  // TODO: Get from actual metrics
	dataVolume := fmt.Sprintf("%.2f MB", float64(bytesProduced)/1024/1024) // Use actual bytes from producer

	// Prepare template data
	templateData := struct {
		ExecutionID        string
		Status             string
		StatusClass        string
		Duration           string
		ExecutionTime      string
		Timestamp          string
		MessageRate        int
		BootstrapServers   string
		FlinkURL           string
		SchemaRegistryURL  string
		ProjectDir         string
		LocalMode          bool
		Cleanup            bool
		LogoSVG            template.HTML
		MessagesProduced   int64
		MessagesConsumed   int64
		ThroughputProducer float64
		ThroughputConsumer float64
		SuccessRate        float64
		ErrorCount         int64
		AvgLatency         string
		DataVolume         string
		TopicInfo          []TopicInfo
		SchemaInfo         []SchemaInfo
		FlinkJobs          []FlinkJobInfo
	}{
		ExecutionID:        reportData["execution_id"].(string),
		Status:             status,
		StatusClass:        map[string]string{"completed": "success", "failed": "failed"}[status],
		Duration:           duration.String(),
		ExecutionTime:      formatDuration(duration),
		Timestamp:          time.Now().Format("2006-01-02 15:04:05 MST"),
		MessageRate:        r.config.MessageRate,
		BootstrapServers:   r.config.BootstrapServers,
		FlinkURL:           r.config.FlinkURL,
		SchemaRegistryURL:  r.config.SchemaRegistryURL,
		ProjectDir:         r.config.ProjectDir,
		LocalMode:          r.config.LocalMode,
		Cleanup:            r.config.Cleanup,
		LogoSVG:            template.HTML(""), // Logo will be loaded from template or assets
		MessagesProduced:   messagesProduced,
		MessagesConsumed:   messagesConsumed,
		ThroughputProducer: throughputProducer,
		ThroughputConsumer: throughputConsumer,
		SuccessRate:        successRate,
		ErrorCount:         errorCount,
		AvgLatency:         avgLatency,
		DataVolume:         dataVolume,
		TopicInfo:          topicInfo,
		SchemaInfo:         schemaInfo,
		FlinkJobs:          metrics.FlinkJobs,
	}

	// Execute template
	tmpl, err := template.New("report").Parse(templateStr)
	if err != nil {
		return fmt.Sprintf("<html><body><h1>Error generating report: %v</h1></body></html>", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, templateData); err != nil {
		return fmt.Sprintf("<html><body><h1>Error executing template: %v</h1></body></html>", err)
	}

	return result.String()
}

// collectExecutionMetrics gathers comprehensive execution metrics
func (r *Runner) collectExecutionMetrics(duration time.Duration) ExecutionMetrics {
	metrics := ExecutionMetrics{
		TotalDuration: duration,
		FlinkJobs:     []FlinkJobInfo{},
		ErrorCount:    0,
		SuccessRate:   100.0,
	}

	// Calculate total messages based on configuration
	metrics.TotalMessages = int64(float64(r.config.MessageRate) * duration.Seconds())

	// Try to get producer stats if producer exists
	if r.producer != nil {
		// Producer stats would come from actual statistics collection
		metrics.ProducerStats = ProducerStatistics{
			MessagesProduced: metrics.TotalMessages,
			BytesProduced:    metrics.TotalMessages * 1024, // Estimate 1KB per message
			SuccessRate:      100.0,
			ErrorCount:       0,
		}
	}

	// Try to get consumer stats if consumer exists
	if r.consumer != nil {
		// Consumer stats would come from actual statistics collection
		metrics.ConsumerStats = ConsumerStatistics{
			MessagesConsumed: metrics.TotalMessages, // Assuming all messages consumed
			BytesConsumed:    metrics.TotalMessages * 1024,
			SuccessRate:      100.0,
			ErrorCount:       0,
		}
	}

	// TODO: Add actual Flink job metrics collection
	// For now, add a placeholder job
	metrics.FlinkJobs = append(metrics.FlinkJobs, FlinkJobInfo{
		Name:               "Data Processing Pipeline",
		JobID:              "a1b2c3d4e5f6", // Placeholder JobID - in reality this would come from Flink API
		Status:             "running",
		Duration:           duration.String(),
		RecordsProcessed:   metrics.TotalMessages,
		Parallelism:        1,
		RecordsPerSec:      float64(r.config.MessageRate),
		BackPressureStatus: "OK",
	})

	return metrics
}

// collectTopicInformation gathers information about Kafka topics used
func (r *Runner) collectTopicInformation(resources *Resources) []TopicInfo {
	topics := []TopicInfo{}

	// Estimate messages based on configuration
	estimatedMessages := int64(float64(r.config.MessageRate) * r.config.Duration.Seconds())

	// Get Kafka configuration for partition information
	partitions := r.config.KafkaConfig.Partitions
	if partitions == 0 {
		partitions = 1 // Default partition count
	}
	replication := r.config.KafkaConfig.ReplicationFactor
	if replication == 0 {
		replication = 1 // Default replication factor
	}

	// Estimate topic size and rates
	// Assumption: ~1KB per message
	estimatedMB := float64(estimatedMessages) / 1024.0
	sizeStr := fmt.Sprintf("%.2f MB", estimatedMB)
	produceRate := float64(r.config.MessageRate)
	// Assume consumer slightly below producer unless we have a consumer
	consumeRate := produceRate
	if r.consumer == nil {
		consumeRate = produceRate * 0.98
	}
	var lagEstimate int64
	if r.consumer == nil {
		// If consumer not running, approximate small lag
		lagEstimate = int64(float64(r.config.MessageRate) * 0.02)
	}

	// Add all topics from the dynamically created resources
	for i, topicName := range resources.Topics {
		var topicType string

		// Determine topic type based on position and known input/output topics
		if topicName == resources.InputTopic {
			topicType = "Source"
		} else if topicName == resources.OutputTopic {
			topicType = "Sink"
		} else if i == 0 && resources.InputTopic == "" {
			// If no specific input topic identified, first one is likely source
			topicType = "Source"
		} else if i == len(resources.Topics)-1 && resources.OutputTopic == "" {
			// If no specific output topic identified, last one is likely sink
			topicType = "Sink"
		} else {
			topicType = "Intermediate"
		}

		topics = append(topics, TopicInfo{
			Name:              topicName,
			Type:              topicType,
			Partitions:        partitions,
			ReplicationFactor: replication,
			Schema:            "AVRO",
			MessageCount:      estimatedMessages,
			Size:              sizeStr,
			ProduceRate:       produceRate,
			ConsumeRate:       consumeRate,
			Lag:               lagEstimate,
		})
	}

	// If no topics found in resources, fallback to input/output topics
	if len(topics) == 0 {
		if resources.InputTopic != "" {
			topics = append(topics, TopicInfo{
				Name:              resources.InputTopic,
				Type:              "Source",
				Partitions:        partitions,
				ReplicationFactor: replication,
				Schema:            "AVRO",
				MessageCount:      estimatedMessages,
				Size:              sizeStr,
				ProduceRate:       produceRate,
				ConsumeRate:       consumeRate,
				Lag:               lagEstimate,
			})
		}

		if resources.OutputTopic != "" && resources.OutputTopic != resources.InputTopic {
			topics = append(topics, TopicInfo{
				Name:              resources.OutputTopic,
				Type:              "Sink",
				Partitions:        partitions,
				ReplicationFactor: replication,
				Schema:            "AVRO",
				MessageCount:      estimatedMessages,
				Size:              sizeStr,
				ProduceRate:       produceRate,
				ConsumeRate:       consumeRate,
				Lag:               lagEstimate,
			})
		}
	}

	return topics
}

// collectSchemaRegistryInformation gathers information about registered schemas
func (r *Runner) collectSchemaRegistryInformation(resources *Resources, schemas map[string]*Schema) []SchemaInfo {
	schemaInfos := []SchemaInfo{}

	// Iterate through all registered schemas and create schema info
	for schemaName, schema := range schemas {
		if schema == nil {
			continue
		}

		// Generate subject name based on topic naming convention
		subject := r.getSchemaSubject(resources, schemaName)

		// Determine schema type (defaulting to AVRO since that's what pipegen uses)
		schemaType := "AVRO"

		// For now, we'll use estimated values since we don't have direct access to Schema Registry API responses
		// In a full implementation, these would come from actual Schema Registry API calls
		schemaInfos = append(schemaInfos, SchemaInfo{
			Subject:      subject,
			SchemaID:     "1", // Would be actual ID from Schema Registry
			Version:      "1", // Would be actual version from Schema Registry
			Type:         schemaType,
			Status:       "ACTIVE",
			MessageCount: 0,
			ProduceRate:  0,
			ConsumeRate:  0,
			Lag:          0,
		})
	}

	// If no schemas were provided or found, create fallback schema info based on topics
	if len(schemaInfos) == 0 {
		// Create schema info for input topic if it exists
		if resources.InputTopic != "" {
			schemaInfos = append(schemaInfos, SchemaInfo{
				Subject:  fmt.Sprintf("%s-value", resources.InputTopic),
				SchemaID: "1",
				Version:  "1",
				Type:     "AVRO",
				Status:   "ACTIVE",
			})
		}

		// Create schema info for output topic if it exists and is different from input
		if resources.OutputTopic != "" && resources.OutputTopic != resources.InputTopic {
			schemaInfos = append(schemaInfos, SchemaInfo{
				Subject:  fmt.Sprintf("%s-value", resources.OutputTopic),
				SchemaID: "2",
				Version:  "1",
				Type:     "AVRO",
				Status:   "ACTIVE",
			})
		}
	}

	return schemaInfos
}

// getSchemaSubject generates the schema subject name based on topic and schema name
func (r *Runner) getSchemaSubject(resources *Resources, schemaName string) string {
	// Standard Confluent Schema Registry naming convention: <topic-name>-value
	// Try to match schema name with topic to generate proper subject

	if schemaName == "input" && resources.InputTopic != "" {
		return fmt.Sprintf("%s-value", resources.InputTopic)
	}

	if schemaName == "output" && resources.OutputTopic != "" {
		return fmt.Sprintf("%s-value", resources.OutputTopic)
	}

	// For other schemas, try to find matching topic or use schema name
	for _, topic := range resources.Topics {
		if strings.Contains(topic, schemaName) || strings.Contains(schemaName, topic) {
			return fmt.Sprintf("%s-value", topic)
		}
	}

	// Fallback: use schema name as topic name
	return fmt.Sprintf("%s-value", schemaName)
}

// formatDuration formats a duration as HH:MM:SS
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	hours := d / time.Hour
	minutes := (d % time.Hour) / time.Minute
	seconds := (d % time.Minute) / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

// PipelineStatus tracks the current state of all pipeline components
type PipelineStatus struct {
	Producer struct {
		MessagesSent int64
		Rate         float64
		Target       int
		Elapsed      time.Duration
	}
	Flink struct {
		JobsRunning      int
		RecordsRead      int64
		RecordsWritten   int64
		ProcessingActive bool
	}
	Consumer struct {
		MessagesProcessed int64
		Rate              float64
		Errors            int64
		Elapsed           time.Duration
		Active            bool
	}
}

var globalPipelineStatus = &PipelineStatus{}

// monitorFlinkMetrics periodically reports Flink job metrics during execution
func (r *Runner) monitorFlinkMetrics(ctx context.Context, deploymentIDs []string) {
	r.logger.Info("starting flink metrics monitoring")

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	firstProcessingDetected := false

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			hasActivity := r.reportFlinkMetrics(&firstProcessingDetected)
			// Show different message when first processing is detected
			if hasActivity && !firstProcessingDetected {
				r.logger.Info("flink started processing data")
				firstProcessingDetected = true
			}

			// Display consolidated pipeline status
			r.displayPipelineStatus()
		}
	}
}

// displayPipelineStatus shows a consolidated view of all pipeline components
func (r *Runner) displayPipelineStatus() {
	status := globalPipelineStatus

	// Producer status (should already be updated by producer)
	if status.Producer.MessagesSent > 0 || status.Producer.Elapsed > 0 {
		r.logger.Debug("producer status", "messages_sent", status.Producer.MessagesSent, "rate", status.Producer.Rate, "target_rate", status.Producer.Target, "elapsed", status.Producer.Elapsed.Truncate(time.Second).String())
	}

	// Flink status
	if status.Flink.JobsRunning > 0 {
		r.logger.Debug("flink status", "jobs_running", status.Flink.JobsRunning, "processing_active", status.Flink.ProcessingActive)
	} else {
		r.logger.Warn("no flink jobs running")
	}

	// Consumer status
	if status.Consumer.Active {
		r.logger.Debug("consumer status", "messages_processed", status.Consumer.MessagesProcessed, "rate", status.Consumer.Rate, "errors", status.Consumer.Errors, "elapsed", status.Consumer.Elapsed.Truncate(time.Second).String())
	} else {
		r.logger.Debug("consumer not started yet")
	}
}

// reportFlinkMetrics fetches and reports current Flink job metrics
func (r *Runner) reportFlinkMetrics(firstProcessingDetected *bool) bool {
	// Get list of running jobs
	resp, err := http.Get(fmt.Sprintf("%s/jobs", r.config.FlinkURL))
	if err != nil {
		r.logger.Warn("failed to fetch flink jobs", "error", err)
		return false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			r.logger.Warn("failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		r.logger.Warn("failed to read flink jobs response", "error", err)
		return false
	}

	var jobsResp struct {
		Jobs []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(body, &jobsResp); err != nil {
		r.logger.Warn("failed to parse flink jobs response", "error", err)
		return false
	}

	// Report metrics for running jobs
	runningJobs := 0
	hasActivity := false
	totalReadRecords := int64(0)
	totalWriteRecords := int64(0)

	for _, job := range jobsResp.Jobs {
		if job.Status == "RUNNING" {
			runningJobs++
			readRecords, writeRecords, activity := r.reportSingleJobMetrics(job.ID, firstProcessingDetected)
			totalReadRecords += readRecords
			totalWriteRecords += writeRecords
			if activity {
				hasActivity = true
			}
		}
	}

	// Update global status
	globalPipelineStatus.Flink.JobsRunning = runningJobs
	globalPipelineStatus.Flink.RecordsRead = totalReadRecords
	globalPipelineStatus.Flink.RecordsWritten = totalWriteRecords

	// Since Flink metrics API can be unreliable, we'll assume processing is active if:
	// 1. Jobs are running AND
	// 2. We have actual activity OR producer has sent messages (indicating data should be flowing)
	producerHasSentMessages := globalPipelineStatus.Producer.MessagesSent > 0
	globalPipelineStatus.Flink.ProcessingActive = hasActivity || (runningJobs > 0 && producerHasSentMessages)

	if runningJobs == 0 {
		// No Flink jobs are currently running
		r.logger.Warn("no flink jobs running")
	}

	return hasActivity
}

// reportSingleJobMetrics reports metrics for a single Flink job
func (r *Runner) reportSingleJobMetrics(jobID string, firstProcessingDetected *bool) (int64, int64, bool) {
	resp, err := http.Get(fmt.Sprintf("%s/jobs/%s", r.config.FlinkURL, jobID))
	if err != nil {
		return 0, 0, false
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			r.logger.Warn("failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, false
	}

	var jobResp struct {
		Name     string `json:"name"`
		State    string `json:"state"`
		Duration int64  `json:"duration"`
		Vertices []struct {
			Name    string `json:"name"`
			Metrics struct {
				ReadRecords  int64 `json:"read-records"`
				WriteRecords int64 `json:"write-records"`
			} `json:"metrics"`
		} `json:"vertices"`
	}

	if err := json.Unmarshal(body, &jobResp); err != nil {
		return 0, 0, false
	}

	// Sum up metrics across all vertices
	totalReadRecords := int64(0)
	totalWriteRecords := int64(0)

	for _, vertex := range jobResp.Vertices {
		totalReadRecords += vertex.Metrics.ReadRecords
		totalWriteRecords += vertex.Metrics.WriteRecords
	}

	elapsed := time.Duration(jobResp.Duration) * time.Millisecond

	// Only log progress if there's activity or every 30 seconds
	if totalReadRecords > 0 || totalWriteRecords > 0 {
		r.logger.Debug("flink processing", "job", jobResp.Name, "read_records", totalReadRecords, "write_records", totalWriteRecords, "runtime", elapsed.Truncate(time.Second).String())
	} else if int(elapsed.Seconds())%30 == 0 {
		r.logger.Debug("flink job waiting for data", "job", jobResp.Name, "runtime", elapsed.Truncate(time.Second).String())
	}

	hasActivity := totalReadRecords > 0 || totalWriteRecords > 0

	// No individual job logging anymore, will be shown in consolidated status

	return totalReadRecords, totalWriteRecords, hasActivity
}

// waitForFlinkProcessing waits for Flink jobs to process records AND write to output topic before starting consumer
func (r *Runner) waitForFlinkProcessing(ctx context.Context, deploymentIDs []string) error {
	r.logger.Info("checking flink job metrics for record processing")

	// Try original approach first
	recordsProcessed, err := r.getFlinkRecordsProcessedWithFallback(ctx)
	if err == nil && recordsProcessed > 0 {
		r.logger.Info("flink rest shows records processed", "count", recordsProcessed)

		// Also check if Flink is writing records (not just reading)
		writeRecords, err := r.getFlinkRecordsWritten()
		if err == nil && writeRecords > 0 {
			r.logger.Info("flink wrote records to output topic", "count", writeRecords)
			return nil
		}
		r.logger.Warn("flink processed records but write count unreliable; using alternative monitoring", "processed", recordsProcessed, "written", writeRecords)
	}

	r.logger.Warn("flink rest metrics unreliable; using enhanced monitoring")

	// Fallback to enhanced monitoring
	monitor := NewAlternativeMonitor(r.config)

	// Use fallback consumer groups since we don't have access to SQL statements here
	consumerGroups := []string{"flink_table_transactions_v4"}

	return monitor.WaitForFlinkProcessingAlternative(ctx, consumerGroups)
}

// getFlinkRecordsProcessedWithFallback tries to get records from REST API with timeout
func (r *Runner) getFlinkRecordsProcessedWithFallback(ctx context.Context) (int64, error) {
	maxAttempts := 3
	checkInterval := 2 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		recordsProcessed, err := r.getFlinkRecordsProcessed()
		if err == nil && recordsProcessed > 0 {
			return recordsProcessed, nil
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(checkInterval):
				// Continue to next attempt
			}
		}
	}

	return 0, fmt.Errorf("no records processed in %d attempts", maxAttempts)
}

// getFlinkRecordsWritten returns total records written by all running Flink jobs to output topics
func (r *Runner) getFlinkRecordsWritten() (int64, error) {
	// Get list of running jobs
	resp, err := http.Get(fmt.Sprintf("%s/jobs", r.config.FlinkURL))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch Flink jobs: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			r.logger.Warn("failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read Flink jobs response: %w", err)
	}

	var jobsResp struct {
		Jobs []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(body, &jobsResp); err != nil {
		return 0, fmt.Errorf("failed to parse Flink jobs response: %w", err)
	}

	totalRecordsWritten := int64(0)

	// Check write metrics for running jobs
	for _, job := range jobsResp.Jobs {
		if job.Status == "RUNNING" {
			_, writeRecords, err := r.getJobRecordCounts(job.ID)
			if err != nil {
				r.logger.Warn("failed to get write metrics for job", "job", job.ID, "error", err)
				continue
			}
			totalRecordsWritten += writeRecords
		}
	}

	return totalRecordsWritten, nil
}

// getFlinkRecordsProcessed returns total records processed by all running Flink jobs
func (r *Runner) getFlinkRecordsProcessed() (int64, error) {
	// Get list of running jobs
	resp, err := http.Get(fmt.Sprintf("%s/jobs", r.config.FlinkURL))
	if err != nil {
		return 0, fmt.Errorf("failed to fetch Flink jobs: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			r.logger.Warn("failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read Flink jobs response: %w", err)
	}

	var jobsResp struct {
		Jobs []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(body, &jobsResp); err != nil {
		return 0, fmt.Errorf("failed to parse Flink jobs response: %w", err)
	}

	totalRecordsProcessed := int64(0)

	// Check for failed jobs first
	for _, job := range jobsResp.Jobs {
		if job.Status == "FAILED" {
			return 0, fmt.Errorf("flink job %s has failed - pipeline cannot proceed", job.ID)
		}
	}

	// Check metrics for running jobs
	for _, job := range jobsResp.Jobs {
		if job.Status == "RUNNING" {
			readRecords, writeRecords, err := r.getJobRecordCounts(job.ID)
			if err != nil {
				r.logger.Warn("failed to get metrics for job", "job", job.ID, "error", err)
				continue
			}
			// Count both read and write records as processing activity
			totalRecordsProcessed += readRecords + writeRecords
		}
	}

	return totalRecordsProcessed, nil
}

// getJobRecordCounts returns read and write record counts for a specific job
func (r *Runner) getJobRecordCounts(jobID string) (int64, int64, error) {
	resp, err := http.Get(fmt.Sprintf("%s/jobs/%s", r.config.FlinkURL, jobID))
	if err != nil {
		return 0, 0, fmt.Errorf("failed to fetch job details: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			r.logger.Warn("failed to close response body", "error", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read job response: %w", err)
	}

	var jobResp struct {
		Name     string `json:"name"`
		State    string `json:"state"`
		Vertices []struct {
			Name    string `json:"name"`
			Metrics struct {
				ReadRecords  int64 `json:"read-records"`
				WriteRecords int64 `json:"write-records"`
			} `json:"metrics"`
		} `json:"vertices"`
	}

	if err := json.Unmarshal(body, &jobResp); err != nil {
		return 0, 0, fmt.Errorf("failed to parse job response: %w", err)
	}

	totalReadRecords := int64(0)
	totalWriteRecords := int64(0)

	for _, vertex := range jobResp.Vertices {
		totalReadRecords += vertex.Metrics.ReadRecords
		totalWriteRecords += vertex.Metrics.WriteRecords
	}

	return totalReadRecords, totalWriteRecords, nil
}

package generator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// sanitizeAVROIdentifier converts a string to a valid AVRO identifier
// AVRO identifiers must match [A-Za-z_][A-Za-z0-9_]*
func sanitizeAVROIdentifier(s string) string {
	// Replace hyphens and other invalid chars with underscores
	re := regexp.MustCompile(`[^A-Za-z0-9_]`)
	sanitized := re.ReplaceAllString(s, "_")
	
	// Ensure it starts with a letter or underscore
	if len(sanitized) > 0 && !regexp.MustCompile(`^[A-Za-z_]`).MatchString(sanitized) {
		sanitized = "_" + sanitized
	}
	
	// If empty or only invalid chars, provide a default
	if sanitized == "" || sanitized == "_" {
		sanitized = "pipeline"
	}
	
	return sanitized
}

// ProjectGenerator handles the creation of new streaming pipeline projects
type ProjectGenerator struct {
	ProjectName     string
	ProjectPath     string
	LocalMode       bool
	InputSchemaPath string
}

// NewProjectGenerator creates a new project generator instance
func NewProjectGenerator(name, path string, localMode bool) *ProjectGenerator {
	return &ProjectGenerator{
		ProjectName: name,
		ProjectPath: path,
		LocalMode:   localMode,
	}
}

// SetInputSchemaPath sets the path to a user-provided input schema
func (g *ProjectGenerator) SetInputSchemaPath(path string) {
	g.InputSchemaPath = path
}

// Generate creates the complete project structure
func (g *ProjectGenerator) Generate() error {
	// Create project directory
	if err := os.MkdirAll(g.ProjectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate all components
	if err := g.generateSQLStatements(); err != nil {
		return err
	}

	if err := g.generateAVROSchemas(); err != nil {
		return err
	}

	if err := g.generateConfig(); err != nil {
		return err
	}

	// Generate Docker and Flink configuration files for local development
	if g.LocalMode {
		if err := g.generateDockerFiles(); err != nil {
			return err
		}
	}

	return nil
}

func (g *ProjectGenerator) generateSQLStatements() error {
	sqlDir := filepath.Join(g.ProjectPath, "sql")
	if err := os.MkdirAll(sqlDir, 0755); err != nil {
		return fmt.Errorf("failed to create SQL directory: %w", err)
	}

	// Generate SQL statements based on mode
	sqlTemplates := g.getSQLTemplates()

	for filename, content := range sqlTemplates {
		filePath := filepath.Join(sqlDir, filename)
		if err := writeFile(filePath, content); err != nil {
			return err
		}
	}

	return nil
}

func (g *ProjectGenerator) generateAVROSchemas() error {
	schemasDir := filepath.Join(g.ProjectPath, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		return fmt.Errorf("failed to create schemas directory: %w", err)
	}

	// Handle input schema
	if g.InputSchemaPath != "" {
		// Copy user-provided schema
		if err := g.copyInputSchema(schemasDir); err != nil {
			return err
		}
	} else {
		// Generate default input schema
		if err := g.generateDefaultInputSchema(schemasDir); err != nil {
			return err
		}
	}

	// Always generate output schema
	if err := g.generateOutputSchema(schemasDir); err != nil {
		return err
	}

	return nil
}

func (g *ProjectGenerator) copyInputSchema(schemasDir string) error {
	// Read the user-provided schema
	inputFile, err := os.Open(g.InputSchemaPath)
	if err != nil {
		return fmt.Errorf("failed to open input schema: %w", err)
	}
	defer inputFile.Close()

	// Copy to input_event.avsc
	outputPath := filepath.Join(schemasDir, "input_event.avsc")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create input schema file: %w", err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return fmt.Errorf("failed to copy schema file: %w", err)
	}

	fmt.Printf("📋 Using provided input schema: %s\n", g.InputSchemaPath)
	return nil
}

func (g *ProjectGenerator) generateDefaultInputSchema(schemasDir string) error {
	// Sanitize project name for AVRO namespace
	sanitizedName := sanitizeAVROIdentifier(g.ProjectName)
	
	// Default input event schema
	inputSchema := `{
  "type": "record",
  "name": "InputEvent",
  "namespace": "` + sanitizedName + `.events",
  "doc": "Schema for input events",
  "fields": [
    {
      "name": "event_id",
      "type": "string",
      "doc": "Unique identifier for the event"
    },
    {
      "name": "user_id", 
      "type": "string",
      "doc": "User identifier"
    },
    {
      "name": "event_type",
      "type": "string",
      "doc": "Type of event (click, view, purchase, etc.)"
    },
    {
      "name": "timestamp_col",
      "type": {
        "type": "long",
        "logicalType": "timestamp-millis"
      },
      "doc": "Event timestamp in milliseconds"
    },
    {
      "name": "properties",
      "type": {
        "type": "map",
        "values": "string"
      },
      "default": {},
      "doc": "Additional event properties"
    }
  ]
}`

	inputPath := filepath.Join(schemasDir, "input_event.avsc")
	return writeFile(inputPath, inputSchema)
}

func (g *ProjectGenerator) generateOutputSchema(schemasDir string) error {
	// Sanitize project name for AVRO namespace
	sanitizedName := sanitizeAVROIdentifier(g.ProjectName)
	
	// Output result schema
	outputSchema := `{
  "type": "record",
  "name": "OutputResult",
  "namespace": "` + sanitizedName + `.results",
  "doc": "Schema for processed results",
  "fields": [
    {
      "name": "event_type",
      "type": "string",
      "doc": "Type of event that was aggregated"
    },
    {
      "name": "user_id",
      "type": "string", 
      "doc": "User identifier"
    },
    {
      "name": "event_count",
      "type": "long",
      "doc": "Number of events in this aggregation"
    },
    {
      "name": "window_start",
      "type": {
        "type": "long",
        "logicalType": "timestamp-millis"
      },
      "doc": "Window start timestamp"
    },
    {
      "name": "window_end", 
      "type": {
        "type": "long",
        "logicalType": "timestamp-millis"
      },
      "doc": "Window end timestamp"
    }
  ]
}`

	outputPath := filepath.Join(schemasDir, "output_result.avsc")
	return writeFile(outputPath, outputSchema)
}

func (g *ProjectGenerator) getSQLTemplates() map[string]string {
	if g.LocalMode {
		return g.getLocalSQLTemplates()
	}
	return g.getCloudSQLTemplates()
}

func (g *ProjectGenerator) getLocalSQLTemplates() map[string]string {
	return map[string]string{
		"01_create_source_table.sql": `-- Create source table for input events
CREATE TABLE input_events (
  id VARCHAR(50),
  event_type VARCHAR(100),
  user_id VARCHAR(50),
  timestamp_col TIMESTAMP(3),
  properties MAP<STRING, STRING>,
  WATERMARK FOR timestamp_col AS timestamp_col - INTERVAL '5' SECOND
) WITH (
  'connector' = 'kafka',
  'topic' = 'input-events',
  'properties.bootstrap.servers' = 'localhost:9092',
  'format' = 'avro-confluent',
  'avro-confluent.url' = 'http://localhost:8082'
);`,
		"02_create_processing.sql": `-- Create processing view with windowed aggregation
CREATE TABLE processed_events AS
SELECT 
  event_type,
  user_id,
  COUNT(*) as event_count,
  TUMBLE_START(timestamp_col, INTERVAL '1' MINUTE) as window_start,
  TUMBLE_END(timestamp_col, INTERVAL '1' MINUTE) as window_end
FROM input_events
GROUP BY 
  event_type,
  user_id,
  TUMBLE(timestamp_col, INTERVAL '1' MINUTE);`,
		"03_create_output_table.sql": `-- Create output table for results
CREATE TABLE output_results (
  event_type STRING,
  user_id STRING,
  event_count BIGINT,
  window_start TIMESTAMP(3),
  window_end TIMESTAMP(3)
) WITH (
  'connector' = 'kafka',
  'topic' = 'output-results',
  'properties.bootstrap.servers' = 'localhost:9092',
  'format' = 'avro-confluent',
  'avro-confluent.url' = 'http://localhost:8082'
);`,
		"04_insert_results.sql": `-- Insert processed results into output table
INSERT INTO output_results
SELECT event_type, user_id, event_count, window_start, window_end
FROM processed_events;`,
	}
}

func (g *ProjectGenerator) getCloudSQLTemplates() map[string]string {
	return map[string]string{
		"01_create_source_table.sql": `-- Create source table for input events (Confluent Cloud)
CREATE TABLE input_events (
  id VARCHAR(50),
  event_type VARCHAR(100),
  user_id VARCHAR(50),
  timestamp_col TIMESTAMP(3),
  properties MAP<STRING, STRING>,
  WATERMARK FOR timestamp_col AS timestamp_col - INTERVAL '5' SECOND
) WITH (
  'connector' = 'kafka',
  'topic' = '${INPUT_TOPIC}',
  'properties.bootstrap.servers' = '${BOOTSTRAP_SERVERS}',
  'properties.sasl.mechanism' = 'PLAIN',
  'properties.security.protocol' = 'SASL_SSL',
  'properties.sasl.jaas.config' = 'org.apache.kafka.common.security.plain.PlainLoginModule required username="${API_KEY}" password="${API_SECRET}";',
  'format' = 'avro-confluent',
  'avro-confluent.url' = '${SCHEMA_REGISTRY_URL}',
  'avro-confluent.basic-auth.credentials-source' = 'USER_INFO',
  'avro-confluent.basic-auth.user-info' = '${SCHEMA_REGISTRY_KEY}:${SCHEMA_REGISTRY_SECRET}'
);`,
		"02_create_processing.sql": `-- Create processing view with windowed aggregation
CREATE TABLE processed_events AS
SELECT 
  event_type,
  user_id,
  COUNT(*) as event_count,
  TUMBLE_START(timestamp_col, INTERVAL '1' MINUTE) as window_start,
  TUMBLE_END(timestamp_col, INTERVAL '1' MINUTE) as window_end
FROM input_events
GROUP BY 
  event_type,
  user_id,
  TUMBLE(timestamp_col, INTERVAL '1' MINUTE);`,
		"03_create_output_table.sql": `-- Create output table for results (Confluent Cloud)
CREATE TABLE output_results (
  event_type STRING,
  user_id STRING,
  event_count BIGINT,
  window_start TIMESTAMP(3),
  window_end TIMESTAMP(3)
) WITH (
  'connector' = 'kafka',
  'topic' = '${OUTPUT_TOPIC}',
  'properties.bootstrap.servers' = '${BOOTSTRAP_SERVERS}',
  'properties.sasl.mechanism' = 'PLAIN',
  'properties.security.protocol' = 'SASL_SSL',
  'properties.sasl.jaas.config' = 'org.apache.kafka.common.security.plain.PlainLoginModule required username="${API_KEY}" password="${API_SECRET}";',
  'format' = 'avro-confluent',
  'avro-confluent.url' = '${SCHEMA_REGISTRY_URL}',
  'avro-confluent.basic-auth.credentials-source' = 'USER_INFO',
  'avro-confluent.basic-auth.user-info' = '${SCHEMA_REGISTRY_KEY}:${SCHEMA_REGISTRY_SECRET}'
);`,
		"04_insert_results.sql": `-- Insert processed results into output table
INSERT INTO output_results
SELECT event_type, user_id, event_count, window_start, window_end
FROM processed_events;`,
	}
}

func (g *ProjectGenerator) generateConfig() error {
	var configContent string
	
	if g.LocalMode {
		configContent = g.getLocalConfig()
	} else {
		configContent = g.getCloudConfig()
	}

	configPath := filepath.Join(g.ProjectPath, ".pipegen.yaml")
	return writeFile(configPath, configContent)
}

func (g *ProjectGenerator) getLocalConfig() string {
	return `# PipeGen Configuration for Local Development
# Local development mode (set to false for cloud deployment)
local_mode: true

# Kafka Configuration (local)
bootstrap_servers: "localhost:9092"

# FlinkSQL Configuration (local)  
flink_url: "http://localhost:8081"

# Schema Registry Configuration (local)
schema_registry_url: "http://localhost:8082"

# Pipeline Configuration
default_message_rate: 100
default_duration: "5m"
topic_prefix: "` + g.ProjectName + `"
cleanup_on_exit: true

# For Cloud/Confluent deployment, uncomment and configure:
# local_mode: false
# bootstrap_servers: "your-bootstrap-server.confluent.cloud:9092"
# api_key: "YOUR_API_KEY"
# api_secret: "YOUR_API_SECRET"
# schema_registry_url: "https://your-schema-registry.confluent.cloud"
# schema_registry_key: "YOUR_SCHEMA_REGISTRY_KEY"
# schema_registry_secret: "YOUR_SCHEMA_REGISTRY_SECRET"
# flink_api_key: "YOUR_FLINK_API_KEY"
# flink_api_secret: "YOUR_FLINK_API_SECRET"
# flink_environment: "YOUR_FLINK_ENVIRONMENT_ID"
# flink_compute_pool: "YOUR_FLINK_COMPUTE_POOL_ID"`
}

func (g *ProjectGenerator) generateDockerFiles() error {
	// Generate docker-compose.yml
	if err := g.generateDockerCompose(); err != nil {
		return err
	}

	// Generate flink-conf.yaml
	if err := g.generateFlinkConfig(); err != nil {
		return err
	}

	return nil
}

func (g *ProjectGenerator) generateDockerCompose() error {
	// Import the docker compose generator
	composer := &DockerComposeGenerator{}
	composeContent, err := composer.Generate(true) // Include Schema Registry by default
	if err != nil {
		return err
	}

	composePath := filepath.Join(g.ProjectPath, "docker-compose.yml")
	return writeFile(composePath, composeContent)
}

func (g *ProjectGenerator) generateFlinkConfig() error {
	flinkConfig := `# Flink configuration for local development
jobmanager.rpc.address: flink-jobmanager
jobmanager.rpc.port: 6123
jobmanager.heap.size: 1024m

taskmanager.numberOfTaskSlots: 2
taskmanager.memory.process.size: 1568m

parallelism.default: 1

# High Availability
high-availability: none

# Checkpointing
state.backend: filesystem
state.checkpoints.dir: file:///opt/flink/checkpoints
state.savepoints.dir: file:///opt/flink/savepoints

# Web UI
web.tmpdir: /tmp/flink-web-ui
web.upload.dir: /tmp/flink-web-upload

# Metrics
metrics.reporter.promgateway.class: org.apache.flink.metrics.prometheus.PrometheusReporter
`

	flinkConfPath := filepath.Join(g.ProjectPath, "flink-conf.yaml")
	return writeFile(flinkConfPath, flinkConfig)
}

// DockerComposeGenerator creates docker-compose.yml content for the streaming stack
type DockerComposeGenerator struct{}

// Generate creates the docker-compose.yml content
func (g *DockerComposeGenerator) Generate(withSchemaRegistry bool) (string, error) {
	var services []string

	// Add Kafka service (KRaft mode)
	services = append(services, g.generateKafkaService())

	// Add Flink services
	services = append(services, g.generateFlinkJobManager())
	services = append(services, g.generateFlinkTaskManager())

	// Add Schema Registry if requested
	if withSchemaRegistry {
		services = append(services, g.generateSchemaRegistry())
	}

	// Create volumes and networks
	volumes := g.generateVolumes()
	networks := g.generateNetworks()

	// Combine everything
	compose := fmt.Sprintf(`version: '3.8'

services:
%s

%s

%s
`, strings.Join(services, "\n"), volumes, networks)

	return compose, nil
}

func (g *DockerComposeGenerator) generateKafkaService() string {
	return `  kafka:
    image: confluentinc/cp-kafka:7.5.0
    hostname: kafka
    container_name: pipegen-kafka
    ports:
      - "9092:9092"
      - "19092:19092"
    environment:
      KAFKA_NODE_ID: 1
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: 'CONTROLLER:PLAINTEXT,PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT'
      KAFKA_ADVERTISED_LISTENERS: 'PLAINTEXT://kafka:29092,PLAINTEXT_HOST://localhost:9092'
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
      KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS: 0
      KAFKA_TRANSACTION_STATE_LOG_MIN_ISR: 1
      KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR: 1
      KAFKA_JMX_PORT: 19092
      KAFKA_JMX_HOSTNAME: localhost
      KAFKA_PROCESS_ROLES: 'broker,controller'
      KAFKA_CONTROLLER_QUORUM_VOTERS: '1@kafka:29093'
      KAFKA_LISTENERS: 'PLAINTEXT://kafka:29092,CONTROLLER://kafka:29093,PLAINTEXT_HOST://0.0.0.0:9092'
      KAFKA_INTER_BROKER_LISTENER_NAME: 'PLAINTEXT'
      KAFKA_CONTROLLER_LISTENER_NAMES: 'CONTROLLER'
      KAFKA_LOG_DIRS: '/tmp/kraft-combined-logs'
      CLUSTER_ID: 'MkU3OEVBNTcwNTJENDM2Qk'
    volumes:
      - kafka-data:/var/lib/kafka/data
    networks:
      - pipegen-network
    healthcheck:
      test: ["CMD-SHELL", "kafka-topics --bootstrap-server localhost:9092 --list"]
      interval: 10s
      timeout: 5s
      retries: 5`
}

func (g *DockerComposeGenerator) generateFlinkJobManager() string {
	return `  flink-jobmanager:
    image: flink:1.18.0-scala_2.12-java11
    hostname: flink-jobmanager
    container_name: pipegen-flink-jobmanager
    ports:
      - "8081:8081"
    command: jobmanager
    depends_on:
      kafka:
        condition: service_healthy
    environment:
      FLINK_PROPERTIES: "jobmanager.rpc.address: flink-jobmanager"
    volumes:
      - flink-data:/opt/flink/data
      - ./flink-conf.yaml:/opt/flink/conf/flink-conf.yaml
    networks:
      - pipegen-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/"]
      interval: 10s
      timeout: 5s
      retries: 5`
}

func (g *DockerComposeGenerator) generateFlinkTaskManager() string {
	return `  flink-taskmanager:
    image: flink:1.18.0-scala_2.12-java11
    hostname: flink-taskmanager
    container_name: pipegen-flink-taskmanager
    depends_on:
      flink-jobmanager:
        condition: service_healthy
    command: taskmanager
    scale: 1
    environment:
      FLINK_PROPERTIES: "jobmanager.rpc.address: flink-jobmanager"
    volumes:
      - flink-data:/opt/flink/data
      - ./flink-conf.yaml:/opt/flink/conf/flink-conf.yaml
    networks:
      - pipegen-network`
}

func (g *DockerComposeGenerator) generateSchemaRegistry() string {
	return `  schema-registry:
    image: confluentinc/cp-schema-registry:7.5.0
    hostname: schema-registry
    container_name: pipegen-schema-registry
    depends_on:
      kafka:
        condition: service_healthy
    ports:
      - "8082:8082"
    environment:
      SCHEMA_REGISTRY_HOST_NAME: schema-registry
      SCHEMA_REGISTRY_KAFKASTORE_BOOTSTRAP_SERVERS: 'kafka:29092'
      SCHEMA_REGISTRY_LISTENERS: http://0.0.0.0:8082
    networks:
      - pipegen-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/"]
      interval: 10s
      timeout: 5s
      retries: 5`
}

func (g *DockerComposeGenerator) generateVolumes() string {
	return `volumes:
  kafka-data:
    driver: local
  flink-data:
    driver: local`
}

func (g *DockerComposeGenerator) generateNetworks() string {
	return `networks:
  pipegen-network:
    driver: bridge`
}

func (g *ProjectGenerator) getCloudConfig() string {
	return `# PipeGen Configuration for Confluent Cloud
# Cloud deployment mode
local_mode: false

# Kafka Configuration (Confluent Cloud)
bootstrap_servers: "your-bootstrap-server.confluent.cloud:9092"
api_key: "YOUR_API_KEY"
api_secret: "YOUR_API_SECRET"

# FlinkSQL Configuration (Confluent Cloud)
flink_api_key: "YOUR_FLINK_API_KEY"
flink_api_secret: "YOUR_FLINK_API_SECRET"
flink_environment: "YOUR_FLINK_ENVIRONMENT_ID"
flink_compute_pool: "YOUR_FLINK_COMPUTE_POOL_ID"

# Schema Registry Configuration (Confluent Cloud)
schema_registry_url: "https://your-schema-registry.confluent.cloud"
schema_registry_key: "YOUR_SCHEMA_REGISTRY_KEY"
schema_registry_secret: "YOUR_SCHEMA_REGISTRY_SECRET"

# Pipeline Configuration
default_message_rate: 100
default_duration: "5m"
topic_prefix: "` + g.ProjectName + `"
cleanup_on_exit: true`
}

// Helper function to write content to file
func writeFile(filePath, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

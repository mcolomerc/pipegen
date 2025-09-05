package pipeline

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"pipegen/internal/types"

	"github.com/google/uuid"
)

// Resources holds the dynamically generated resource names
type Resources struct {
	Prefix      string
	InputTopic  string
	OutputTopic string
	Topics      []string
}

// ResourceManager handles creation and cleanup of pipeline resources
type ResourceManager struct {
	config *Config
	Kafka  *KafkaService
}

// DeleteSchemas removes schemas from Schema Registry
func (rm *ResourceManager) DeleteSchemas(ctx context.Context, resources *Resources, schemas map[string]*Schema) error {
	fmt.Println("🗑️  Deleting schemas from Schema Registry...")
	for name := range schemas {
		subject := rm.getSchemaSubject(resources, name)
		// TODO: Implement actual Schema Registry API call to delete schema
		fmt.Printf("    🗑️  Deleting schema for subject: %s\n", subject)
		// Simulated deletion
	}
	return nil
}

// NewResourceManager creates a new resource manager
func NewResourceManager(config *Config) *ResourceManager {
	broker := config.BootstrapServers
	ks := NewKafkaService(broker)
	return &ResourceManager{
		config: config,
		Kafka:  ks,
	}
}

// GenerateResources creates resource names based on mode and SQL statements
func (rm *ResourceManager) GenerateResources(statements []*types.SQLStatement) (*Resources, error) {
	// First, try to extract topics from SQL statements
	sqlTopics := rm.extractTopicsFromSQL(statements)

	if rm.config.LocalMode {
		if len(sqlTopics) > 0 {
			// Use topics from SQL statements
			resources := &Resources{
				Prefix: "pipegen-local",
				Topics: sqlTopics,
			}

			// Set input and output topics if we can identify them
			if len(sqlTopics) >= 2 {
				resources.InputTopic = sqlTopics[0]
				resources.OutputTopic = sqlTopics[len(sqlTopics)-1]
			} else if len(sqlTopics) == 1 {
				resources.InputTopic = sqlTopics[0]
				resources.OutputTopic = sqlTopics[0] // Same topic for input/output
			}

			fmt.Printf("📋 Using topics from SQL statements: %v\n", sqlTopics)
			return resources, nil
		} else {
			// Fallback to default topics
			fmt.Println("📋 No topics found in SQL statements, using default topics")
			resources := &Resources{
				Prefix:      "pipegen-local",
				InputTopic:  "input-events",
				OutputTopic: "output-results",
				Topics:      []string{"input-events", "output-results", "processed-events"},
			}
			return resources, nil
		}
	}

	// For cloud mode, generate unique resource names to avoid conflicts
	timestamp := time.Now().Format("20060102-150405")
	shortUUID := uuid.New().String()[:8]
	prefix := fmt.Sprintf("pipegen-%s-%s", timestamp, shortUUID)

	if len(sqlTopics) > 0 {
		// Use SQL topics with cloud prefix
		var cloudTopics []string
		for _, topic := range sqlTopics {
			cloudTopics = append(cloudTopics, fmt.Sprintf("%s-%s", prefix, topic))
		}

		resources := &Resources{
			Prefix: prefix,
			Topics: cloudTopics,
		}

		if len(cloudTopics) >= 2 {
			resources.InputTopic = cloudTopics[0]
			resources.OutputTopic = cloudTopics[len(cloudTopics)-1]
		}

		return resources, nil
	}

	// Fallback to default cloud topics
	inputTopic := fmt.Sprintf("%s-input", prefix)
	outputTopic := fmt.Sprintf("%s-output", prefix)
	processedTopic := fmt.Sprintf("%s-processed", prefix)

	resources := &Resources{
		Prefix:      prefix,
		InputTopic:  inputTopic,
		OutputTopic: outputTopic,
		Topics:      []string{inputTopic, outputTopic, processedTopic},
	}

	return resources, nil
}

// extractTopicsFromSQL extracts topic names from CREATE TABLE statements
func (rm *ResourceManager) extractTopicsFromSQL(statements []*types.SQLStatement) []string {
	var topics []string
	topicSet := make(map[string]bool) // Use map to avoid duplicates

	for _, stmt := range statements {
		// Look for CREATE TABLE statements
		if strings.Contains(strings.ToUpper(stmt.Content), "CREATE TABLE") {
			topic := rm.extractTopicFromCreateTable(stmt.Content)
			if topic != "" && !topicSet[topic] {
				topics = append(topics, topic)
				topicSet[topic] = true
			}
		}
	}

	return topics
}

// extractTopicFromCreateTable extracts topic name from a CREATE TABLE statement
func (rm *ResourceManager) extractTopicFromCreateTable(sql string) string {
	// Look for 'topic' = 'value' pattern
	lines := strings.Split(sql, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "'topic'") && strings.Contains(line, "=") {
			// Use regex to extract the topic value more reliably
			re := regexp.MustCompile(`'topic'\s*=\s*'([^']+)'`)
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				return matches[1]
			}

			// Fallback to the old method if regex fails
			parts := strings.Split(line, "'topic'")
			if len(parts) > 1 {
				afterTopic := parts[1]
				if strings.Contains(afterTopic, "=") {
					valueParts := strings.Split(afterTopic, "=")
					if len(valueParts) > 1 {
						value := strings.TrimSpace(valueParts[1])
						// Find the content between single quotes
						start := strings.Index(value, "'")
						if start != -1 {
							end := strings.Index(value[start+1:], "'")
							if end != -1 {
								return value[start+1 : start+1+end]
							}
						}
					}
				}
			}
		}
	}

	return ""
}

// CreateTopics creates the required Kafka topics
func (rm *ResourceManager) CreateTopics(ctx context.Context, resources *Resources) error {
	fmt.Printf("🔧 Creating topics with prefix: %s\n", resources.Prefix)
	for _, topic := range resources.Topics {
		cfg := rm.GetDefaultTopicConfig(topic)
		err := rm.Kafka.CreateTopic(ctx, topic, cfg.Partitions, cfg.ReplicationFactor)
		if err != nil {
			return fmt.Errorf("failed to create topic %s: %w", topic, err)
		}
		fmt.Printf("  ✅ Created topic: %s\n", topic)
	}
	return nil
}

// DeleteTopics removes the created Kafka topics
func (rm *ResourceManager) DeleteTopics(ctx context.Context, resources *Resources) error {
	fmt.Printf("🗑️  Deleting topics with prefix: %s\n", resources.Prefix)
	for _, topic := range resources.Topics {
		err := rm.Kafka.DeleteTopic(ctx, topic)
		if err != nil {
			return fmt.Errorf("failed to delete topic %s: %w", topic, err)
		}
		fmt.Printf("  ✅ Deleted topic: %s\n", topic)
	}
	return nil
}

// RegisterSchemas registers AVRO schemas in Schema Registry
func (rm *ResourceManager) RegisterSchemas(ctx context.Context, resources *Resources, schemas map[string]*Schema) error {
	fmt.Println("📋 Registering schemas in Schema Registry...")

	for name, schema := range schemas {
		subject := rm.getSchemaSubject(resources, name)
		if err := rm.registerSchema(ctx, subject, schema); err != nil {
			return fmt.Errorf("failed to register schema %s: %w", name, err)
		}
		fmt.Printf("  ✅ Registered schema: %s -> %s\n", name, subject)
	}

	return nil
}

// registerSchema registers an AVRO schema in Schema Registry
func (rm *ResourceManager) registerSchema(ctx context.Context, subject string, schema *Schema) error {
	// TODO: Implement actual Schema Registry API call
	// This is a placeholder for the actual implementation

	// Simulated schema registration
	fmt.Printf("    📋 Registering schema for subject: %s\n", subject)

	// Here you would use the Confluent Schema Registry API to:
	// 1. Check if schema already exists
	// 2. Register new schema version if needed
	// 3. Return schema ID for use in producer/consumer

	return nil
}

// getSchemaSubject generates the Schema Registry subject name for a schema
func (rm *ResourceManager) getSchemaSubject(resources *Resources, schemaName string) string {
	// Map schema names to topic names for subject naming
	switch schemaName {
	case "input":
		return fmt.Sprintf("%s-value", resources.InputTopic)
	case "output":
		return fmt.Sprintf("%s-value", resources.OutputTopic)
	default:
		return fmt.Sprintf("%s-%s-value", resources.Prefix, schemaName)
	}
}

// TopicConfig holds configuration for topic creation
type TopicConfig struct {
	Name              string
	Partitions        int
	ReplicationFactor int
	Config            map[string]string
}

// GetDefaultTopicConfig returns default configuration for topics
func (rm *ResourceManager) GetDefaultTopicConfig(topicName string) *TopicConfig {
	return &TopicConfig{
		Name:              topicName,
		Partitions:        rm.config.KafkaConfig.Partitions,
		ReplicationFactor: rm.config.KafkaConfig.ReplicationFactor,
		Config: map[string]string{
			"retention.ms":     fmt.Sprintf("%d", rm.config.KafkaConfig.RetentionMs),
			"cleanup.policy":   "delete",
			"compression.type": "snappy",
		},
	}
}

package pipeline

import (
	"context"
	"fmt"
	"time"

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
}

// NewResourceManager creates a new resource manager
func NewResourceManager(config *Config) *ResourceManager {
	return &ResourceManager{
		config: config,
	}
}

// GenerateResources creates resource names based on mode
func (rm *ResourceManager) GenerateResources() (*Resources, error) {
	if rm.config.LocalMode {
		// For local mode, use fixed topic names that match deployment
		resources := &Resources{
			Prefix:      "pipegen-local",
			InputTopic:  "input-events",
			OutputTopic: "output-results",
			Topics:      []string{"input-events", "output-results"},
		}
		return resources, nil
	}

	// For cloud mode, generate unique resource names to avoid conflicts
	timestamp := time.Now().Format("20060102-150405")
	shortUUID := uuid.New().String()[:8]
	prefix := fmt.Sprintf("pipegen-%s-%s", timestamp, shortUUID)

	inputTopic := fmt.Sprintf("%s-input", prefix)
	outputTopic := fmt.Sprintf("%s-output", prefix)

	resources := &Resources{
		Prefix:      prefix,
		InputTopic:  inputTopic,
		OutputTopic: outputTopic,
		Topics:      []string{inputTopic, outputTopic},
	}

	return resources, nil
}

// CreateTopics creates the required Kafka topics
func (rm *ResourceManager) CreateTopics(ctx context.Context, resources *Resources) error {
	fmt.Printf("🔧 Creating topics with prefix: %s\n", resources.Prefix)
	
	for _, topic := range resources.Topics {
		if err := rm.createTopic(ctx, topic); err != nil {
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
		if err := rm.deleteTopic(ctx, topic); err != nil {
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

// createTopic creates a single Kafka topic using Confluent Cloud Admin API
func (rm *ResourceManager) createTopic(ctx context.Context, topicName string) error {
	// TODO: Implement actual Kafka Admin API call
	// This is a placeholder for the actual implementation
	
	// Simulated topic creation
	fmt.Printf("    📝 Creating topic: %s\n", topicName)
	
	// Here you would use the Confluent Cloud Admin API or Kafka Admin Client
	// to create the topic with appropriate configuration:
	// - Partitions: 3 (configurable)
	// - Replication factor: 3 (for production)
	// - Retention: 7 days (configurable)
	// - Cleanup policy: delete
	
	return nil
}

// deleteTopic removes a single Kafka topic
func (rm *ResourceManager) deleteTopic(ctx context.Context, topicName string) error {
	// TODO: Implement actual Kafka Admin API call
	// This is a placeholder for the actual implementation
	
	// Simulated topic deletion
	fmt.Printf("    🗑️  Deleting topic: %s\n", topicName)
	
	// Here you would use the Confluent Cloud Admin API or Kafka Admin Client
	// to delete the topic
	
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
		Partitions:        3,
		ReplicationFactor: 3,
		Config: map[string]string{
			"retention.ms":     "604800000", // 7 days
			"cleanup.policy":   "delete",
			"compression.type": "snappy",
		},
	}
}

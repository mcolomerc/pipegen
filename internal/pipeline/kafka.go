package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// KafkaService handles Kafka topic operations
// Reads broker address from config

type KafkaService struct {
	BrokerAddress string
}

// NewKafkaService creates a new KafkaService
func NewKafkaService(brokerAddress string) *KafkaService {
	return &KafkaService{BrokerAddress: brokerAddress}
}

// CreateTopic creates a Kafka topic using Kafka CLI
func (ks *KafkaService) CreateTopic(ctx context.Context, topic string, partitions, replicationFactor int) error {
	fmt.Printf("📝 Creating topic: %s (partitions=%d, replication=%d)\n", topic, partitions, replicationFactor)

	// Use docker exec to run kafka-topics.sh command
	cmd := exec.Command("docker", "exec", "broker",
		"/opt/kafka/bin/kafka-topics.sh",
		"--bootstrap-server", "broker:29092", // Use internal container address
		"--create",
		"--topic", topic,
		"--partitions", strconv.Itoa(partitions),
		"--replication-factor", strconv.Itoa(replicationFactor),
		"--if-not-exists")

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("  ❌ Failed to create topic %s: %v\nOutput: %s\n", topic, err, string(output))
		return fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	fmt.Printf("  ✅ Topic created: %s\n", topic)
	return nil
}

// DeleteTopic deletes a Kafka topic using Kafka CLI
func (ks *KafkaService) DeleteTopic(ctx context.Context, topic string) error {
	fmt.Printf("🗑️  Deleting topic: %s\n", topic)

	// Use docker exec to run kafka-topics.sh command
	cmd := exec.Command("docker", "exec", "broker",
		"/opt/kafka/bin/kafka-topics.sh",
		"--bootstrap-server", "broker:29092",
		"--delete",
		"--topic", topic)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if the error is just that the topic doesn't exist
		outputStr := string(output)
		if strings.Contains(outputStr, "does not exist") || strings.Contains(outputStr, "UnknownTopicOrPartitionException") {
			fmt.Printf("  ℹ️  Topic %s does not exist (already deleted)\n", topic)
			return nil
		}
		fmt.Printf("  ❌ Failed to delete topic %s: %v\nOutput: %s\n", topic, err, outputStr)
		return fmt.Errorf("failed to delete topic %s: %w", topic, err)
	}

	fmt.Printf("  ✅ Topic deleted: %s\n", topic)
	return nil
}

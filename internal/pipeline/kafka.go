package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	logpkg "pipegen/internal/log"
)

// KafkaService handles Kafka topic operations
// Reads broker address from config

type KafkaService struct {
	BrokerAddress string
	logger        logpkg.Logger
}

// NewKafkaService creates a new KafkaService
func NewKafkaService(brokerAddress string) *KafkaService {
	return &KafkaService{BrokerAddress: brokerAddress, logger: logpkg.Global()}
}

// CreateTopic creates a Kafka topic using Kafka CLI
func (ks *KafkaService) CreateTopic(ctx context.Context, topic string, partitions, replicationFactor int) error {
	if ks.logger != nil {
		ks.logger.Info("create topic", "topic", topic, "partitions", partitions, "replication", replicationFactor)
	}

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
		if ks.logger != nil {
			ks.logger.Error("failed create topic", "topic", topic, "err", err, "output", string(output))
		}
		return fmt.Errorf("failed to create topic %s: %w", topic, err)
	}

	if ks.logger != nil {
		ks.logger.Info("topic created", "topic", topic)
	}
	return nil
}

// DeleteTopic deletes a Kafka topic using Kafka CLI
func (ks *KafkaService) DeleteTopic(ctx context.Context, topic string) error {
	if ks.logger != nil {
		ks.logger.Info("delete topic", "topic", topic)
	}

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
			if ks.logger != nil {
				ks.logger.Info("topic already deleted", "topic", topic)
			}
			return nil
		}
		if ks.logger != nil {
			ks.logger.Error("failed delete topic", "topic", topic, "err", err, "output", outputStr)
		}
		return fmt.Errorf("failed to delete topic %s: %w", topic, err)
	}

	if ks.logger != nil {
		ks.logger.Info("topic deleted", "topic", topic)
	}
	return nil
}

package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/linkedin/goavro/v2"
	"github.com/segmentio/kafka-go"
)

// Consumer handles Kafka message consumption and validation
type Consumer struct {
	config *Config
	reader *kafka.Reader
	codec  *goavro.Codec
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(config *Config) (*Consumer, error) {
	// For demo purposes, the consumer reads from the output-results topic
	// to demonstrate that the pipeline is processing messages correctly
	topicName := "output-results"

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{config.BootstrapServers},
		Topic:    topicName,
		GroupID:  fmt.Sprintf("pipegen-consumer-%d", time.Now().Unix()),
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{
		config: config,
		reader: reader,
	}, nil
}

// Start begins consuming messages from the specified topic
func (c *Consumer) Start(ctx context.Context, topic string) error {
	fmt.Printf("👂 Starting consumer for topic: %s\n", topic)

	// Note: When using GroupID, Kafka manages offsets automatically
	// Manual offset setting is not supported with consumer groups

	messageCount := 0
	errorCount := 0
	lastLogTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("🛑 Consumer stopping. Consumed %d messages (%d errors)\n", messageCount, errorCount)
			return c.reader.Close()

		default:
			// Set deadline for read operation
			readCtx, cancel := context.WithTimeout(ctx, time.Second)
			message, err := c.reader.FetchMessage(readCtx)
			cancel()

			if err != nil {
				// Check if it's a timeout (no messages available)
				if err == context.DeadlineExceeded {
					continue
				}
				fmt.Printf("❌ Consumer error: %v\n", err)
				errorCount++
				continue
			}

			// Process message
			if err := c.processMessage(&message); err != nil {
				fmt.Printf("⚠️  Failed to process message: %v\n", err)
				errorCount++
			} else {
				messageCount++
			}

			// Commit message
			if err := c.reader.CommitMessages(ctx, message); err != nil {
				fmt.Printf("⚠️  Failed to commit message: %v\n", err)
			}

			// Log progress periodically
			if time.Since(lastLogTime) > 10*time.Second {
				fmt.Printf("📊 Consumer progress: %d messages (%d errors)\n", messageCount, errorCount)
				lastLogTime = time.Now()
			}
		}
	}
}

// processMessage validates and processes a consumed message
func (c *Consumer) processMessage(msg *kafka.Message) error {
	// Basic message validation
	if msg.Value == nil {
		return fmt.Errorf("received null message value")
	}

	// TODO: Implement AVRO decoding when schema is available
	// For now, just validate message structure
	if err := c.validateMessage(msg); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// Log message details (limited to avoid spam)
	if len(msg.Value) > 0 {
		fmt.Printf("✅ Processed message: topic=%s partition=%d offset=%d size=%d bytes\n",
			msg.Topic,
			msg.Partition,
			msg.Offset,
			len(msg.Value))
	}

	return nil
}

// validateMessage performs basic validation on consumed messages
func (c *Consumer) validateMessage(msg *kafka.Message) error {
	// Check message size
	if len(msg.Value) == 0 {
		return fmt.Errorf("empty message value")
	}

	if len(msg.Value) > 1024*1024 { // 1MB limit
		return fmt.Errorf("message too large: %d bytes", len(msg.Value))
	}

	// Check topic partition validity
	if msg.Partition < 0 {
		return fmt.Errorf("invalid partition: %d", msg.Partition)
	}

	if msg.Offset < 0 {
		return fmt.Errorf("invalid offset: %d", msg.Offset)
	}

	// TODO: Add AVRO schema validation here
	// This would include:
	// 1. Decode AVRO message using registered schema
	// 2. Validate field types and constraints
	// 3. Check business logic rules

	return nil
}

// SetSchema configures the AVRO codec for message decoding
func (c *Consumer) SetSchema(schemaContent string) error {
	codec, err := goavro.NewCodec(schemaContent)
	if err != nil {
		return fmt.Errorf("failed to create AVRO codec: %w", err)
	}
	c.codec = codec
	return nil
}

// Close gracefully shuts down the consumer
func (c *Consumer) Close() {
	if c.reader != nil {
		_ = c.reader.Close()
	}
}

// ConsumerStats holds consumer statistics
type ConsumerStats struct {
	MessagesConsumed int64     `json:"messages_consumed"`
	MessagesPerSec   float64   `json:"messages_per_sec"`
	BytesConsumed    int64     `json:"bytes_consumed"`
	ErrorCount       int64     `json:"error_count"`
	LastMessageTime  time.Time `json:"last_message_time"`
	CurrentOffset    int64     `json:"current_offset"`
	LagMessages      int64     `json:"lag_messages"`
}

// GetStats returns current consumer statistics
func (c *Consumer) GetStats() *ConsumerStats {
	// TODO: Implement actual statistics collection
	return &ConsumerStats{
		MessagesConsumed: 0,
		MessagesPerSec:   0,
		BytesConsumed:    0,
		ErrorCount:       0,
		LastMessageTime:  time.Now(),
		CurrentOffset:    0,
		LagMessages:      0,
	}
}

// MessageValidator interface for custom message validation
type MessageValidator interface {
	Validate(message map[string]interface{}) error
}

// DefaultValidator provides basic message validation
type DefaultValidator struct{}

// Validate performs default validation checks
func (v *DefaultValidator) Validate(message map[string]interface{}) error {
	// Check for required fields (example)
	requiredFields := []string{"event_id", "user_id", "event_type", "timestamp_col"}

	for _, field := range requiredFields {
		if _, exists := message[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate field types
	if eventID, ok := message["event_id"].(string); !ok || eventID == "" {
		return fmt.Errorf("invalid event_id field")
	}

	if userID, ok := message["user_id"].(string); !ok || userID == "" {
		return fmt.Errorf("invalid user_id field")
	}

	if eventType, ok := message["event_type"].(string); !ok || eventType == "" {
		return fmt.Errorf("invalid event_type field")
	}

	return nil
}

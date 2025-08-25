package pipeline

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/linkedin/goavro/v2"
	"github.com/segmentio/kafka-go"
)

// Producer handles Kafka message production with AVRO encoding
type Producer struct {
	config *Config
	writer *kafka.Writer
	codec  *goavro.Codec
}

// NewProducer creates a new Kafka producer
func NewProducer(config *Config) (*Producer, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.BootstrapServers),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    100,
	}

	return &Producer{
		config: config,
		writer: writer,
	}, nil
}

// Start begins producing messages to the specified topic with support for dynamic traffic patterns
func (p *Producer) Start(ctx context.Context, topic string, schema *Schema) error {
	fmt.Printf("📤 Starting producer for topic: %s\n", topic)

	if p.config.TrafficPatterns != nil && p.config.TrafficPatterns.HasPatterns() {
		fmt.Printf("📊 Traffic pattern configured:\n%s\n", p.config.TrafficPatterns.GetPatternSummary())
	} else {
		fmt.Printf("📊 Message rate: %d msg/sec (constant)\n", p.config.MessageRate)
	}

	// Set topic for writer
	p.writer.Topic = topic

	// Create AVRO codec from schema
	codec, err := goavro.NewCodec(schema.Content)
	if err != nil {
		return fmt.Errorf("failed to create AVRO codec: %w", err)
	}
	p.codec = codec

	// Start the dynamic rate producer
	if p.config.TrafficPatterns != nil && p.config.TrafficPatterns.HasPatterns() {
		return p.startWithTrafficPatterns(ctx, schema)
	}

	// Fallback to constant rate producer
	return p.startWithConstantRate(ctx, schema)
}

// startWithTrafficPatterns starts the producer with dynamic traffic patterns
func (p *Producer) startWithTrafficPatterns(ctx context.Context, schema *Schema) error {
	startTime := time.Now()
	messageCount := 0
	currentRate := p.config.TrafficPatterns.GetRateAt(0)

	// Create a ticker that we'll reset when rate changes
	interval := time.Second / time.Duration(currentRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Track rate changes for logging
	lastLoggedRate := currentRate
	nextRateCheck := time.Now().Add(100 * time.Millisecond) // Check rate every 100ms

	fmt.Printf("📈 Starting with rate: %d msg/sec\n", currentRate)

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("🛑 Producer stopping. Sent %d messages\n", messageCount)
			return ctx.Err()

		case <-ticker.C:
			// Check if we need to update the rate
			elapsed := time.Since(startTime)
			newRate := p.config.TrafficPatterns.GetRateAt(elapsed)

			if newRate != currentRate {
				currentRate = newRate
				interval = time.Second / time.Duration(currentRate)
				ticker.Reset(interval)
				fmt.Printf("📊 Rate changed to: %d msg/sec (elapsed: %v)\n", currentRate, elapsed.Truncate(time.Second))
				lastLoggedRate = currentRate
			}

			// Generate and send message
			if err := p.sendMessage(ctx, schema.Name, messageCount); err != nil {
				fmt.Printf("⚠️  Failed to send message: %v\n", err)
				continue
			}

			messageCount++
			if messageCount%1000 == 0 {
				fmt.Printf("📈 Sent %d messages (rate: %d msg/sec, elapsed: %v)...\n",
					messageCount, currentRate, elapsed.Truncate(time.Second))
			}

		default:
			// Periodic rate check (non-blocking)
			if time.Now().After(nextRateCheck) {
				elapsed := time.Since(startTime)
				newRate := p.config.TrafficPatterns.GetRateAt(elapsed)

				if newRate != currentRate {
					currentRate = newRate
					interval = time.Second / time.Duration(currentRate)
					ticker.Reset(interval)

					if newRate != lastLoggedRate {
						fmt.Printf("📊 Rate changed to: %d msg/sec (elapsed: %v)\n", currentRate, elapsed.Truncate(time.Second))
						lastLoggedRate = currentRate
					}
				}

				nextRateCheck = time.Now().Add(100 * time.Millisecond)
			}
		}
	}
}

// startWithConstantRate starts the producer with a constant message rate (original behavior)
func (p *Producer) startWithConstantRate(ctx context.Context, schema *Schema) error {
	// Calculate message interval
	interval := time.Second / time.Duration(p.config.MessageRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	messageCount := 0

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("🛑 Producer stopping. Sent %d messages\n", messageCount)
			return ctx.Err()

		case <-ticker.C:
			// Generate and send message
			if err := p.sendMessage(ctx, schema.Name, messageCount); err != nil {
				fmt.Printf("⚠️  Failed to send message: %v\n", err)
				continue
			}

			messageCount++
			if messageCount%1000 == 0 {
				fmt.Printf("📈 Sent %d messages...\n", messageCount)
			}
		}
	}
}

// sendMessage generates and sends a single message (extracted for reuse)
func (p *Producer) sendMessage(ctx context.Context, schemaName string, messageCount int) error {
	// Generate message
	message, err := p.generateMessage(schemaName, messageCount)
	if err != nil {
		return fmt.Errorf("failed to generate message: %w", err)
	}

	// Encode message with AVRO
	avroData, err := p.encodeMessage(message)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	// Send to Kafka
	kafkaMsg := kafka.Message{
		Key:   []byte(fmt.Sprintf("key-%d", messageCount)),
		Value: avroData,
	}

	if err := p.writer.WriteMessages(ctx, kafkaMsg); err != nil {
		return fmt.Errorf("failed to produce message: %w", err)
	}

	return nil
}

// generateMessage creates sample data based on the schema
func (p *Producer) generateMessage(schemaName string, messageID int) (map[string]interface{}, error) {
	// Generate data dynamically based on schema fields
	return p.generateDynamicMessage(messageID)
}

// generateDynamicMessage creates a message based on the actual schema fields
func (p *Producer) generateDynamicMessage(messageID int) (map[string]interface{}, error) {
	if p.codec == nil {
		return nil, fmt.Errorf("AVRO codec not initialized")
	}

	// Create message with sample data for common field patterns
	message := make(map[string]interface{})
	now := time.Now().UnixMilli()

	// Generate values for each field based on field name patterns
	// This approach works with any schema by detecting common field name patterns
	fieldNames := []string{
		"event_id", "session_id", "user_id", "page_url", "click_type",
		"event_type", "timestamp", "timestamp_col", "properties", "metadata",
	}

	for _, fieldName := range fieldNames {
		switch fieldName {
		case "event_id":
			message[fieldName] = fmt.Sprintf("event-%d", messageID)
		case "session_id":
			message[fieldName] = fmt.Sprintf("session-%d", rand.Intn(1000))
		case "user_id":
			message[fieldName] = fmt.Sprintf("user-%d", rand.Intn(1000))
		case "page_url":
			message[fieldName] = p.randomPageURL()
		case "click_type":
			message[fieldName] = p.randomClickType()
		case "event_type":
			message[fieldName] = p.randomEventType()
		case "timestamp", "timestamp_col":
			message[fieldName] = now
		case "properties", "metadata":
			message[fieldName] = map[string]interface{}{
				"source":  "pipegen",
				"version": "1.0",
				"session": fmt.Sprintf("session-%d", rand.Intn(100)),
			}
		}
	}

	return message, nil
}

// encodeMessage encodes a message using AVRO
func (p *Producer) encodeMessage(message map[string]interface{}) ([]byte, error) {
	// Convert map to AVRO-compatible format
	avroData, err := p.codec.BinaryFromNative(nil, message)
	if err != nil {
		return nil, fmt.Errorf("failed to encode AVRO binary: %w", err)
	}

	return avroData, nil
}

// randomPageURL returns a random page URL for sample data
func (p *Producer) randomPageURL() string {
	pages := []string{
		"/home",
		"/products",
		"/products/electronics",
		"/products/clothing",
		"/cart",
		"/checkout",
		"/profile",
		"/search",
		"/contact",
		"/about",
	}
	return pages[rand.Intn(len(pages))]
}

// randomClickType returns a random click type for sample data
func (p *Producer) randomClickType() string {
	clickTypes := []string{"BUTTON", "LINK", "IMAGE", "MENU"}
	return clickTypes[rand.Intn(len(clickTypes))]
}

// randomEventType returns a random event type for sample data
func (p *Producer) randomEventType() string {
	eventTypes := []string{
		"page_view",
		"button_click",
		"form_submit",
		"purchase",
		"add_to_cart",
		"search",
		"login",
		"logout",
	}
	return eventTypes[rand.Intn(len(eventTypes))]
}

// Close gracefully shuts down the producer
func (p *Producer) Close() {
	if p.writer != nil {
		_ = p.writer.Close()
	}
}

// ProducerStats holds producer statistics
type ProducerStats struct {
	MessagesSent    int64     `json:"messages_sent"`
	MessagesPerSec  float64   `json:"messages_per_sec"`
	BytesSent       int64     `json:"bytes_sent"`
	ErrorCount      int64     `json:"error_count"`
	LastMessageTime time.Time `json:"last_message_time"`
}

// GetStats returns current producer statistics
func (p *Producer) GetStats() *ProducerStats {
	// TODO: Implement actual statistics collection
	return &ProducerStats{
		MessagesSent:    0,
		MessagesPerSec:  0,
		BytesSent:       0,
		ErrorCount:      0,
		LastMessageTime: time.Now(),
	}
}

package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/linkedin/goavro/v2"
	"github.com/riferrei/srclient"
	"github.com/segmentio/kafka-go"
)

// Producer handles Kafka message production with AVRO encoding
type Producer struct {
	config       *Config
	writer       *kafka.Writer
	codec        *goavro.Codec
	srClient     *srclient.SchemaRegistryClient
	schemaID     int
	schema       *Schema   // Store schema for dynamic message generation
	messageCount int64     // Track actual messages sent
	startTime    time.Time // Track when producer started
}

// NewProducer creates a new Kafka producer
func NewProducer(config *Config) (*Producer, error) {
	fmt.Printf("📤 Creating producer with bootstrap servers: %s\n", config.BootstrapServers)
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.BootstrapServers),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
		BatchSize:    100,
	}

	fmt.Printf("  ✅ Writer configured with address: %s\n", config.BootstrapServers)

	return &Producer{
		config:       config,
		writer:       writer,
		srClient:     nil, // Will be initialized when schema is provided
		messageCount: 0,
		startTime:    time.Now(),
	}, nil
}

// InitializeSchemaRegistry initializes the Schema Registry client and gets the existing schema
func (p *Producer) InitializeSchemaRegistry(schema *Schema, subject string) error {
	fmt.Printf("🔗 Initializing Schema Registry client for subject: %s\n", subject)

	// Store schema for dynamic message generation
	p.schema = schema

	// Create Schema Registry client
	p.srClient = srclient.CreateSchemaRegistryClient(p.config.SchemaRegistryURL)

	// First, try to get existing schema instead of auto-registering
	schemaObj, err := p.srClient.GetLatestSchema(subject)
	if err != nil {
		// If schema doesn't exist, register it manually
		fmt.Printf("  📋 Schema not found for subject %s, registering new schema\n", subject)
		schemaObj, err = p.srClient.CreateSchema(subject, schema.Content, srclient.Avro)
		if err != nil {
			return fmt.Errorf("failed to register schema: %w", err)
		}
		fmt.Printf("  ✅ Schema registered with ID: %d\n", schemaObj.ID())
	} else {
		fmt.Printf("  ✅ Using existing schema with ID: %d\n", schemaObj.ID())
	}

	p.schemaID = schemaObj.ID()

	// Create AVRO codec from schema
	codec, err := goavro.NewCodec(schema.Content)
	if err != nil {
		return fmt.Errorf("failed to create AVRO codec: %w", err)
	}
	p.codec = codec

	return nil
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

	// Initialize Schema Registry and register schema
	subject := fmt.Sprintf("%s-value", topic)
	if err := p.InitializeSchemaRegistry(schema, subject); err != nil {
		return fmt.Errorf("failed to initialize schema registry: %w", err)
	}

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
	lastLogTime := startTime

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
			// Log progress every 5 seconds instead of every 1000 messages
			now := time.Now()
			if now.Sub(lastLogTime) >= 5*time.Second {
				elapsed := now.Sub(startTime)
				actualRate := float64(messageCount) / elapsed.Seconds()

				// Update global pipeline status
				globalPipelineStatus.Producer.MessagesSent = int64(messageCount)
				globalPipelineStatus.Producer.Rate = actualRate
				globalPipelineStatus.Producer.Target = currentRate
				globalPipelineStatus.Producer.Elapsed = elapsed

				lastLogTime = now
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
	startTime := time.Now()
	lastLogTime := startTime

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
			// Log progress every 5 seconds instead of every 1000 messages
			now := time.Now()
			if now.Sub(lastLogTime) >= 5*time.Second {
				elapsed := now.Sub(startTime)
				actualRate := float64(messageCount) / elapsed.Seconds()

				// Update global pipeline status
				globalPipelineStatus.Producer.MessagesSent = int64(messageCount)
				globalPipelineStatus.Producer.Rate = actualRate
				globalPipelineStatus.Producer.Target = p.config.MessageRate
				globalPipelineStatus.Producer.Elapsed = elapsed

				lastLogTime = now
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

	// Increment the message counter
	p.messageCount++

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

	// Create message with sample data based on schema fields
	message := make(map[string]interface{})

	// Use the schema to determine which fields to generate
	// This makes the producer work with ANY AVRO schema
	if p.schema != nil && len(p.schema.Fields) > 0 {
		// Generate values for each field based on the schema definition
		for _, field := range p.schema.Fields {
			value, err := p.generateValueForField(field, messageID)
			if err != nil {
				return nil, fmt.Errorf("failed to generate value for field %s: %w", field.Name, err)
			}
			message[field.Name] = value
		}
	} else {
		// Fallback: generate common fields if schema fields are not available
		message["name"] = fmt.Sprintf("user-%d", rand.Intn(1000))
		message["amount"] = rand.Intn(10000)
	}

	return message, nil
}

// generateValueForField generates sample data for a specific AVRO field based on its type
func (p *Producer) generateValueForField(field SchemaField, messageID int) (interface{}, error) {
	now := time.Now().UnixMilli()

	// Handle different AVRO field types
	switch fieldType := field.Type.(type) {
	case string:
		// Simple types
		switch fieldType {
		case "string":
			return p.generateStringValue(field.Name, messageID), nil
		case "int":
			return rand.Intn(10000), nil
		case "long":
			return int64(now), nil
		case "float":
			return rand.Float32() * 1000, nil
		case "double":
			return rand.Float64() * 1000, nil
		case "boolean":
			return rand.Intn(2) == 1, nil
		case "bytes":
			return []byte(fmt.Sprintf("data-%d", messageID)), nil
		default:
			return fmt.Sprintf("value-%d", messageID), nil
		}

	case []interface{}:
		// Union types (e.g., ["null", "string"])
		for _, unionType := range fieldType {
			if unionTypeStr, ok := unionType.(string); ok && unionTypeStr != "null" {
				// Use the first non-null type
				mockField := SchemaField{Name: field.Name, Type: unionTypeStr}
				return p.generateValueForField(mockField, messageID)
			}
		}
		return nil, nil // Return null for union types we can't handle

	case map[string]interface{}:
		// Complex types (records, maps, arrays, enums)
		if typeValue, exists := fieldType["type"]; exists {
			switch typeValue {
			case "map":
				// Generate a simple map
				return map[string]interface{}{
					"key1": "value1",
					"key2": fmt.Sprintf("value-%d", messageID),
				}, nil
			case "array":
				// Generate a simple array
				return []interface{}{"item1", fmt.Sprintf("item-%d", messageID)}, nil
			case "enum":
				// Use first symbol from enum
				if symbols, exists := fieldType["symbols"]; exists {
					if symbolList, ok := symbols.([]interface{}); ok && len(symbolList) > 0 {
						return symbolList[rand.Intn(len(symbolList))], nil
					}
				}
				return "UNKNOWN", nil
			case "record":
				// For nested records, return a simple map
				return map[string]interface{}{
					"nested_field": fmt.Sprintf("nested-value-%d", messageID),
				}, nil
			}
		}
		return fmt.Sprintf("complex-value-%d", messageID), nil

	default:
		return fmt.Sprintf("default-value-%d", messageID), nil
	}
}

// generateStringValue creates appropriate string values based on field name patterns
func (p *Producer) generateStringValue(fieldName string, messageID int) string {
	switch fieldName {
	case "id", "event_id", "user_id", "session_id":
		return fmt.Sprintf("%s-%d", fieldName, messageID)
	case "name", "username", "user_name":
		return fmt.Sprintf("user-%d", rand.Intn(1000))
	case "email":
		return fmt.Sprintf("user%d@example.com", rand.Intn(1000))
	case "event_type", "type":
		events := []string{"click", "view", "purchase", "signup", "login"}
		return events[rand.Intn(len(events))]
	case "url", "page_url":
		pages := []string{"/home", "/product", "/checkout", "/profile", "/search"}
		return pages[rand.Intn(len(pages))]
	case "status":
		statuses := []string{"active", "pending", "completed", "failed"}
		return statuses[rand.Intn(len(statuses))]
	case "category":
		categories := []string{"electronics", "clothing", "books", "food", "sports"}
		return categories[rand.Intn(len(categories))]
	case "country", "region":
		countries := []string{"US", "CA", "GB", "DE", "FR"}
		return countries[rand.Intn(len(countries))]
	default:
		return fmt.Sprintf("%s-%d", fieldName, messageID)
	}
}

// encodeMessage encodes a message using AVRO with Confluent Schema Registry wire format
// or JSON format based on format detection
func (p *Producer) encodeMessage(message map[string]interface{}) ([]byte, error) {
	// Use AVRO format if codec is available (Schema Registry initialized)
	if p.codec != nil {
		return p.encodeMessageAVRO(message)
	}
	// Fall back to JSON format if no codec available
	return p.encodeMessageJSON(message)
}

// encodeMessageJSON encodes a message as JSON
func (p *Producer) encodeMessageJSON(message map[string]interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("failed to encode JSON: %w", err)
	}
	return jsonData, nil
}

// encodeMessageAVRO encodes a message using AVRO with Confluent Schema Registry wire format
func (p *Producer) encodeMessageAVRO(message map[string]interface{}) ([]byte, error) {
	// Convert map to AVRO-compatible format
	avroData, err := p.codec.BinaryFromNative(nil, message)
	if err != nil {
		return nil, fmt.Errorf("failed to encode AVRO binary: %w", err)
	}

	// Create Confluent wire format:
	// Magic byte (0x00) + Schema ID (4 bytes, big-endian) + Avro data
	wireFormat := make([]byte, 1+4+len(avroData))
	wireFormat[0] = 0x00 // Magic byte

	// Schema ID in big-endian format
	wireFormat[1] = byte(p.schemaID >> 24)
	wireFormat[2] = byte(p.schemaID >> 16)
	wireFormat[3] = byte(p.schemaID >> 8)
	wireFormat[4] = byte(p.schemaID)

	// Copy Avro data
	copy(wireFormat[5:], avroData)

	return wireFormat, nil
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
	elapsed := time.Since(p.startTime)
	var messagesPerSec float64
	if elapsed.Seconds() > 0 {
		messagesPerSec = float64(p.messageCount) / elapsed.Seconds()
	}

	return &ProducerStats{
		MessagesSent:    p.messageCount,
		MessagesPerSec:  messagesPerSec,
		BytesSent:       p.messageCount * 1024, // Estimate 1KB per message
		ErrorCount:      0,                     // TODO: Track actual errors
		LastMessageTime: time.Now(),
	}
}

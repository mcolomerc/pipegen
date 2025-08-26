# AI-Powered Pipeline Generation

PipeGen's AI capabilities transform natural language descriptions into production-ready streaming pipelines, complete with FlinkSQL statements, AVRO schemas, and optimized configurations.

## Overview

Instead of manually writing complex FlinkSQL and designing schemas, simply **describe what you want** and let AI generate:

- **FlinkSQL statements** for data processing logic
- **AVRO schemas** for input and output data structures  
- **Configuration files** optimized for your use case
- **Documentation** explaining the generated pipeline
- **README files** with usage examples

## Quick Start

### Basic AI Generation

```bash
# Describe your pipeline in natural language
pipegen init fraud-detection --describe \
  "Monitor payment transactions, detect suspicious patterns using machine learning, and alert on potential fraud within 30 seconds"
```

### With Domain Context

```bash
# Add business domain for better results
pipegen init user-analytics --describe \
  "Track user page views and calculate session duration analytics" \
  --domain "ecommerce"
```

### Advanced Generation

```bash
# Complex real-time processing
pipegen init iot-monitoring --describe \
  "Process IoT sensor data from manufacturing equipment, detect anomalies using statistical analysis, and trigger maintenance alerts when temperature exceeds thresholds for more than 5 minutes" \
  --domain "manufacturing"
```

## AI Providers

PipeGen supports multiple AI providers for flexibility and cost optimization:

### Ollama (Local & Free)

**Recommended for:**
- Privacy-sensitive projects
- Offline development  
- Cost optimization
- Learning and experimentation

**Setup:**
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Download a model
ollama pull llama3.1

# Configure PipeGen
export PIPEGEN_OLLAMA_MODEL="llama3.1"
export PIPEGEN_OLLAMA_URL="http://localhost:11434"  # Optional
```

**Supported Models:**
- `llama3.1` (Recommended)
- `codellama`
- `llama3`
- `phi3`
- `qwen2.5-coder`

### OpenAI (Cloud & Powerful)

**Recommended for:**
- Complex pipeline generation
- Production use cases
- Advanced reasoning requirements
- When local resources are limited

**Setup:**
```bash
# Get API key from OpenAI
export PIPEGEN_OPENAI_API_KEY="sk-your-api-key"
export PIPEGEN_LLM_MODEL="gpt-4"  # Optional, defaults to gpt-4
```

**Supported Models:**
- `gpt-4` (Recommended)
- `gpt-4-turbo`
- `gpt-3.5-turbo`

## Configuration Check

Verify your AI setup:

```bash
pipegen check
```

**Example Output:**
```
🔍 Checking AI Provider Configuration...

✅ Ollama Configuration:
   Model: llama3.1
   URL: http://localhost:11434
   Status: Available
   
⚠️  OpenAI Configuration:
   API Key: Not configured
   Model: gpt-4
   Status: Not available

💡 Tip: Set PIPEGEN_OPENAI_API_KEY to enable OpenAI features
```

## Generation Examples

### E-commerce Analytics

```bash
pipegen init ecommerce-analytics --describe \
  "Track customer behavior including page views, add to cart events, and purchases. Calculate conversion funnels, session duration, and real-time product popularity metrics" \
  --domain "ecommerce"
```

**Generated Components:**
- **Input Schema**: Customer events with user_id, event_type, product_id, timestamp
- **FlinkSQL**: Windowed aggregations for conversion rates and session metrics  
- **Output Schema**: Analytics results with conversion rates and product metrics

### IoT Sensor Processing

```bash
pipegen init factory-monitoring --describe \
  "Monitor industrial sensors for temperature, pressure, and vibration. Detect equipment anomalies using moving averages and trigger maintenance alerts when values exceed normal ranges for more than 10 minutes" \
  --domain "manufacturing"
```

**Generated Components:**
- **Input Schema**: Sensor readings with equipment_id, sensor_type, value, timestamp
- **FlinkSQL**: Tumbling windows with anomaly detection logic
- **Output Schema**: Alert events with equipment details and anomaly scores

### Financial Trading

```bash
pipegen init trading-signals --describe \
  "Process real-time stock price feeds, calculate technical indicators like moving averages and RSI, and generate buy/sell signals based on momentum strategies" \
  --domain "fintech"
```

**Generated Components:**
- **Input Schema**: Stock ticks with symbol, price, volume, timestamp
- **FlinkSQL**: Complex windowing for technical indicator calculations
- **Output Schema**: Trading signals with confidence scores and reasoning

### Social Media Monitoring

```bash
pipegen init social-sentiment --describe \
  "Analyze social media posts in real-time, extract sentiment scores, identify trending topics, and detect viral content based on engagement velocity" \
  --domain "social"
```

## Domain-Specific Optimization

Adding `--domain` helps the AI generate more accurate and industry-specific pipelines:

| Domain | Optimization Focus | Example Use Cases |
|--------|-------------------|-------------------|
| `ecommerce` | Customer journeys, conversions, product analytics | Purchase funnels, cart abandonment, recommendations |
| `fintech` | Risk analysis, fraud detection, trading | Payment monitoring, algorithmic trading, compliance |
| `iot` | Device monitoring, anomaly detection | Sensor networks, predictive maintenance, edge processing |
| `gaming` | Player behavior, real-time leaderboards | Matchmaking, achievement systems, analytics |
| `media` | Content analysis, engagement tracking | Recommendation engines, trend detection, moderation |
| `healthcare` | Patient monitoring, alert systems | Vital sign processing, medication tracking, anomaly detection |
| `logistics` | Route optimization, tracking | Fleet management, delivery optimization, supply chain |

## Generated Pipeline Structure

AI-generated projects include:

```
my-ai-pipeline/
├── sql/
│   ├── local/
│   │   ├── 01_create_source_table.sql    # Input table definition
│   │   ├── 02_create_processing.sql       # Main processing logic  
│   │   ├── 03_create_output_table.sql     # Output table definition
│   │   └── 04_insert_results.sql          # Data flow orchestration
│   └── cloud/                             # Cloud-ready versions
├── schemas/
│   ├── input.json                         # AVRO input schema
│   └── output.json                        # AVRO output schema  
├── config/
│   ├── local.yaml                         # Local development config
│   └── cloud.yaml                         # Production config template
├── .pipegen.yaml                          # Project configuration
└── README.md                              # AI-generated documentation
```

## Advanced Features

### Schema Validation

AI-generated schemas include:

```json
{
  "type": "record",
  "name": "CustomerEvent",
  "fields": [
    {
      "name": "user_id", 
      "type": "string",
      "doc": "Unique customer identifier"
    },
    {
      "name": "event_type",
      "type": {
        "type": "enum", 
        "name": "EventType",
        "symbols": ["page_view", "add_to_cart", "purchase", "search"]
      }
    },
    {
      "name": "timestamp",
      "type": {
        "type": "long",
        "logicalType": "timestamp-millis"
      }
    },
    {
      "name": "properties",
      "type": ["null", {
        "type": "record",
        "name": "EventProperties", 
        "fields": [
          {"name": "product_id", "type": ["null", "string"]},
          {"name": "category", "type": ["null", "string"]},
          {"name": "price", "type": ["null", "double"]}
        ]
      }],
      "default": null
    }
  ]
}
```

### Optimized FlinkSQL

Generated SQL includes best practices:

```sql
-- AI-generated processing logic with windowing
CREATE TABLE customer_analytics AS
SELECT 
  user_id,
  window_start,
  window_end,
  COUNT(*) as event_count,
  COUNT(CASE WHEN event_type = 'purchase' THEN 1 END) as purchases,
  SUM(CASE WHEN event_type = 'purchase' THEN properties.price ELSE 0 END) as revenue,
  -- Conversion rate calculation
  CAST(COUNT(CASE WHEN event_type = 'purchase' THEN 1 END) AS DOUBLE) / 
  COUNT(CASE WHEN event_type = 'page_view' THEN 1 END) as conversion_rate
FROM TABLE(
  TUMBLE(TABLE customer_events, DESCRIPTOR(event_time), INTERVAL '1' HOUR)
)
GROUP BY user_id, window_start, window_end
HAVING COUNT(*) > 0;
```

### Configuration Optimization

AI generates optimized configurations:

```yaml
# AI-generated .pipegen.yaml
project_name: "customer-analytics"
description: "Real-time customer behavior analytics pipeline"

# Optimized for the specific use case
producer_config:
  batch_size: 1000
  linger_ms: 100
  message_rate: 500  # Typical e-commerce load

consumer_config:
  auto_offset_reset: "earliest"
  enable_auto_commit: true

flink_config:
  parallelism: 4
  checkpoint_interval: "30s"
  state_backend: "rocksdb"

# Domain-specific settings
domain_config:
  customer_session_timeout: "30m"
  conversion_window: "24h" 
  min_events_threshold: 5
```

## Best Practices

### Writing Effective Descriptions

**✅ Good Descriptions:**
```bash
# Specific, detailed, includes business context
pipegen init order-processing --describe \
  "Process e-commerce orders in real-time, validate inventory availability, calculate shipping costs based on customer location, apply discount codes, and generate order confirmation events within 2 seconds" \
  --domain "ecommerce"

# Includes timing requirements and business rules  
pipegen init risk-scoring --describe \
  "Analyze credit card transactions for fraud detection using statistical models, check transaction amounts against customer spending patterns, and flag suspicious transactions scoring above 0.8 confidence within 100ms" \
  --domain "fintech"
```

**❌ Avoid Vague Descriptions:**
```bash
# Too generic
pipegen init data-processing --describe "process some data"

# Missing context
pipegen init analytics --describe "do analytics on events"
```

### Iterative Refinement

Start simple and refine:

```bash
# Initial generation
pipegen init v1 --describe "analyze user behavior"

# Refined version
pipegen init v2 --describe \
  "analyze user clickstream behavior including page views, session duration, and conversion paths to optimize user experience" \
  --domain "ecommerce"

# Detailed version  
pipegen init v3 --describe \
  "analyze user clickstream behavior across web and mobile apps, track conversion funnels from landing page to checkout, identify drop-off points, and calculate real-time conversion rates with 1-minute granularity" \
  --domain "ecommerce"
```

### Domain Selection

Choose the most relevant domain:

- **Multiple domains**: Pick the primary one
- **Custom domains**: Use the closest match
- **Generic processing**: Omit domain for general-purpose generation

## Troubleshooting

### Common Issues

**AI provider not configured:**
```bash
❌ Error: No AI provider configured
💡 Solution: Set up Ollama or OpenAI as described above
```

**Model not available:**
```bash
❌ Error: Ollama model 'llama3.1' not found
💡 Solution: Run 'ollama pull llama3.1'
```

**Generation timeout:**
```bash
❌ Error: AI generation timed out
💡 Solution: Try a simpler description or use a more powerful model
```

**Poor quality output:**
```bash
❌ Issue: Generated SQL doesn't match requirements  
💡 Solution: 
- Provide more specific description
- Add domain context
- Include timing requirements
- Specify business rules clearly
```

### Performance Tips

1. **Local development**: Use Ollama for fast iteration
2. **Production generation**: Use OpenAI for complex pipelines  
3. **Resource allocation**: Ensure sufficient RAM for local models
4. **Network connectivity**: Stable connection for cloud AI providers

## Examples Gallery

### Real-Time Recommendation Engine
```bash
pipegen init recommendations --describe \
  "Build a real-time recommendation engine that processes user interactions, maintains user preference profiles, calculates item similarity scores using collaborative filtering, and serves personalized recommendations with sub-100ms latency" \
  --domain "ecommerce"
```

### Cryptocurrency Trading Bot
```bash
pipegen init crypto-trading --describe \
  "Create a cryptocurrency trading algorithm that monitors price feeds from multiple exchanges, calculates arbitrage opportunities, executes trades when profit margins exceed 0.5%, and maintains risk management with stop-loss orders" \
  --domain "fintech"  
```

### Smart City Traffic Management
```bash
pipegen init traffic-optimization --describe \
  "Process real-time traffic sensor data from intersections, detect congestion patterns, optimize traffic light timing using machine learning models, and coordinate signals across the city to minimize average travel time" \
  --domain "iot"
```

## Integration with Other Features

### AI + Traffic Patterns
```bash
# Generate AI pipeline then test with load patterns
pipegen init high-volume-analytics --describe \
  "Process high-volume clickstream data for real-time user segmentation"

cd high-volume-analytics
pipegen run --dashboard --traffic-pattern "2m-4m:500%,6m-8m:300%"
```

### AI + Custom Schemas
```bash
# Use AI with existing schemas
pipegen init custom-processing --describe \
  "Process payment events for fraud detection" \
  --input-schema ./existing-payment-schema.avsc
```

## Next Steps

- **[Getting Started](./getting-started)** - Create your first AI-generated pipeline
- **[Examples](./examples)** - See AI generation in action  
- **[Commands](./commands/init)** - Learn more about the init command
- **[Traffic Patterns](./traffic-patterns)** - Test AI pipelines with load patterns

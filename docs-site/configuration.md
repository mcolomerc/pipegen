# Configuration

PipeGen uses YAML configuration files to define pipeline behavior, connections, and processing logic.

## Configuration File Structure

```yaml
# config.yaml
pipeline:
  name: "my-streaming-pipeline"
  version: "1.0.0"
  description: "Process user events in real-time"

kafka:
  brokers:
    - "localhost:9092"
    - "kafka-2:9092"
  topics:
    input: "user-events"
    output: "processed-events"
  consumer_group: "pipegen-processor"
  
flink:
  jobmanager_url: "http://localhost:8081"
  parallelism: 4
  checkpoint:
    interval: "30s"
    timeout: "10m"
    mode: "exactly_once"
  memory:
    jobmanager: "1gb"
    taskmanager: "2gb"

schemas:
  input: "schemas/user_event.avsc"
  output: "schemas/processed_event.avsc"
  
processing:
  sql_files:
    - "sql/01_create_source_table.sql"
    - "sql/02_create_processing.sql"
    - "sql/03_create_output_table.sql"
    - "sql/04_insert_results.sql"

monitoring:
  metrics_enabled: true
  dashboard_port: 8080
  log_level: "info"
```

## Configuration Sections

### Pipeline Configuration

```yaml
pipeline:
  name: "my-pipeline"           # Pipeline identifier
  version: "1.0.0"              # Version for tracking
  description: "Pipeline desc"  # Human-readable description
  tags:                         # Optional tags for organization
    - "production"
    - "user-analytics"
```

### Kafka Configuration

```yaml
kafka:
  brokers:                      # List of Kafka brokers
    - "broker1:9092"
    - "broker2:9092"
  topics:
    input: "input-topic"        # Input topic name
    output: "output-topic"      # Output topic name
    error: "error-topic"        # Error topic (optional)
  consumer_group: "pipegen"     # Consumer group ID
  
  # Advanced Kafka settings
  producer:
    batch_size: 16384
    linger_ms: 5
    compression_type: "gzip"
    acks: "all"
    retries: 2147483647
    
  consumer:
    auto_offset_reset: "earliest"
    enable_auto_commit: false
    session_timeout_ms: 30000
    max_poll_records: 500
```

### Flink Configuration

```yaml
flink:
  jobmanager_url: "http://localhost:8081"
  
  # Parallelism and scaling
  parallelism: 4                # Default parallelism
  max_parallelism: 128          # Maximum parallelism
  
  # Checkpointing
  checkpoint:
    interval: "30s"             # Checkpoint interval
    timeout: "10m"              # Checkpoint timeout
    min_pause: "5s"             # Minimum pause between checkpoints
    max_concurrent: 1           # Max concurrent checkpoints
    mode: "exactly_once"        # Processing guarantee
    
  # Resource allocation
  memory:
    jobmanager: "1gb"
    taskmanager: "2gb"
    network: "128mb"
    
  # Recovery
  restart_strategy:
    type: "failure_rate"
    failure_rate: 3
    failure_interval: "5m"
    delay: "10s"
```

### Schema Configuration

```yaml
schemas:
  input: "schemas/input.avsc"           # Input schema file
  output: "schemas/output.avsc"         # Output schema file
  format: "avro"                        # Schema format (avro, json)
  
  # Schema evolution settings
  compatibility: "backward"             # Compatibility mode
  auto_register: true                   # Auto-register schemas
  
  # Schema registry (if using Confluent Schema Registry)
  registry:
    url: "http://schema-registry:8081"
    auth:
      username: "user"
      password: "pass"
```

### Processing Configuration

```yaml
processing:
  sql_files:                            # SQL files in execution order
    - "sql/01_create_source.sql"
    - "sql/02_transform.sql"
    - "sql/03_create_sink.sql"
    - "sql/04_insert.sql"
    
  # Custom functions
  functions:
    - name: "CUSTOM_AGGREGATE"
      class: "com.example.CustomAggregate"
      jar: "libs/custom-functions.jar"
```

### Monitoring Configuration

```yaml
monitoring:
  metrics_enabled: true                 # Enable metrics collection
  dashboard_port: 8080                  # Dashboard port
  log_level: "info"                     # Log level
  
  # Prometheus metrics
  prometheus:
    enabled: true
    port: 9090
    endpoint: "/metrics"
    
  # Health checks
  health_check:
    enabled: true
    port: 8081
    endpoint: "/health"
    interval: "30s"
```

## Environment-Specific Configurations

### Local Development

```yaml
# config-local.yaml
kafka:
  brokers:
    - "localhost:9092"
    
flink:
  jobmanager_url: "http://localhost:8081"
  parallelism: 1
  memory:
    jobmanager: "512mb"
    taskmanager: "1gb"
    
monitoring:
  dashboard_port: 8080
  log_level: "debug"
```

### Staging Environment

```yaml
# config-staging.yaml
kafka:
  brokers:
    - "kafka-staging-1:9092"
    - "kafka-staging-2:9092"
    
flink:
  jobmanager_url: "http://flink-staging:8081"
  parallelism: 2
  memory:
    jobmanager: "1gb"
    taskmanager: "2gb"
    
monitoring:
  log_level: "warn"
```

### Production Environment

```yaml
# config-production.yaml
kafka:
  brokers:
    - "kafka-prod-1:9092"
    - "kafka-prod-2:9092"
    - "kafka-prod-3:9092"
    
flink:
  jobmanager_url: "http://flink-prod:8081"
  parallelism: 8
  max_parallelism: 128
  
  checkpoint:
    interval: "10s"
    mode: "exactly_once"
    
  memory:
    jobmanager: "2gb"
    taskmanager: "4gb"
    
  restart_strategy:
    type: "failure_rate"
    failure_rate: 5
    failure_interval: "10m"
    
monitoring:
  log_level: "error"
  prometheus:
    enabled: true
```

## Configuration Validation

Use the validate command to check configuration:

```bash
# Validate default configuration
pipegen validate

# Validate specific configuration
pipegen validate --config config-production.yaml

# Strict validation
pipegen validate --strict
```

## Environment Variables

Override configuration with environment variables:

```bash
export PIPEGEN_KAFKA_BROKERS="prod-kafka-1:9092,prod-kafka-2:9092"
export PIPEGEN_FLINK_PARALLELISM=8
export PIPEGEN_LOG_LEVEL=warn

pipegen run
```

Variable naming convention:
- Prefix: `PIPEGEN_`
- Nested keys: Use `_` separator
- Arrays: Comma-separated values

## Configuration Templating

Use templates for dynamic configuration:

```yaml
# config-template.yaml
kafka:
  brokers:
    - "{{.KAFKA_BROKER_1}}:9092"
    - "{{.KAFKA_BROKER_2}}:9092"
    
flink:
  parallelism: {{.FLINK_PARALLELISM | default 4}}
  memory:
    taskmanager: "{{.TASKMANAGER_MEMORY | default "2gb"}}"
```

## Configuration Inheritance

Extend base configurations:

```yaml
# base-config.yaml
kafka: &kafka
  consumer_group: "pipegen"
  topics:
    error: "pipeline-errors"
    
flink: &flink
  checkpoint:
    mode: "exactly_once"
    interval: "30s"

# production-config.yaml
kafka:
  <<: *kafka
  brokers:
    - "prod-kafka:9092"
    
flink:
  <<: *flink
  parallelism: 8
```

## Security Configuration

### SSL/TLS Configuration

```yaml
kafka:
  security:
    protocol: "SSL"
    ssl:
      truststore_location: "/path/to/truststore.jks"
      truststore_password: "truststore-password"
      keystore_location: "/path/to/keystore.jks"
      keystore_password: "keystore-password"
      key_password: "key-password"
```

### SASL Authentication

```yaml
kafka:
  security:
    protocol: "SASL_SSL"
    sasl:
      mechanism: "PLAIN"
      username: "kafka-user"
      password: "kafka-password"
```

## Best Practices

### Configuration Organization
- Use environment-specific files
- Keep sensitive data in environment variables
- Validate configurations in CI/CD
- Version control configuration files

### Performance Tuning
- Set appropriate parallelism for your workload
- Configure memory based on data volume
- Tune checkpoint intervals for your latency requirements
- Monitor and adjust based on metrics

### Security
- Use SSL/TLS for production
- Store passwords in secure vaults
- Limit access to configuration files
- Regular security audits

## See Also

- [Getting Started](./getting-started.md)
- [Commands](./commands.md)
- [Troubleshooting](./advanced/troubleshooting.md)

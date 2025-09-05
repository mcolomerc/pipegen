# Run Workflow Deep Dive

This guide provides a comprehensive understanding of what happens when you execute `pipegen run`, including technical details, error handling, and troubleshooting.

## Overview

The `pipegen run` command orchestrates a complex streaming pipeline execution involving multiple components:
- **Docker Stack** (Kafka, Flink, Schema Registry)
- **Project Validation** (SQL, Schemas, Configuration)
- **Resource Management** (Topics, Schemas, Jobs)
- **Data Generation** (Producer with AVRO encoding)
- **Processing Monitoring** (Flink job tracking)
- **Output Validation** (Consumer with Schema Registry)
- **Reporting** (HTML reports with comprehensive analytics)

---

## Detailed Execution Flow

### Phase 1: Pre-Execution Validation 🔍

```bash
pipegen run --message-rate 50 --duration 5m
```

#### 1.1 Configuration Loading
```
✅ Using config file: /path/to/project/.pipegen.yaml
```
- Loads project configuration from `.pipegen.yaml`
- Merges with global config and command-line flags
- Validates required fields and formats

**Potential Issues:**
- Missing configuration file → Creates default
- Invalid YAML syntax → Fails with syntax error
- Missing required fields → Prompts for values

#### 1.2 Docker Stack Health Check
```
✅ Docker stack is already running. Skipping deploy.
```
- Checks if Kafka, Flink, Schema Registry are running
- Verifies network connectivity
- Validates service health endpoints

**Potential Issues:**
- Services not running → Auto-deploys with `pipegen deploy`
- Port conflicts → Reports conflicting processes
- Network issues → Provides connectivity troubleshooting

#### 1.3 Project Structure Validation
```
📖 Loading SQL statements...
📖 Loaded 3 SQL statements from sql/
📋 Loading AVRO schemas...
📋 Loaded 2 AVRO schemas from schemas/
```
- Validates SQL file syntax (FlinkSQL compatibility)
- Parses AVRO schema files (JSON format validation)
- Checks file references and dependencies
- Ensures schema-SQL table compatibility

**Potential Issues:**
- SQL syntax errors → Points to specific line/file
- Invalid AVRO schema → Schema validation error details
- Missing files → Lists expected files and locations

### Phase 2: Resource Preparation 🏷️

#### 2.1 Dynamic Resource Naming
```
🏷️  Generating dynamic resource names...
✅ Generated resources with prefix: pipegen-local-20250903-141842
```
- Creates timestamp-based prefixes to avoid conflicts
- Generates unique topic names
- Prepares schema registry subjects
- Plans cleanup scope

#### 2.2 Resource Cleanup (Pre-execution)
```
🧹 Cleaning up existing topics and schemas...
🗑️  Deleting topics with prefix: pipegen-local
🗑️  Deleting schemas from Schema Registry...
```
- Removes any existing resources with same prefix
- Cleans up orphaned topics and schemas
- Prepares clean environment for execution

**Potential Issues:**
- Topic deletion failures → Warns but continues
- Schema registry connectivity → Retries with backoff
- Insufficient permissions → Reports permission requirements

### Phase 3: Infrastructure Setup 📝

#### 3.1 Kafka Topic Creation
```
📝 Creating Kafka topics...
🔧 Creating topics with prefix: pipegen-local
✅ Created topic: transactions (partitions=1, replication=1)
✅ Created topic: output-results (partitions=1, replication=1)
```
- Extracts topic names from SQL statements
- Creates topics with configured partitions/replication
- Validates topic creation success
- Sets retention and cleanup policies

#### 3.2 Schema Registry Setup
```
📋 Registering AVRO schemas...
📋 Registering schemas in Schema Registry...
✅ Registered schema: input -> transactions-value
✅ Registered schema: output -> output-results-value
```
- Registers AVRO schemas with Schema Registry
- Maps schemas to topic subjects (topic-value pattern)
- Validates schema compatibility
- Returns schema IDs for producer/consumer use

### Phase 4: Flink Deployment ⚡

#### 4.1 SQL Gateway Session Management
```
[Flink SQL Gateway] Session creation response: {"sessionHandle":"abc..."}
```
- Creates new Flink SQL Gateway session (or reuses global session)
- Establishes connection to Flink cluster
- Prepares execution environment
- Sets session-specific configurations

#### 4.2 FlinkSQL Statement Deployment
```
📝 Deploying statement 1: 01_create_source_table
🚀 Deploying SQL statement to Flink SQL Gateway API...
✅ SQL statement executed successfully.
```

**For each SQL file:**
- Submits SQL statement to Flink SQL Gateway
- Monitors execution status (RUNNING → FINISHED)
- Validates successful deployment
- Handles DDL vs DML statements appropriately

**Statement Types Handled:**
- **DDL (Data Definition)**: `CREATE TABLE` statements
- **DML (Data Manipulation)**: `INSERT INTO` statements (create streaming jobs)
- **Query statements**: `SELECT` for validation

### Phase 5: Pipeline Execution 📤

#### 5.1 Producer Initialization
```
📤 Starting Kafka producer...
[Producer] Initializing Schema Registry client for subject: transactions-value
[Producer] Using existing schema with ID: 1
```

**AVRO Producer Process:**
- Connects to Schema Registry
- Retrieves schema by subject name
- Creates AVRO codec for message encoding
- Initializes Kafka producer with AVRO serialization

**Message Generation:**
- Uses schema fields to generate realistic test data
- Encodes messages in Confluent AVRO wire format:
  ```
  Magic Byte (0x00) + Schema ID (4 bytes) + AVRO Binary Data
  ```
- Sends messages at configured rate with traffic patterns

#### 5.2 Traffic Pattern Execution
```
📊 Message rate: 50 msg/sec (constant)
📊 Traffic pattern: 30s-60s:300% → Rate: 150 msg/sec
```
- Implements dynamic rate changes based on traffic pattern
- Calculates rate multipliers for each time window
- Smoothly transitions between different rates
- Provides real-time rate updates

### Phase 6: Monitoring & Validation 📊

#### 6.1 Flink Job Monitoring
```
📊 Flink Job [insert-into_revenue]: Read 450 records, Wrote 450 records, Running for 5m
```
- Monitors Flink streaming job metrics via REST API
- Tracks record processing rates (input/output)
- Monitors job health and status
- Detects processing delays or failures

#### 6.2 Enhanced Monitoring System
```
⚠️  Flink REST API metrics unreliable, using enhanced monitoring...
✅ Processing detected via Consumer Group Lag: Consumer group has lag 0
```

**Multi-layered Monitoring:**
- **Primary**: Flink REST API metrics
- **Fallback**: Consumer group lag analysis
- **Deep**: Topic offset tracking
- **Health**: Service endpoint monitoring

#### 6.3 Consumer Validation
```
👂 Starting Kafka consumer to read Flink processing results...
[Consumer] Initializing Schema Registry client for topic: output-results
[Consumer] Retrieved schema for subject: output-results-value (ID: 2)
```

**AVRO Consumer Process:**
- Initializes Schema Registry client
- Retrieves output schema by subject
- Creates AVRO codec for message deserialization
- Consumes and validates output messages

### Phase 7: Reporting & Cleanup 📄

#### 7.1 Execution Report Generation
```
📄 Generating execution report...
✅ Execution report generated: reports/pipegen-execution-report-20250903-141842.html
```

**Report Contents:**
- Execution timeline and metrics
- Performance charts (throughput, latency)
- Configuration details and parameters
- Error logs and troubleshooting info
- Interactive visualizations with Chart.js

#### 7.2 Resource Cleanup (Post-execution)
```
🧹 Cleaning up resources...
🧹 Cleaning up FlinkSQL deployments...
✅ Cancelled job with ID: abc123...
🗑️  Deleting topics and schemas...
✅ Cleanup completed
```

**Cleanup Process (if `--cleanup=true`):**
- Cancels running Flink streaming jobs
- Deletes created Kafka topics
- Removes registered schemas from Schema Registry
- Closes producer/consumer connections
- Terminates monitoring processes

---

## Configuration Deep Dive

### Project Configuration (`.pipegen.yaml`)

```yaml
# Producer settings
producer_config:
  compression_type: "snappy"
  batch_size: 16384
  linger_ms: 5

# Consumer settings
consumer_config:
  session_timeout_ms: 30000
  heartbeat_interval_ms: 3000
  max_poll_records: 500

# Kafka settings
kafka_config:
  partitions: 1
  replication_factor: 1
  retention_ms: 604800000  # 7 days
```

### Runtime Configuration Priority

1. **Command-line flags** (highest priority)
   ```bash
   pipegen run --message-rate 100 --duration 5m
   ```

2. **Environment variables**
   ```bash
   export PIPEGEN_MESSAGE_RATE=100
   export PIPEGEN_DURATION=5m
   ```

3. **Project `.pipegen.yaml`**
4. **Global `~/.pipegen.yaml`**
5. **Built-in defaults** (lowest priority)

---

## Error Handling & Troubleshooting

### Common Failure Points

#### 1. Docker Stack Issues
```
❌ Error: Cannot connect to Kafka at localhost:9093
```
**Solutions:**
- Check Docker services: `docker compose ps`
- Restart stack: `pipegen clean && pipegen deploy`
- Verify ports: `netstat -tlnp | grep :9093`

#### 2. Schema Registry Problems
```
❌ Error: Failed to register schema: 409 Conflict
```
**Solutions:**
- Check schema compatibility mode
- Delete conflicting schemas: `curl -X DELETE http://localhost:8082/subjects/topic-value`
- Verify schema syntax: `pipegen validate --schemas-only`

#### 3. Flink Job Failures
```
❌ Flink job failed with: NoSuchMethodError
```
**Solutions:**
- Check connector compatibility with Flink 1.18.x
- Verify Flink version alignment
- Review and update connector JARs in `connectors/` directory

#### Connector Management
The `connectors/` directory is automatically populated by PipeGen with required Flink connector JARs. You can also add custom connectors:

```bash
# Add custom connector JARs
cp your-custom-connector.jar ./connectors/

# Restart containers to load new connectors
docker-compose restart flink-jobmanager flink-taskmanager sql-gateway

# Verify connectors are loaded
docker exec flink-jobmanager ls -la /opt/flink/lib/
```

**Default Connectors Included:**
- `flink-sql-connector-kafka-3.1.0-1.18.jar` - Kafka connectivity
- `flink-sql-avro-confluent-registry-1.18.1.jar` - AVRO/Schema Registry support
- `kafka-clients-3.4.0.jar` - Kafka client library
- Jackson libraries for JSON processing
- Additional supporting dependencies

#### 4. AVRO Encoding Issues
```
❌ Failed to deserialize AVRO record: Unknown data format
```
**Solutions:**
- Verify Schema Registry connectivity
- Check schema ID mapping
- Validate wire format encoding

### Debugging Commands

```bash
# Check overall health
pipegen check

# Validate configuration
pipegen validate --strict

# Run with detailed logging
pipegen run --duration 1m --dry-run

# Monitor specific components
docker logs flink-jobmanager
docker logs schema-registry
curl -s http://localhost:8081/jobs
```

---

## Advanced Usage Patterns

### Development Workflow
```bash
# 1. Quick validation
pipegen run --dry-run

# 2. Short test with no cleanup
pipegen run --duration 30s --cleanup=false

# 3. Debug specific issues
pipegen run --duration 1m --message-rate 1 --reports-dir ./reports
```

### Load Testing Workflow
```bash
# 1. Baseline test
pipegen run --message-rate 50 --duration 5m

# 2. Traffic spike simulation
pipegen run --message-rate 50 --duration 10m \
  --traffic-pattern "2m-4m:500%,6m-8m:300%"

# 3. Extended load test with reporting
pipegen run --message-rate 100 --duration 30m \
  --traffic-pattern "5m-15m:200%,20m-25m:400%" \
  --generate-report
```

### Production Readiness Testing
```bash
# 1. Schema evolution test
pipegen run --duration 2m --cleanup=false
# (modify schemas between runs)

# 2. Failure recovery test
pipegen run --duration 10m --cleanup=false
# (manually kill services during execution)

# 3. Performance benchmarking
pipegen run --message-rate 500 --duration 15m \
  --generate-report
```

---

## Report Generation

Every `pipegen run` execution automatically generates comprehensive HTML reports:

### Automatic Report Creation
- Professional HTML reports saved to `reports/` folder
- Timestamped filenames for easy identification
- Interactive charts using Chart.js for data visualization
- Complete configuration snapshots for reproducibility

### Report Contents
```
📊 Report Location: ./reports/pipegen-execution-report-YYYYMMDD-HHMMSS.html
```

**Main Sections:**
- **Executive Summary**: Key metrics and execution status
- **Performance Analytics**: Throughput and latency charts
- **Configuration Details**: Complete pipeline settings snapshot
- **Traffic Pattern Analysis**: Load testing insights and patterns
- **System Health**: Resource usage and component status
- **Error Analysis**: Issue tracking and resolution suggestions

---

## Performance Considerations

### Tuning Parameters

**For High Throughput:**
```bash
pipegen run --message-rate 1000 --duration 10m \
  --producer-config '{"batch.size": 32768, "linger.ms": 10}'
```

**For Low Latency:**
```bash
pipegen run --message-rate 100 --duration 5m \
  --producer-config '{"batch.size": 1, "linger.ms": 0}'
```

**For Resource Efficiency:**
```bash
pipegen run --message-rate 200 --duration 15m \
  --flink-config '{"parallelism": 2, "taskmanager.memory": "1gb"}'
```

### Monitoring Performance Impact

- Report generation adds ~1-2% CPU overhead
- Report generation adds ~2-3% overhead
- Enhanced monitoring adds ~1-2% overhead
- AVRO encoding adds ~10-15% latency vs JSON

---

## See Also

- **[Command Reference](./commands.md)** - All available commands and flags
- **[Configuration Guide](./configuration.md)** - Detailed configuration options
- **[Traffic Patterns](./traffic-patterns.md)** - Advanced traffic simulation
- **[Execution Reports Guide](./features/reports.md)** - Professional HTML report generation
- **[Troubleshooting](./advanced/troubleshooting.md)** - Common issues and solutions

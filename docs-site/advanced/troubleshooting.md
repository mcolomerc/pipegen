# Troubleshooting

Comprehensive guide to diagnosing and resolving common issues in PipeGen pipelines.

## General Troubleshooting Steps

### 1. Check System Health
```bash
# Overall system status
pipegen check

# Detailed health check
pipegen check --verbose

# Continuous monitoring
pipegen check --watch --interval 30s
```

### 2. Validate Configuration
```bash
# Validate configuration files
pipegen validate

# Strict validation
pipegen validate --strict

# Validate specific components
pipegen validate --schemas-only
pipegen validate --sql-only
```

### 3. Examine Logs
```bash
# PipeGen logs
pipegen logs --tail 100

# System logs
docker logs pipegen-pipeline
kubectl logs -f deployment/pipegen-pipeline

# Application logs
tail -f /var/log/pipegen/application.log
```

## Common Issues and Solutions

### Pipeline Won't Start

#### Configuration File Not Found
```
Error: config file not found: config.yaml
```

**Solution:**
```bash
# Check current directory
ls -la config.yaml

# Specify config path
pipegen run --config /path/to/config.yaml

# Create default config
pipegen init
```

#### Invalid YAML Syntax
```
Error: yaml: line 10: mapping values are not allowed in this context
```

**Solution:**
```bash
# Validate YAML syntax
yamllint config.yaml

# Use proper indentation (spaces, not tabs)
# Fix quoted strings and special characters
```

#### Missing Schema Files
```
Error: schema file not found: schemas/input.avsc
```

**Solution:**
```bash
# Check file existence
ls -la schemas/

# Verify file paths in config
grep -r "schemas/" config.yaml

# Create missing schemas
mkdir -p schemas/
# Add your schema files
```

### Kafka Connection Issues

#### Broker Unreachable
```
Error: Failed to connect to Kafka broker localhost:9092
```

**Diagnosis:**
```bash
# Test network connectivity
telnet localhost 9092
nc -zv localhost 9092

# Check if Kafka is running
docker ps | grep kafka
systemctl status kafka
```

**Solutions:**
```bash
# Start Kafka if not running
docker-compose up kafka

# Check broker configuration
cat /opt/kafka/config/server.properties | grep listeners

# Update config with correct broker addresses
# config.yaml
kafka:
  brokers:
    - "correct-broker-address:9092"
```

#### Topic Not Found
```
Error: Topic 'input-topic' does not exist
```

**Solution:**
```bash
# List existing topics
kafka-topics.sh --bootstrap-server localhost:9092 --list

# Create missing topic
kafka-topics.sh --bootstrap-server localhost:9092 \
  --create --topic input-topic \
  --partitions 3 --replication-factor 1

# Auto-create topics (not recommended for production)
# In Kafka server.properties:
auto.create.topics.enable=true
```

#### Authentication/Authorization Issues
```
Error: Authentication failed or insufficient permissions
```

**Solution:**
```yaml
# Add authentication to config
kafka:
  security:
    protocol: "SASL_SSL"
    sasl:
      mechanism: "PLAIN"
      username: "your-username"
      password: "your-password"
```

### Flink Connection Issues

#### JobManager Unreachable
```
Error: Could not connect to Flink JobManager at localhost:8081
```

**Diagnosis:**
```bash
# Check if Flink is running
curl http://localhost:8081/overview
docker ps | grep flink
jps | grep -i flink

# Check Flink logs
tail -f /opt/flink/log/flink-*-jobmanager-*.log
```

**Solutions:**
```bash
# Start Flink cluster
/opt/flink/bin/start-cluster.sh

# Or using Docker
docker-compose up flink-jobmanager flink-taskmanager

# Check and update JobManager URL
# config.yaml
flink:
  jobmanager_url: "http://correct-jobmanager:8081"
```

#### Insufficient Resources
```
Error: Not enough free slots available to run the job
```

**Solution:**
```bash
# Check available slots
curl http://localhost:8081/taskmanagers

# Add more TaskManagers
docker-compose up --scale flink-taskmanager=3

# Or increase slots per TaskManager
# flink-conf.yaml
taskmanager.numberOfTaskSlots: 4
```

### Processing Issues

#### Job Failing Repeatedly
```
Job is restarting frequently due to failures
```

**Diagnosis:**
```bash
# Check job status
curl http://localhost:8081/jobs/overview

# Get job details and exceptions
curl http://localhost:8081/jobs/{job-id}/exceptions

# Check Flink TaskManager logs
tail -f /opt/flink/log/flink-*-taskmanager-*.log
```

**Common Causes and Solutions:**

1. **Out of Memory**
```bash
# Increase TaskManager memory
# flink-conf.yaml
taskmanager.memory.process.size: 4gb

# Or in PipeGen config
flink:
  memory:
    taskmanager: "4gb"
```

2. **Serialization Issues**
```
Error: Could not serialize/deserialize object
```
```bash
# Check schema compatibility
pipegen validate --schemas-only

# Verify AVRO schema evolution rules
# Update schemas with proper evolution strategy
```

3. **SQL Syntax Errors**
```
Error: SQL validation failed
```
```bash
# Validate SQL files
pipegen validate --sql-only

# Check FlinkSQL documentation for supported features
# Fix SQL syntax in processing files
```

### Performance Issues

#### Low Throughput
```
Pipeline processing fewer messages than expected
```

**Diagnosis:**
```bash
# Check current metrics
pipegen dashboard

# Monitor resource usage
docker stats
top -p $(pgrep -f taskmanager)

# Check Kafka consumer lag
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --describe --group pipegen-consumer
```

**Solutions:**

1. **Increase Parallelism**
```yaml
flink:
  parallelism: 8  # Increase from default
```

2. **Optimize Kafka Settings**
```yaml
kafka:
  consumer:
    max_poll_records: 1000
    fetch_min_bytes: 50000
  producer:
    batch_size: 65536
    linger_ms: 10
```

3. **Add More Resources**
```yaml
flink:
  memory:
    taskmanager: "8gb"  # Increase memory
  cpu:
    cores: 4            # Allocate more CPU
```

#### High Latency
```
Processing latency higher than acceptable
```

**Solutions:**

1. **Reduce Checkpoint Interval**
```yaml
flink:
  checkpoint:
    interval: "10s"  # Reduce from 30s
```

2. **Optimize Windowing**
```sql
-- Use smaller windows for lower latency
SELECT user_id, COUNT(*)
FROM events
GROUP BY user_id, TUMBLE(event_time, INTERVAL '1' MINUTE)  -- Smaller window
```

3. **Reduce Buffering**
```yaml
kafka:
  producer:
    linger_ms: 1      # Reduce batching delay
    batch_size: 1024  # Smaller batches
```

### Memory Issues

#### Out of Memory Errors
```
Error: java.lang.OutOfMemoryError: Java heap space
```

**Solutions:**

1. **Increase Heap Size**
```bash
# In flink-conf.yaml
jobmanager.memory.heap.size: 2gb
taskmanager.memory.heap.size: 4gb
```

2. **Tune Garbage Collection**
```bash
# Add JVM options
env.java.opts: "-XX:+UseG1GC -XX:MaxGCPauseMillis=200"
```

3. **Use Off-Heap Storage**
```yaml
flink:
  state:
    backend: "rocksdb"  # Use RocksDB for large state
```

#### Memory Leaks
```
Memory usage continuously increasing
```

**Diagnosis:**
```bash
# Monitor memory over time
while true; do
  docker stats --no-stream | grep pipegen
  sleep 60
done

# Generate heap dump
jcmd $(pgrep -f taskmanager) GC.run_finalization
jcmd $(pgrep -f taskmanager) VM.gc
```

**Solutions:**
- Review custom code for object retention
- Check for unclosed resources
- Monitor object creation patterns
- Use profiling tools for detailed analysis

### Data Issues

#### Data Not Appearing in Output
```
Pipeline running but no data in output topic
```

**Diagnosis:**
```bash
# Check input topic has data
kafka-console-consumer.sh --bootstrap-server localhost:9092 \
  --topic input-topic --from-beginning

# Check processing SQL logic
cat sql/processing.sql

# Verify output topic configuration
kafka-topics.sh --bootstrap-server localhost:9092 \
  --describe --topic output-topic
```

#### Schema Evolution Problems
```
Error: Schema compatibility check failed
```

**Solutions:**

1. **Check Schema Registry**
```bash
# List schemas
curl http://schema-registry:8081/subjects

# Check compatibility
curl http://schema-registry:8081/compatibility/subjects/topic-value/versions/latest
```

2. **Update Schema Safely**
```json
{
  "type": "record",
  "name": "Event",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "timestamp", "type": "long"},
    {"name": "new_field", "type": ["null", "string"], "default": null}
  ]
}
```

### Deployment Issues

#### Docker Container Issues
```
Error: Container exits immediately
```

**Diagnosis:**
```bash
# Check container logs
docker logs pipegen-pipeline

# Check container health
docker inspect pipegen-pipeline | grep -i health

# Run container interactively
docker run -it --entrypoint /bin/bash pipegen:latest
```

#### Kubernetes Deployment Issues
```
Error: Pod stuck in Pending state
```

**Diagnosis:**
```bash
# Check pod status
kubectl describe pod pipegen-pipeline-xxx

# Check resource availability
kubectl get nodes
kubectl top nodes

# Check events
kubectl get events --sort-by=.metadata.creationTimestamp
```

**Solutions:**
```yaml
# Update resource requests/limits
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - name: pipegen
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
```

## Debugging Tools and Techniques

### Log Analysis

#### Structured Logging
```bash
# Filter logs by level
grep "ERROR" /var/log/pipegen/application.log

# Filter by component
grep "kafka" /var/log/pipegen/application.log

# Follow logs in real-time
tail -f /var/log/pipegen/application.log | grep -i error
```

#### Log Aggregation
```bash
# Using journalctl
journalctl -u pipegen -f

# Using Docker logs
docker logs -f --tail 100 pipegen-pipeline

# Using Kubernetes
kubectl logs -f deployment/pipegen-pipeline --tail=100
```

### Metrics and Monitoring

#### Key Metrics to Monitor
```bash
# Throughput metrics
curl http://localhost:8080/metrics | grep throughput

# Error rate
curl http://localhost:8080/metrics | grep error_rate

# Resource usage
curl http://localhost:8080/metrics | grep memory_usage
```

#### Alerting Setup
```yaml
# Prometheus alerting rules
groups:
- name: pipegen
  rules:
  - alert: HighErrorRate
    expr: pipegen_error_rate > 0.05
    for: 5m
    annotations:
      summary: "High error rate in PipeGen pipeline"
      
  - alert: LowThroughput
    expr: pipegen_throughput < 100
    for: 10m
    annotations:
      summary: "Low throughput in PipeGen pipeline"
```

### Performance Profiling

#### JVM Profiling
```bash
# Enable JFR profiling
export JAVA_OPTS="-XX:+FlightRecorder -XX:StartFlightRecording=duration=300s,filename=profile.jfr"

# Analyze with JProfiler or similar tools
jprofiler profile.jfr
```

#### Application Profiling
```bash
# Enable detailed metrics
export PIPEGEN_PROFILE=true
pipegen run --verbose

# Use built-in profiler
pipegen run --profile --duration 300s
```

## Emergency Procedures

### Pipeline Recovery

#### Stop Failing Pipeline
```bash
# Graceful shutdown
pipegen stop

# Force stop if needed
pkill -f pipegen
docker stop pipegen-pipeline
kubectl delete deployment pipegen-pipeline
```

#### Restart from Last Checkpoint
```bash
# Check available checkpoints
ls -la /path/to/checkpoints/

# Restart from specific checkpoint
pipegen run --restore-from-checkpoint /path/to/checkpoint
```

### Data Recovery

#### Recover from Kafka
```bash
# Reset consumer group to replay data
kafka-consumer-groups.sh --bootstrap-server localhost:9092 \
  --group pipegen-consumer --reset-offsets --to-earliest --topic input-topic --execute
```

#### Backup and Restore
```bash
# Backup critical data
pipegen backup --output backup-$(date +%Y%m%d).tar.gz

# Restore from backup
pipegen restore --input backup-20240115.tar.gz
```

## Getting Help

### Support Resources
- [GitHub Issues](https://github.com/your-org/pipegen/issues)
- [Community Forum](https://forum.pipegen.io)
- [Documentation](https://docs.pipegen.io)
- [Slack Channel](https://pipegen.slack.com)

### Reporting Issues
```bash
# Generate diagnostic report
pipegen diagnose --output diagnostic-report.zip

# Include system information
pipegen info > system-info.txt
```

### Best Practices for Support
1. Include full error messages and stack traces
2. Provide configuration files (redact sensitive data)
3. Include steps to reproduce the issue
4. Specify environment details (versions, OS, etc.)
5. Attach diagnostic reports and logs

## See Also

- [Performance Optimization](./performance.md)
- [Configuration](../configuration.md)
- [Dashboard](../dashboard.md)
- [Commands Reference](../commands.md)

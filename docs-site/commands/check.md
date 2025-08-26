# pipegen check

The `check` command performs health checks on your pipeline infrastructure and running processes.

## Usage

```bash
pipegen check [flags]
```

## Examples

```bash
# Check all components
pipegen check

# Check specific components
pipegen check --kafka --flink

# Check with detailed output
pipegen check --verbose

# Check and output as JSON
pipegen check --format json

# Continuous health monitoring
pipegen check --watch --interval 30s
```

## Flags

- `--kafka` - Check only Kafka connectivity
- `--flink` - Check only Flink cluster health
- `--pipeline` - Check only pipeline status
- `--all` - Check all components (default)
- `--format` - Output format (table, json, yaml)
- `--verbose` - Show detailed check results
- `--watch` - Continuous monitoring mode
- `--interval` - Check interval in watch mode (default: 10s)
- `--timeout` - Timeout for each check (default: 30s)
- `--help` - Show help for check command

## Health Checks

### Kafka Connectivity
Verifies Kafka cluster health:

- **Broker Connectivity**: Can connect to all brokers
- **Topic Accessibility**: Can list and access topics
- **Producer Test**: Can produce test messages
- **Consumer Test**: Can consume test messages
- **Partition Health**: All partitions have leaders

### Flink Cluster Health
Checks Flink cluster status:

- **JobManager**: JobManager is running and accessible
- **TaskManagers**: All TaskManagers are registered
- **Job Status**: Pipeline jobs are running correctly
- **Checkpointing**: Checkpoints are successful
- **Resource Availability**: Sufficient slots available

### Pipeline Status
Monitors running pipeline:

- **Process Health**: Pipeline process is running
- **Message Flow**: Messages are being processed
- **Error Rates**: Error rates within acceptable limits
- **Latency**: Processing latency is acceptable
- **Backpressure**: No significant backpressure

### System Resources
Checks system-level resources:

- **Memory Usage**: Available memory for processing
- **CPU Usage**: CPU utilization levels
- **Disk Space**: Available disk space
- **Network**: Network connectivity and bandwidth

## Output Formats

### Table Format (Default)
```
Component       Status    Details
─────────────────────────────────────────────────
Kafka           ✅ OK     3/3 brokers online
Flink           ✅ OK     1 JobManager, 2 TaskManagers
Pipeline        ✅ OK     Processing 1.2k msg/s
Memory          ⚠️ WARN   82% usage (6.5GB/8GB)
Disk            ✅ OK     45% usage (450GB/1TB)
```

### JSON Format
```json
{
  "timestamp": "2024-01-15T10:30:00Z",
  "overall_status": "healthy",
  "checks": [
    {
      "component": "kafka",
      "status": "healthy",
      "details": {
        "brokers_online": 3,
        "brokers_total": 3,
        "topics_accessible": 5,
        "response_time_ms": 23
      }
    },
    {
      "component": "flink",
      "status": "healthy",
      "details": {
        "jobmanager_status": "running",
        "taskmanagers_count": 2,
        "available_slots": 8,
        "running_jobs": 1
      }
    }
  ]
}
```

## Check Categories

### Critical Checks ❌
Issues that prevent pipeline operation:

- Kafka brokers unreachable
- Flink cluster down
- Pipeline process crashed
- Out of memory/disk space

### Warning Checks ⚠️
Issues that may impact performance:

- High resource usage (>80%)
- Elevated error rates (>1%)
- Increased latency
- Reduced throughput

### Info Checks ℹ️
Informational status updates:

- Current throughput rates
- Resource usage levels
- Recent job restarts
- Configuration changes

## Automated Monitoring

### Watch Mode
Continuous monitoring with automatic refresh:

```bash
pipegen check --watch --interval 30s
```

### Alerting Integration
Configure alerts based on check results:

```yaml
# alerts.yaml
checks:
  kafka:
    critical:
      action: "email"
      recipients: ["admin@company.com"]
    warning:
      action: "slack"
      channel: "#monitoring"
  
  flink:
    critical:
      action: "pagerduty"
      service_key: "your-service-key"
```

### CI/CD Integration
Use in deployment pipelines:

```yaml
# GitHub Actions
- name: Health Check
  run: |
    pipegen check --format json > health-check.json
    if [ $? -ne 0 ]; then
      echo "Health check failed"
      cat health-check.json
      exit 1
    fi
```

## Troubleshooting

### Kafka Issues

**Broker Connection Failed**
```bash
# Check network connectivity
telnet kafka-broker 9092

# Verify broker configuration
cat /opt/kafka/config/server.properties
```

**Topic Not Found**
```bash
# List available topics
kafka-topics.sh --bootstrap-server localhost:9092 --list

# Create missing topic
kafka-topics.sh --bootstrap-server localhost:9092 --create --topic my-topic
```

### Flink Issues

**JobManager Unreachable**
```bash
# Check Flink processes
jps | grep -i flink

# Check Flink logs
tail -f /opt/flink/log/flink-*-standalonesession-*.log
```

**Insufficient Task Slots**
```bash
# Check available slots
curl http://jobmanager:8081/taskmanagers

# Scale TaskManagers
docker-compose up --scale taskmanager=3
```

### Pipeline Issues

**High Error Rate**
```bash
# Check pipeline logs
pipegen logs --tail 100

# Validate configuration
pipegen validate --strict
```

**Low Throughput**
```bash
# Check resource usage
pipegen check --verbose

# Monitor Kafka lag
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group my-group
```

## Configuration

Check-specific configuration:

```yaml
# check.yaml
checks:
  timeout: 30s
  retry_count: 3
  retry_delay: 5s
  
kafka:
  brokers:
    - "localhost:9092"
  test_topic: "__pipegen_health_check"
  
flink:
  jobmanager_url: "http://localhost:8081"
  required_taskmanagers: 1
  
thresholds:
  memory_warning: 80
  memory_critical: 95
  error_rate_warning: 0.01
  error_rate_critical: 0.05
```

## Exit Codes

- `0`: All checks passed
- `1`: Warning conditions detected
- `2`: Critical issues found
- `3`: Check execution failed

## See Also

- [Dashboard](../dashboard.md)
- [Troubleshooting](../advanced/troubleshooting.md)
- [pipegen validate](./validate.md)

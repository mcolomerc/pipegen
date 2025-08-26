# Live Dashboard & Monitoring

PipeGen includes a sophisticated real-time monitoring dashboard that provides comprehensive visibility into your streaming pipeline performance with WebSocket-based live updates.

<div style="text-align: center; margin: 2rem 0;">
  <img src="/screenshot.png" alt="PipeGen Dashboard" style="max-width: 100%; border-radius: 8px; box-shadow: 0 4px 12px rgba(0,0,0,0.1);">
</div>

## Overview

The dashboard transforms complex streaming metrics into intuitive visualizations, making it easy to:

- **Monitor pipeline health** in real-time
- **Identify bottlenecks** and performance issues  
- **Track data quality** and schema validation
- **Analyze traffic patterns** and load testing results
- **Generate comprehensive reports** for stakeholders

## Getting Started

### Integrated Mode
Run your pipeline with the integrated dashboard:

```bash
pipegen run --dashboard
```

The dashboard automatically opens at `http://localhost:3000` with your pipeline running.

### Standalone Mode  
Start the dashboard without running a pipeline:

```bash
pipegen dashboard --standalone
```

Perfect for monitoring existing services or running multiple pipeline sessions.

### Custom Port
```bash
pipegen run --dashboard --dashboard-port 8080
pipegen dashboard --port 8080
```

## Dashboard Features

### 🔴 Real-Time Metrics

**Live Updates Every Second**
- WebSocket-based streaming updates
- No page refresh required
- Automatic reconnection on connection loss

**Key Metrics Displayed:**
- Messages per second (producer/consumer)
- End-to-end latency percentiles  
- Success/error rates
- Resource utilization
- Data quality scores

### 📊 Pipeline Flow Visualization

Interactive pipeline diagram showing:

```
📤 Producer → 🔄 Kafka → ⚡ Flink → 📥 Output → 👂 Consumer
```

Each component shows:
- **Status indicators** (running, stopped, error)
- **Throughput rates** in real-time
- **Message counts** and processing rates
- **Color-coded health** status

### 📈 Performance Charts

**Throughput Trends**
- Producer rate over time
- Consumer rate over time  
- Traffic pattern visualization
- Comparative analysis

**Latency Analysis**
- P50, P95, P99 latency percentiles
- End-to-end processing time
- Component-wise latency breakdown

**Error Tracking**
- Error rate trends
- Error categorization by component
- Real-time error notifications

### 🎯 Traffic Pattern Monitoring

When using `--traffic-pattern`, the dashboard provides:

- **Pattern timeline** showing current phase
- **Rate transition** visualization  
- **Load impact** on system components
- **Performance correlation** with traffic spikes

Example with traffic patterns:
```bash
pipegen run --message-rate 100 --duration 10m \
  --traffic-pattern "2m-4m:400%,6m-8m:300%" \
  --dashboard
```

The dashboard will show:
- Base rate: 100 msg/sec (0-2m, 4-6m, 8-10m)
- Peak 1: 400 msg/sec (2-4m)  
- Peak 2: 300 msg/sec (6-8m)

### 🔍 System Health Monitoring

**Kafka Metrics**
- Broker availability and health
- Topic partition details
- Consumer lag tracking
- Offset commit rates

**Flink Metrics**  
- Job status and health
- TaskManager availability
- Checkpoint statistics
- Backpressure indicators
- CPU/Memory utilization

**Schema Registry**
- Schema validation rates
- Registry connectivity  
- Version compatibility
- Serialization errors

### ⚠️ Error Management

**Real-Time Error Detection**
- Immediate notification of issues
- Component-specific error tracking
- Severity classification
- Historical error trends

**Error Categories:**
- **Connection errors** (Kafka, Flink, Schema Registry)
- **Serialization errors** (AVRO schema issues)
- **Processing errors** (FlinkSQL failures)  
- **Data quality errors** (validation failures)

**Resolution Suggestions**
AI-powered suggestions for common error patterns:
- Connection troubleshooting
- Configuration fixes
- Resource scaling recommendations
- Schema compatibility guidance

### 📊 Data Quality Monitoring

**Schema Validation**
- Valid vs invalid records
- Schema evolution tracking
- Compatibility checks
- Serialization success rates

**Quality Metrics**
- Record completeness scores
- Data type validation
- Business rule violations
- Anomaly detection

## Dashboard URLs

When running, the dashboard provides multiple endpoints:

| Endpoint | Purpose | Description |
|----------|---------|-------------|
| `/` | Main Dashboard | Interactive real-time monitoring |
| `/report` | HTML Report | Comprehensive pipeline report |
| `/diagram` | Pipeline Diagram | ASCII and visual architecture |
| `/api/metrics` | Raw Metrics | JSON API for custom integrations |
| `/api/export` | Export Report | Download HTML report |

### API Integration

Access metrics programmatically:

```bash
# Get current metrics
curl http://localhost:3000/api/metrics

# Export report
curl http://localhost:3000/api/export -o pipeline-report.html
```

## Report Generation

### Automatic Reports

The dashboard automatically generates comprehensive HTML reports containing:

- **Executive Summary** with key metrics
- **Performance Charts** with interactive visualizations  
- **Pipeline Diagrams** showing architecture
- **Error Analysis** with resolution recommendations
- **Traffic Pattern Analysis** (if applicable)
- **Resource Utilization** trends

### Manual Report Generation

```bash
# Generate report during run
pipegen run --generate-report --reports-dir ./my-reports

# Custom report location
pipegen run --dashboard --reports-dir /path/to/reports
```

Reports are saved with timestamps: `execution-report-2025-08-26-14-30-15.html`

### Report Features

- **Professional styling** matching the dashboard theme
- **Interactive charts** using Chart.js
- **Print-friendly** formatting
- **Shareable HTML** files
- **Embedded logos** and branding

## Advanced Features

### WebSocket Integration

The dashboard uses WebSocket for real-time communication:

```javascript
// Dashboard connects to WebSocket endpoint
ws://localhost:3000/ws/metrics

// Receives updates every second
{
  "timestamp": "2025-08-26T14:30:15Z",
  "producer": {
    "rate": 150,
    "total": 9000,
    "errors": 0
  },
  "consumer": {
    "rate": 148,
    "lag": 2,
    "total": 8880
  },
  "kafka": {
    "topics": 3,
    "partitions": 12,
    "brokers": 1
  }
}
```

### Custom Styling

The dashboard supports custom themes and branding:

```yaml
# .pipegen.yaml
dashboard:
  theme: "dark" # or "light"
  brand_color: "#1890ff"
  logo_url: "/custom-logo.png"
```

### Multi-Pipeline Monitoring

Monitor multiple pipelines simultaneously:

```bash
# Terminal 1
pipegen run --dashboard-port 3001 --project-dir ./pipeline-1

# Terminal 2  
pipegen run --dashboard-port 3002 --project-dir ./pipeline-2

# Terminal 3 - Aggregate dashboard
pipegen dashboard --port 3000 --aggregate-ports 3001,3002
```

## Performance Considerations

### Resource Usage
- **Low overhead**: ~1-2% CPU impact
- **Memory efficient**: ~10MB RAM usage
- **Network minimal**: WebSocket compression enabled

### Scaling
- **High-frequency updates**: Handles up to 10,000 msg/sec monitoring
- **Concurrent users**: Supports multiple dashboard viewers
- **Large pipelines**: Efficiently handles complex topologies

## Troubleshooting

### Common Issues

**Dashboard not opening:**
```bash
# Check if port is available
lsof -i :3000

# Use custom port
pipegen run --dashboard --dashboard-port 8080
```

**WebSocket connection issues:**
```bash
# Check firewall settings
# Verify no proxy interference  
# Try different browser
```

**Slow dashboard performance:**
```bash
# Reduce update frequency
export PIPEGEN_DASHBOARD_UPDATE_INTERVAL=5s

# Disable detailed metrics
export PIPEGEN_DASHBOARD_MINIMAL_MODE=true
```

### Browser Compatibility

Supported browsers:
- **Chrome/Chromium** 88+
- **Firefox** 85+
- **Safari** 14+
- **Edge** 88+

WebSocket and modern JavaScript required.

## Integration Examples

### CI/CD Integration
```bash
# Automated testing with report generation
pipegen run --duration 5m --dashboard \
  --generate-report --reports-dir ./ci-reports \
  --traffic-pattern "1m-2m:200%"

# Upload report to artifacts
# Fail build if error rate > 1%
```

### Monitoring Existing Infrastructure
```bash
# Monitor production Kafka/Flink
pipegen dashboard --standalone \
  --bootstrap-servers prod-kafka:9092 \
  --flink-url https://flink.company.com:8081
```

### Custom Integrations
```bash
# Export metrics to monitoring systems
while true; do
  curl -s http://localhost:3000/api/metrics | \
    jq '.producer.rate' | \
    curl -X POST https://monitoring.company.com/metrics \
    -H "Content-Type: application/json" -d @-
  sleep 30
done
```

## Next Steps

- **[Traffic Patterns](./traffic-patterns)** - Learn about load testing
- **[Commands](./commands/run)** - Explore run command options
- **[Examples](./examples)** - See dashboard in action
- **[Performance Tuning](./advanced/performance)** - Optimize for monitoring

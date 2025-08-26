# pipegen dashboard

The `dashboard` command starts a web-based dashboard for monitoring and managing your streaming pipelines.

## Usage

```bash
pipegen dashboard [flags]
```

## Examples

```bash
# Start dashboard on default port (8080)
pipegen dashboard

# Start on custom port
pipegen dashboard --port 9000

# Start with specific host binding
pipegen dashboard --host 0.0.0.0 --port 8080

# Start in read-only mode
pipegen dashboard --read-only

# Start with authentication
pipegen dashboard --auth-file users.yaml
```

## Flags

- `--port` - Port to serve dashboard (default: 8080)
- `--host` - Host to bind to (default: localhost)
- `--read-only` - Enable read-only mode
- `--auth-file` - Authentication configuration file
- `--config` - Pipeline configuration file
- `--help` - Show help for dashboard command

## Features

### Pipeline Overview
- **Status Monitoring**: Real-time pipeline health
- **Performance Metrics**: Throughput, latency, errors
- **Resource Usage**: CPU, memory, network utilization
- **Traffic Patterns**: Visual representation of message flow

### Real-time Metrics
- **Message Throughput**: Messages per second
- **Processing Latency**: End-to-end latency distribution
- **Error Rates**: Error counts and percentages
- **Backpressure**: Queue depths and processing delays

### Configuration Management
- **Live Configuration**: View current pipeline settings
- **Schema Browser**: Explore input/output schemas
- **SQL Editor**: View and validate SQL transformations
- **Environment Variables**: Runtime configuration values

### Historical Data
- **Execution Reports**: Past pipeline runs
- **Performance Trends**: Historical metrics and trends
- **Error Analysis**: Error patterns and root causes
- **Capacity Planning**: Resource usage over time

## Dashboard Interface

### Main Dashboard
```
┌─────────────────────────────────────────┐
│ PipeGen Dashboard                       │
├─────────────────────────────────────────┤
│ Pipeline Status: ● RUNNING              │
│ Messages/sec: 1,247                     │
│ Avg Latency: 125ms                      │
│ Error Rate: 0.02%                       │
├─────────────────────────────────────────┤
│ [Traffic Pattern Graph]                 │
│ [Performance Metrics Chart]             │
│ [Error Rate Timeline]                   │
└─────────────────────────────────────────┘
```

### Navigation Menu
- **Overview**: Pipeline status and key metrics
- **Metrics**: Detailed performance charts
- **Logs**: Real-time log streaming
- **Configuration**: Pipeline and system settings
- **Reports**: Historical execution data
- **Health**: System health and alerts

## Authentication

Configure user authentication with `users.yaml`:

```yaml
users:
  admin:
    password: "hashed_password"
    role: admin
  viewer:
    password: "hashed_password"
    role: readonly

roles:
  admin:
    permissions:
      - read
      - write
      - admin
  readonly:
    permissions:
      - read
```

## API Endpoints

The dashboard exposes REST API endpoints:

### Metrics
```bash
# Get current metrics
curl http://localhost:8080/api/v1/metrics

# Get historical data
curl http://localhost:8080/api/v1/metrics/history?hours=24
```

### Pipeline Control
```bash
# Get pipeline status
curl http://localhost:8080/api/v1/pipeline/status

# Start/stop pipeline (admin only)
curl -X POST http://localhost:8080/api/v1/pipeline/start
curl -X POST http://localhost:8080/api/v1/pipeline/stop
```

### Configuration
```bash
# Get current configuration
curl http://localhost:8080/api/v1/config

# Update configuration (admin only)
curl -X PUT http://localhost:8080/api/v1/config \
  -H "Content-Type: application/json" \
  -d @new-config.json
```

## Monitoring Integration

### Prometheus Metrics
The dashboard exposes metrics in Prometheus format:

```bash
curl http://localhost:8080/metrics
```

Available metrics:
- `pipegen_messages_total`
- `pipegen_processing_latency_seconds`
- `pipegen_error_rate`
- `pipegen_memory_usage_bytes`

### Grafana Integration
Import the provided Grafana dashboard:

```json
{
  "dashboard": {
    "title": "PipeGen Pipeline Metrics",
    "panels": [
      {
        "title": "Message Throughput",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(pipegen_messages_total[5m])"
          }
        ]
      }
    ]
  }
}
```

## Alerting

Configure alerts for critical conditions:

```yaml
alerts:
  high_error_rate:
    condition: "error_rate > 0.05"
    duration: "5m"
    action: "email"
    recipients: ["admin@company.com"]
  
  low_throughput:
    condition: "throughput < 100"
    duration: "10m"
    action: "slack"
    channel: "#alerts"
```

## Security

### Network Security
- **HTTPS**: Enable TLS encryption
- **IP Filtering**: Restrict access by IP address
- **Reverse Proxy**: Use nginx or similar for production

### Authentication
- **Basic Auth**: Username/password authentication
- **OAuth**: Integration with OAuth providers
- **JWT**: Token-based authentication
- **RBAC**: Role-based access control

## Performance

### Resource Usage
- **Memory**: ~50MB base usage
- **CPU**: Minimal overhead for monitoring
- **Network**: Depends on metric collection frequency
- **Storage**: Historical data storage requirements

### Optimization
- **Sampling**: Reduce metric collection frequency
- **Compression**: Enable gzip compression
- **Caching**: Cache static dashboard assets
- **CDN**: Use CDN for asset delivery

## Troubleshooting

### Common Issues

**Dashboard Not Loading**
```
Error: Connection refused
Solution: Check port binding and firewall rules
```

**Missing Metrics**
```
Issue: No data in charts
Solution: Verify pipeline is running and producing metrics
```

**High Memory Usage**
```
Issue: Dashboard consuming too much memory
Solution: Reduce historical data retention period
```

## Configuration

Dashboard-specific configuration in `dashboard.yaml`:

```yaml
dashboard:
  port: 8080
  host: "0.0.0.0"
  read_only: false
  
metrics:
  collection_interval: "10s"
  retention_period: "7d"
  
ui:
  theme: "light"
  auto_refresh: "30s"
  
auth:
  enabled: true
  file: "users.yaml"
```

## See Also

- [Metrics and Monitoring](../dashboard.md)
- [Configuration](../configuration.md)
- [pipegen run](./run.md)

# Execution Reports

PipeGen automatically generates comprehensive HTML reports for every pipeline execution, providing detailed insights into performance, configuration, and execution metrics.

## Overview

Execution reports are professional, interactive HTML documents that capture:

- **Pipeline execution metrics** (throughput, latency, duration)
- **Configuration snapshot** (all parameters used during execution)
- **Performance analytics** with interactive charts
- **System resource utilization**
- **Error tracking and analysis**
- **Traffic pattern analysis** (when applicable)

## Report Generation

### Automatic Generation

Reports are automatically generated for every `pipegen run` execution:

```bash
pipegen run --duration 30s
# Report automatically saved to ./reports/
```

### Custom Report Location

Specify a custom directory for reports:

```bash
pipegen run --duration 30s --reports-dir ./my-reports
```

### Report Naming

Reports are saved with timestamps for easy identification:
```
pipegen-execution-report-YYYYMMDD-HHMMSS.html
```

Example:
```
reports/pipegen-execution-report-20250903-141842.html
```

## Report Contents

### Executive Summary

- **Execution Status**: Success/failure indication
- **Key Metrics**: Total messages, duration, average throughput
- **Performance Highlights**: Peak rates, latencies
- **Error Summary**: Count and categorization of any issues

### Pipeline Configuration

Complete snapshot of all configuration used during execution:

- **Pipeline Settings**: Duration, cleanup options, local mode
- **Kafka Configuration**: Broker URLs, topic names, partitions
- **Schema Registry**: URL and schema validation settings
- **Resource Limits**: Memory, CPU, network settings

### Performance Analytics

Interactive charts powered by Chart.js:

- **Throughput Over Time**: Producer and consumer rates
- **Latency Percentiles**: P50, P95, P99 latency analysis
- **Message Volume**: Cumulative message counts
- **Error Rate Timeline**: Error frequency over time

### Traffic Pattern Analysis

When using traffic patterns, reports include:

- **Pattern Progression**: Visual timeline of pattern phases
- **Peak Performance**: Maximum rates achieved during spikes
- **Pattern Effectiveness**: How well the pipeline handled load variations

### System Health

- **Resource Utilization**: CPU, memory, and network usage
- **Component Status**: Kafka, Flink, Schema Registry health
- **Connection Metrics**: Latency to external services

## Using Reports

### Viewing Reports

Reports are self-contained HTML files that can be:

- **Opened in any web browser** for interactive viewing
- **Shared with team members** via email or file sharing
- **Archived for historical analysis**
- **Embedded in documentation** or presentations

### Report Location

By default, reports are saved in a `reports/` directory within your pipeline project:

```
your-pipeline-project/
├── reports/
│   ├── pipegen-execution-report-20250903-141842.html
│   ├── pipegen-execution-report-20250903-125153.html
│   └── pipegen-execution-report-20250902-225321.html
├── sql/
├── schemas/
└── docker-compose.yml
```

### Professional Styling

Reports feature:

- **Clean, professional design** suitable for stakeholder sharing
- **Interactive charts** that respond to user interaction
- **Print-friendly formatting** for hard copies
- **Mobile-responsive layout** for viewing on any device
- **PipeGen branding** with links to project documentation

## Integration with Commands

### Run Command

The `pipegen run` command automatically generates reports:

```bash
# Basic execution with automatic report
pipegen run --duration 10s

# Custom report directory
pipegen run --duration 30s --reports-dir /path/to/reports

# With cleanup disabled (preserves all execution data)
pipegen run --duration 10s --cleanup=false
```

### Configuration

Control report generation through configuration:

```yaml
# .pipegen.yaml
reports:
  enabled: true
  directory: "./execution-reports"
  include_charts: true
  include_config: true
```

## Best Practices

### Report Management

- **Regular cleanup**: Archive old reports to prevent disk space issues
- **Meaningful names**: Use descriptive report directories for different environments
- **Version control**: Consider excluding reports from git (add to `.gitignore`)

### Analysis Workflow

1. **Run pipeline** with appropriate duration for meaningful metrics
2. **Review report** immediately after execution
3. **Compare reports** across different runs to identify trends
4. **Share findings** with team using the HTML reports
5. **Archive reports** by environment or time period

### Performance Monitoring

Use reports to:

- **Establish baselines** for normal performance
- **Identify degradation** by comparing historical reports
- **Validate optimizations** by measuring before/after metrics
- **Document system behavior** for capacity planning

## Example Report Structure

A typical report includes these sections:

```
📊 PipeGen Execution Report
├── 📈 Executive Summary
│   ├── Execution Status: ✅ Success
│   ├── Duration: 30.2 seconds
│   ├── Total Messages: 4,521
│   └── Avg Throughput: 149.7 msg/sec
├── ⚙️ Pipeline Configuration
│   ├── Kafka Broker: localhost:9093
│   ├── Schema Registry: localhost:8082
│   ├── Project Directory: ./test-pipeline
│   └── Cleanup on Exit: false
├── 📊 Performance Charts
│   ├── Throughput Timeline
│   ├── Latency Analysis
│   └── Error Rate Tracking
└── 🔍 System Details
    ├── Component Health
    ├── Resource Usage
    └── Execution Environment
```

## Troubleshooting

### Report Not Generated

If reports aren't being generated:

1. **Check permissions** on the reports directory
2. **Verify disk space** is available
3. **Review logs** for template or file system errors
4. **Ensure cleanup** isn't removing reports prematurely

### Report Display Issues

For display problems:

1. **Use modern browser** (Chrome, Firefox, Safari, Edge)
2. **Enable JavaScript** for interactive charts
3. **Check file integrity** if report seems corrupted
4. **Clear browser cache** if styles don't load properly

## See Also

- [Run Command Documentation](../commands/run.md) - Detailed run command options
- [Configuration Guide](../configuration.md) - Pipeline configuration options
- [Traffic Patterns](../traffic-patterns.md) - Understanding traffic pattern analysis in reports
- [Performance Tuning](../advanced/performance.md) - Using reports for optimization

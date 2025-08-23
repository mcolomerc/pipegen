# Execution Reports Feature

The execution reports feature generates comprehensive HTML reports for pipeline executions, providing detailed insights into performance, metrics, and execution parameters.

## Overview

The execution report system consists of:

1. **ExecutionDataCollector** - Collects metrics during pipeline execution
2. **ExecutionReportGenerator** - Generates HTML reports with charts and visualizations
3. **ExecutionReport** - Data structure containing all execution information
4. **Integration** - Seamless integration with the pipeline runner

## Features

### 📊 Comprehensive Metrics
- **Message Processing**: Total messages, throughput (messages/second)
- **Performance**: Latency measurements, success rate, error counts
- **Resource Usage**: Bytes processed, memory usage
- **Producer/Consumer Stats**: Detailed metrics from both components

### 📈 Interactive Charts
- **Messages Over Time**: Real-time visualization of message processing
- **Throughput Analysis**: Messages per second over execution duration
- **Latency Trends**: Response time monitoring
- **Error Rate Tracking**: Error occurrence patterns

### 🎨 Dashboard Theme
- Consistent look and feel with the live dashboard
- Responsive design for desktop and mobile
- Professional color scheme matching PipeGen branding
- Modern UI with FontAwesome icons and Chart.js

### ⚙️ Execution Parameters
- Complete parameter tracking (message rate, duration, URLs)
- Environment configuration (local vs. cloud mode)
- Resource management settings (cleanup, project directory)

## Usage

### Command Line

Enable report generation when running pipelines:

```bash
# Basic usage
pipegen run --generate-report

# Custom reports directory
pipegen run --generate-report --reports-dir ./my-reports

# With dashboard integration
pipegen run --dashboard --generate-report
```

### Programmatic Usage

```go
// Create report generator
reportsDir := "./reports"
generator := dashboard.NewExecutionReportGenerator(reportsDir)

// Create data collector
executionID := "my-execution-123"
params := dashboard.ExecutionParameters{
    MessageRate:       1000,
    Duration:          5 * time.Minute,
    BootstrapServers:  "localhost:9092",
    // ... other parameters
}
collector := dashboard.NewExecutionDataCollector(executionID, params)

// Update metrics during execution
collector.UpdateMetrics(producerStats, consumerStats)
collector.AddLatencyPoint(latency)

// Generate final report
report := collector.GetCurrentReport("My Pipeline", "v1.0.0")
reportPath, err := generator.GenerateReport(report)
```

## Report Structure

### Header Section
- Pipeline name and version
- Execution timestamp and duration  
- Execution status badge
- Unique execution ID

### Summary Statistics
- Large metric cards showing key performance indicators
- Success rate calculation
- Error count and handling

### Parameters Section
- Grid layout of all execution parameters
- Environment configuration display
- Resource management settings

### Performance Metrics
- Detailed producer/consumer statistics
- Latency measurements
- Throughput analysis
- Bytes processed

### Charts and Visualizations
- Interactive time-series charts using Chart.js
- Messages processed over time
- Throughput trends
- Latency analysis

## File Organization

Reports are saved with timestamped filenames:
```
reports/
├── pipegen-execution-report-20240102-143052.html
├── pipegen-execution-report-20240102-145123.html
└── pipegen-execution-report-20240103-091234.html
```

## Configuration

### Default Behavior
- Reports are saved to `{project-dir}/reports/`
- Timestamped filenames prevent overwrites
- HTML format with embedded styles and scripts

### Customization Options
- Custom reports directory via `--reports-dir` flag
- Integration with dashboard for enhanced metrics
- Configurable chart update intervals

## Integration Points

### Pipeline Runner
- Automatic integration when `GenerateReport: true`
- Metrics collection during execution
- Report generation on completion

### Dashboard Integration
- Enhanced metrics when dashboard is enabled
- Real-time data collection
- Consistent styling and branding

### Error Handling
- Graceful degradation if report generation fails
- Non-blocking execution (warnings only)
- Detailed error messages for troubleshooting

## Technical Details

### Data Collection
- Thread-safe metrics collection
- Configurable collection intervals
- Memory-efficient data point storage

### Report Generation
- Server-side HTML template rendering
- Embedded CSS and JavaScript (no external dependencies)
- Mobile-responsive design

### File Management
- Automatic directory creation
- Unique filename generation
- Proper file permissions (0644)

## Example Report Contents

A typical execution report includes:

1. **Execution Summary**
   - 1,000 total messages processed
   - 167 messages/second average throughput
   - 98.5% success rate
   - 15 errors encountered

2. **Performance Charts**
   - Message processing timeline
   - Throughput fluctuations
   - Latency distribution

3. **Configuration Details**
   - All runtime parameters
   - Environment settings
   - Resource configurations

## Browser Compatibility

Reports are compatible with:
- Chrome/Edge (latest)
- Firefox (latest)
- Safari (latest)
- Mobile browsers (iOS Safari, Android Chrome)

The reports use standard HTML5, CSS3, and ES6 JavaScript features with CDN-hosted libraries for maximum compatibility.

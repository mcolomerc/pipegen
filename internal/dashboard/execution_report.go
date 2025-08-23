package dashboard

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExecutionReport represents a complete pipeline execution report
type ExecutionReport struct {
	Timestamp       time.Time
	ExecutionID     string
	Parameters      ExecutionParameters
	Metrics         ExecutionMetrics
	Summary         ExecutionSummary
	Charts          ChartData
	Status          string
	Duration        time.Duration
	PipelineName    string
	PipelineVersion string
}

// ExecutionParameters holds the parameters used for the execution
type ExecutionParameters struct {
	MessageRate       int           `json:"message_rate"`
	Duration          time.Duration `json:"duration"`
	BootstrapServers  string        `json:"bootstrap_servers"`
	FlinkURL          string        `json:"flink_url"`
	SchemaRegistryURL string        `json:"schema_registry_url"`
	LocalMode         bool          `json:"local_mode"`
	ProjectDir        string        `json:"project_dir"`
	Cleanup           bool          `json:"cleanup"`
}

// ExecutionMetrics holds detailed metrics from the execution
type ExecutionMetrics struct {
	TotalMessages     int64           `json:"total_messages"`
	MessagesPerSecond float64         `json:"messages_per_second"`
	BytesProcessed    int64           `json:"bytes_processed"`
	ErrorCount        int64           `json:"error_count"`
	SuccessRate       float64         `json:"success_rate"`
	AvgLatency        time.Duration   `json:"avg_latency"`
	ProducerMetrics   *ProducerStats  `json:"producer_metrics,omitempty"`
	ConsumerMetrics   *ConsumerStats  `json:"consumer_metrics,omitempty"`
	FlinkMetrics      *FlinkMetrics   `json:"flink_metrics,omitempty"`
}

// ChartData holds data for visualization charts
type ChartData struct {
	MessagesOverTime    []TimeSeriesPoint `json:"messages_over_time"`
	ThroughputOverTime  []TimeSeriesPoint `json:"throughput_over_time"`
	LatencyOverTime     []TimeSeriesPoint `json:"latency_over_time"`
	ErrorRateOverTime   []TimeSeriesPoint `json:"error_rate_over_time"`
}

// TimeSeriesPoint represents a data point in time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// ExecutionReportGenerator creates HTML reports for pipeline executions
type ExecutionReportGenerator struct {
	reportsDir string
}

// NewExecutionReportGenerator creates a new report generator
func NewExecutionReportGenerator(reportsDir string) *ExecutionReportGenerator {
	return &ExecutionReportGenerator{
		reportsDir: reportsDir,
	}
}

// GenerateReport creates a complete execution report and saves it to disk
func (g *ExecutionReportGenerator) GenerateReport(report *ExecutionReport) (string, error) {
	// Ensure reports directory exists
	if err := os.MkdirAll(g.reportsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Generate unique filename with timestamp
	filename := fmt.Sprintf("pipegen-execution-report-%s.html", 
		report.Timestamp.Format("20060102-150405"))
	filepath := filepath.Join(g.reportsDir, filename)

	// Generate HTML content
	htmlContent, err := g.renderReportHTML(report)
	if err != nil {
		return "", fmt.Errorf("failed to render HTML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, []byte(htmlContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write report file: %w", err)
	}

	return filepath, nil
}

// renderReportHTML generates the complete HTML report
func (g *ExecutionReportGenerator) renderReportHTML(report *ExecutionReport) (string, error) {
	tmpl := template.Must(template.New("execution-report").Parse(g.getReportTemplate()))
	
	var buf strings.Builder
	if err := tmpl.Execute(&buf, report); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// getReportTemplate returns the HTML template with the same theme as dashboard
func (g *ExecutionReportGenerator) getReportTemplate() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PipeGen Execution Report - {{.ExecutionID}}</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.5.1/css/all.min.css">
    <style>
        :root {
            --primary-color: #ff6b35;      /* Squirrel Orange */
            --secondary-color: #cc3333;    /* Toolbox Red */
            --accent-color: #666666;       /* Metal Gray */
            --dark-color: #2c3e50;         /* Dark Base */
            --bg-light: #f8f9fa;           /* Background Light */
            --text-primary: #2c3e50;
            --text-secondary: #666666;
            --border-color: #e0e6ed;
            --success-color: #28a745;
            --warning-color: #ffc107;
            --danger-color: #dc3545;
            --info-color: #17a2b8;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background-color: var(--bg-light);
            color: var(--text-primary);
            line-height: 1.6;
        }

        .container {
            max-width: 1400px;
            margin: 0 auto;
            padding: 1rem;
            gap: 1rem;
            display: flex;
            flex-direction: column;
        }

        .header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            background: linear-gradient(135deg, var(--primary-color), var(--secondary-color));
            color: white;
            padding: 2rem;
            border-radius: 12px;
            box-shadow: 0 4px 20px rgba(255, 107, 53, 0.3);
        }

        .header h1 {
            font-size: 2rem;
            font-weight: 700;
            display: flex;
            align-items: center;
            gap: 1rem;
        }

        .header .execution-info {
            text-align: right;
            opacity: 0.9;
        }

        .execution-info .timestamp {
            font-size: 1.1rem;
            margin-bottom: 0.5rem;
        }

        .execution-info .duration {
            font-size: 0.9rem;
        }

        .cards-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 1.5rem;
            margin-top: 1rem;
        }

        .card {
            background: white;
            border-radius: 12px;
            padding: 1.5rem;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
            border: 1px solid var(--border-color);
        }

        .card h3 {
            font-size: 1.2rem;
            margin-bottom: 1rem;
            color: var(--primary-color);
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        .metric-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 0.75rem 0;
            border-bottom: 1px solid #f0f0f0;
        }

        .metric-row:last-child {
            border-bottom: none;
        }

        .metric-label {
            font-weight: 500;
            color: var(--text-secondary);
        }

        .metric-value {
            font-weight: 600;
            font-size: 1.1rem;
        }

        .status-badge {
            display: inline-block;
            padding: 0.25rem 0.75rem;
            border-radius: 20px;
            font-size: 0.85rem;
            font-weight: 600;
            text-transform: uppercase;
        }

        .status-success {
            background-color: #d4edda;
            color: var(--success-color);
        }

        .status-error {
            background-color: #f8d7da;
            color: var(--danger-color);
        }

        .status-warning {
            background-color: #fff3cd;
            color: var(--warning-color);
        }

        .chart-container {
            position: relative;
            height: 400px;
            margin: 1rem 0;
        }

        .large-card {
            grid-column: 1 / -1;
        }

        .parameters-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1rem;
        }

        .parameter-item {
            background: var(--bg-light);
            padding: 1rem;
            border-radius: 8px;
            border-left: 4px solid var(--primary-color);
        }

        .parameter-label {
            font-weight: 600;
            color: var(--text-secondary);
            font-size: 0.9rem;
            margin-bottom: 0.25rem;
        }

        .parameter-value {
            font-family: 'Courier New', monospace;
            font-size: 1rem;
            word-break: break-all;
        }

        .summary-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
        }

        .stat-box {
            text-align: center;
            padding: 1.5rem;
            background: var(--bg-light);
            border-radius: 8px;
            border: 1px solid var(--border-color);
        }

        .stat-number {
            font-size: 2rem;
            font-weight: 700;
            color: var(--primary-color);
            margin-bottom: 0.5rem;
        }

        .stat-label {
            color: var(--text-secondary);
            font-size: 0.9rem;
        }

        .footer {
            text-align: center;
            padding: 2rem;
            color: var(--text-secondary);
            border-top: 1px solid var(--border-color);
            margin-top: 2rem;
        }

        @media (max-width: 768px) {
            .header {
                flex-direction: column;
                text-align: center;
                gap: 1rem;
            }

            .cards-grid {
                grid-template-columns: 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <!-- Header -->
        <div class="header">
            <div>
                <h1>
                    <i class="fas fa-chart-line"></i>
                    Execution Report
                </h1>
                <div>{{.PipelineName}} {{.PipelineVersion}}</div>
            </div>
            <div class="execution-info">
                <div class="timestamp">{{.Timestamp.Format "2006-01-02 15:04:05 MST"}}</div>
                <div class="duration">Duration: {{.Duration}}</div>
                <div class="status-badge status-{{if eq .Status "completed"}}success{{else if eq .Status "failed"}}error{{else}}warning{{end}}">
                    {{.Status}}
                </div>
            </div>
        </div>

        <div class="cards-grid">
            <!-- Execution Summary -->
            <div class="card large-card">
                <h3><i class="fas fa-tachometer-alt"></i> Execution Summary</h3>
                <div class="summary-stats">
                    <div class="stat-box">
                        <div class="stat-number">{{.Metrics.TotalMessages}}</div>
                        <div class="stat-label">Total Messages</div>
                    </div>
                    <div class="stat-box">
                        <div class="stat-number">{{printf "%.1f" .Metrics.MessagesPerSecond}}</div>
                        <div class="stat-label">Messages/Second</div>
                    </div>
                    <div class="stat-box">
                        <div class="stat-number">{{printf "%.1f%%" .Metrics.SuccessRate}}</div>
                        <div class="stat-label">Success Rate</div>
                    </div>
                    <div class="stat-box">
                        <div class="stat-number">{{.Metrics.ErrorCount}}</div>
                        <div class="stat-label">Errors</div>
                    </div>
                </div>
            </div>

            <!-- Parameters Used -->
            <div class="card">
                <h3><i class="fas fa-cogs"></i> Execution Parameters</h3>
                <div class="parameters-grid">
                    <div class="parameter-item">
                        <div class="parameter-label">Message Rate</div>
                        <div class="parameter-value">{{.Parameters.MessageRate}} msg/sec</div>
                    </div>
                    <div class="parameter-item">
                        <div class="parameter-label">Duration</div>
                        <div class="parameter-value">{{.Parameters.Duration}}</div>
                    </div>
                    <div class="parameter-item">
                        <div class="parameter-label">Bootstrap Servers</div>
                        <div class="parameter-value">{{.Parameters.BootstrapServers}}</div>
                    </div>
                    <div class="parameter-item">
                        <div class="parameter-label">Flink URL</div>
                        <div class="parameter-value">{{.Parameters.FlinkURL}}</div>
                    </div>
                    <div class="parameter-item">
                        <div class="parameter-label">Local Mode</div>
                        <div class="parameter-value">{{if .Parameters.LocalMode}}Yes{{else}}No{{end}}</div>
                    </div>
                    <div class="parameter-item">
                        <div class="parameter-label">Cleanup</div>
                        <div class="parameter-value">{{if .Parameters.Cleanup}}Yes{{else}}No{{end}}</div>
                    </div>
                </div>
            </div>

            <!-- Detailed Metrics -->
            <div class="card">
                <h3><i class="fas fa-chart-bar"></i> Performance Metrics</h3>
                {{if .Metrics.ProducerMetrics}}
                <div class="metric-row">
                    <span class="metric-label">Messages Sent</span>
                    <span class="metric-value">{{.Metrics.ProducerMetrics.MessagesSent}}</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Bytes Sent</span>
                    <span class="metric-value">{{.Metrics.ProducerMetrics.BytesSent}} B</span>
                </div>
                {{end}}
                {{if .Metrics.ConsumerMetrics}}
                <div class="metric-row">
                    <span class="metric-label">Messages Consumed</span>
                    <span class="metric-value">{{.Metrics.ConsumerMetrics.MessagesConsumed}}</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Bytes Consumed</span>
                    <span class="metric-value">{{.Metrics.ConsumerMetrics.BytesConsumed}} B</span>
                </div>
                {{end}}
                <div class="metric-row">
                    <span class="metric-label">Average Latency</span>
                    <span class="metric-value">{{.Metrics.AvgLatency}}</span>
                </div>
                <div class="metric-row">
                    <span class="metric-label">Bytes Processed</span>
                    <span class="metric-value">{{.Metrics.BytesProcessed}} B</span>
                </div>
            </div>
        </div>

        <!-- Charts Section -->
        {{if .Charts.MessagesOverTime}}
        <div class="card large-card">
            <h3><i class="fas fa-chart-line"></i> Messages Processed Over Time</h3>
            <div class="chart-container">
                <canvas id="messagesChart"></canvas>
            </div>
        </div>
        {{end}}

        {{if .Charts.ThroughputOverTime}}
        <div class="card large-card">
            <h3><i class="fas fa-tachometer-alt"></i> Throughput Over Time</h3>
            <div class="chart-container">
                <canvas id="throughputChart"></canvas>
            </div>
        </div>
        {{end}}

        <!-- Footer -->
        <div class="footer">
            <p>Generated by PipeGen v{{.PipelineVersion}} on {{.Timestamp.Format "2006-01-02 15:04:05 MST"}}</p>
            <p>Execution ID: {{.ExecutionID}}</p>
        </div>
    </div>

    <script>
        // Chart.js configuration
        const chartOptions = {
            responsive: true,
            maintainAspectRatio: false,
            plugins: {
                legend: {
                    position: 'top',
                },
                title: {
                    display: false
                }
            },
            scales: {
                x: {
                    type: 'time',
                    time: {
                        displayFormats: {
                            second: 'HH:mm:ss'
                        }
                    }
                },
                y: {
                    beginAtZero: true
                }
            }
        };

        {{if .Charts.MessagesOverTime}}
        // Messages Over Time Chart
        const messagesCtx = document.getElementById('messagesChart').getContext('2d');
        new Chart(messagesCtx, {
            type: 'line',
            data: {
                datasets: [{
                    label: 'Messages Processed',
                    data: [
                        {{range .Charts.MessagesOverTime}}
                        {x: '{{.Timestamp.Format "2006-01-02T15:04:05Z"}}', y: {{.Value}}},
                        {{end}}
                    ],
                    borderColor: 'rgb(255, 107, 53)',
                    backgroundColor: 'rgba(255, 107, 53, 0.1)',
                    tension: 0.4
                }]
            },
            options: chartOptions
        });
        {{end}}

        {{if .Charts.ThroughputOverTime}}
        // Throughput Over Time Chart
        const throughputCtx = document.getElementById('throughputChart').getContext('2d');
        new Chart(throughputCtx, {
            type: 'line',
            data: {
                datasets: [{
                    label: 'Messages/Second',
                    data: [
                        {{range .Charts.ThroughputOverTime}}
                        {x: '{{.Timestamp.Format "2006-01-02T15:04:05Z"}}', y: {{.Value}}},
                        {{end}}
                    ],
                    borderColor: 'rgb(204, 51, 51)',
                    backgroundColor: 'rgba(204, 51, 51, 0.1)',
                    tension: 0.4
                }]
            },
            options: chartOptions
        });
        {{end}}
    </script>
</body>
</html>`
}
`

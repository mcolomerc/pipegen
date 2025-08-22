package dashboard

import (
	"fmt"
	"html/template"
	"strings"
)

// GenerateHTMLReport creates a comprehensive HTML report
func GenerateHTMLReport(status *PipelineStatus) (string, error) {
	reportTemplate := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>PipeGen Pipeline Report - {{.ExecutionSummary.TotalMessagesProcessed}} Messages Processed</title>
    <style>
        :root {
            --primary-color: #2563eb;
            --success-color: #10b981;
            --warning-color: #f59e0b;
            --error-color: #ef4444;
            --bg-color: #f8fafc;
            --card-bg: white;
            --text-primary: #1f2937;
            --text-secondary: #6b7280;
            --border-color: #e5e7eb;
        }
        
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
            background-color: var(--bg-color);
            color: var(--text-primary);
            line-height: 1.6;
        }
        
        .header {
            background: linear-gradient(135deg, var(--primary-color), #1d4ed8);
            color: white;
            padding: 2rem 0;
            box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        }
        
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 0 1rem;
        }
        
        .header-content {
            text-align: center;
        }
        
        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 0.5rem;
            font-weight: 700;
        }
        
        .header .subtitle {
            font-size: 1.2rem;
            opacity: 0.9;
        }
        
        .meta-info {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin: 1rem 0;
            text-align: left;
        }
        
        .meta-item {
            background: rgba(255, 255, 255, 0.1);
            padding: 1rem;
            border-radius: 8px;
        }
        
        .meta-label {
            font-size: 0.875rem;
            opacity: 0.8;
            margin-bottom: 0.25rem;
        }
        
        .meta-value {
            font-size: 1.125rem;
            font-weight: 600;
        }
        
        .main-content {
            padding: 2rem 0;
        }
        
        .section {
            background: var(--card-bg);
            border-radius: 12px;
            padding: 1.5rem;
            margin-bottom: 2rem;
            box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.1);
            border: 1px solid var(--border-color);
        }
        
        .section-title {
            font-size: 1.5rem;
            font-weight: 600;
            margin-bottom: 1rem;
            color: var(--text-primary);
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }
        
        .status-badge {
            padding: 0.25rem 0.75rem;
            border-radius: 9999px;
            font-size: 0.875rem;
            font-weight: 500;
        }
        
        .status-running { background-color: #dcfce7; color: #166534; }
        .status-error { background-color: #fecaca; color: #991b1b; }
        .status-warning { background-color: #fef3c7; color: #92400e; }
        .status-stopped { background-color: #f3f4f6; color: #374151; }
        
        .metrics-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 1rem;
            margin-top: 1rem;
        }
        
        .metric-card {
            background: var(--bg-color);
            padding: 1rem;
            border-radius: 8px;
            border: 1px solid var(--border-color);
        }
        
        .metric-label {
            font-size: 0.875rem;
            color: var(--text-secondary);
            margin-bottom: 0.25rem;
        }
        
        .metric-value {
            font-size: 1.5rem;
            font-weight: 700;
            color: var(--primary-color);
        }
        
        .metric-unit {
            font-size: 0.875rem;
            color: var(--text-secondary);
            font-weight: normal;
        }
        
        .diagram-container {
            background: white;
            border: 2px dashed var(--border-color);
            border-radius: 8px;
            padding: 2rem;
            text-align: center;
            margin: 1rem 0;
        }
        
        .pipeline-diagram {
            font-family: 'Courier New', monospace;
            font-size: 0.875rem;
            line-height: 1.8;
            white-space: pre;
            color: var(--text-primary);
            background: #f8fafc;
            padding: 1.5rem;
            border-radius: 6px;
            overflow-x: auto;
        }
        
        .error-list {
            margin-top: 1rem;
        }
        
        .error-item {
            background: #fef2f2;
            border-left: 4px solid var(--error-color);
            padding: 1rem;
            margin-bottom: 0.5rem;
            border-radius: 0 8px 8px 0;
        }
        
        .error-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 0.5rem;
        }
        
        .error-component {
            background: var(--error-color);
            color: white;
            padding: 0.25rem 0.5rem;
            border-radius: 4px;
            font-size: 0.75rem;
            font-weight: 500;
        }
        
        .error-timestamp {
            font-size: 0.875rem;
            color: var(--text-secondary);
        }
        
        .error-message {
            font-weight: 600;
            color: #991b1b;
            margin-bottom: 0.5rem;
        }
        
        .error-details {
            color: var(--text-secondary);
            font-size: 0.875rem;
        }
        
        .performance-chart {
            margin-top: 1rem;
            text-align: center;
            color: var(--text-secondary);
            font-style: italic;
            padding: 2rem;
            background: var(--bg-color);
            border-radius: 8px;
        }
        
        .footer {
            margin-top: 3rem;
            padding: 2rem 0;
            text-align: center;
            color: var(--text-secondary);
            border-top: 1px solid var(--border-color);
            font-size: 0.875rem;
        }
        
        .data-quality-bar {
            width: 100%;
            height: 20px;
            background: #f3f4f6;
            border-radius: 10px;
            overflow: hidden;
            margin: 0.5rem 0;
        }
        
        .quality-fill {
            height: 100%;
            background: linear-gradient(90deg, var(--error-color) 0%, var(--warning-color) 50%, var(--success-color) 100%);
            transition: width 0.3s ease;
        }
        
        .topic-table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 1rem;
        }
        
        .topic-table th,
        .topic-table td {
            padding: 0.75rem;
            text-align: left;
            border-bottom: 1px solid var(--border-color);
        }
        
        .topic-table th {
            background: var(--bg-color);
            font-weight: 600;
            color: var(--text-primary);
        }
        
        .highlight-number {
            font-weight: 700;
            color: var(--primary-color);
        }
        
        @media print {
            .header { -webkit-print-color-adjust: exact; }
            .section { break-inside: avoid; }
        }
    </style>
</head>
<body>
    <!-- Header -->
    <div class="header">
        <div class="container">
            <div class="header-content">
                <h1>🚀 Pipeline Execution Report</h1>
                <div class="subtitle">{{.Resources.Prefix}} • Generated {{.LastUpdated.Format "2006-01-02 15:04:05"}}</div>
                <div class="meta-info">
                    <div class="meta-item">
                        <div class="meta-label">Status</div>
                        <div class="meta-value">
                            <span class="status-badge status-{{.Status | lower}}">{{.Status}}</span>
                        </div>
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Duration</div>
                        <div class="meta-value">{{.Duration}}</div>
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Messages Processed</div>
                        <div class="meta-value">{{.ExecutionSummary.TotalMessagesProcessed | formatNumber}}</div>
                    </div>
                    <div class="meta-item">
                        <div class="meta-label">Throughput</div>
                        <div class="meta-value">{{.ExecutionSummary.ThroughputMsgSec | printf "%.1f"}} msg/sec</div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Main Content -->
    <div class="main-content">
        <div class="container">
            
            <!-- Pipeline Diagram -->
            <div class="section">
                <h2 class="section-title">📊 Pipeline Architecture</h2>
                <div class="diagram-container">
                    <div class="pipeline-diagram">{{generatePipelineDiagram .}}</div>
                </div>
            </div>
            
            <!-- Execution Summary -->
            <div class="section">
                <h2 class="section-title">📈 Execution Summary</h2>
                <div class="metrics-grid">
                    <div class="metric-card">
                        <div class="metric-label">Total Messages</div>
                        <div class="metric-value">{{.ExecutionSummary.TotalMessagesProcessed | formatNumber}}</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">Total Data</div>
                        <div class="metric-value">{{.ExecutionSummary.TotalBytesProcessed | formatBytes}}</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">Average Latency</div>
                        <div class="metric-value">{{.ExecutionSummary.AverageLatency}}</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">Success Rate</div>
                        <div class="metric-value">{{.ExecutionSummary.SuccessRate | printf "%.2f"}}<span class="metric-unit">%</span></div>
                    </div>
                </div>
                
                {{if .ExecutionSummary.DataQuality}}
                <h3 style="margin: 2rem 0 1rem 0; color: var(--text-primary);">Data Quality Score</h3>
                <div class="data-quality-bar">
                    <div class="quality-fill" style="width: {{.ExecutionSummary.DataQuality.QualityScore}}%;"></div>
                </div>
                <p style="margin-top: 0.5rem; color: var(--text-secondary);">
                    <span class="highlight-number">{{.ExecutionSummary.DataQuality.ValidRecords | formatNumber}}</span> valid records, 
                    <span class="highlight-number">{{.ExecutionSummary.DataQuality.InvalidRecords | formatNumber}}</span> invalid, 
                    <span class="highlight-number">{{.ExecutionSummary.DataQuality.SchemaViolations | formatNumber}}</span> schema violations
                </p>
                {{end}}
            </div>
            
            <!-- Kafka Metrics -->
            {{if .KafkaMetrics}}
            <div class="section">
                <h2 class="section-title">🔄 Kafka Metrics</h2>
                <div class="metrics-grid">
                    <div class="metric-card">
                        <div class="metric-label">Cluster Health</div>
                        <div class="metric-value">
                            <span class="status-badge status-{{.KafkaMetrics.ClusterHealth | lower}}">{{.KafkaMetrics.ClusterHealth}}</span>
                        </div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">Active Brokers</div>
                        <div class="metric-value">{{.KafkaMetrics.BrokerCount}}</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">Total Topics</div>
                        <div class="metric-value">{{.KafkaMetrics.TopicCount}}</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">Messages/sec</div>
                        <div class="metric-value">{{.KafkaMetrics.MessagesPerSec | printf "%.1f"}}</div>
                    </div>
                </div>
                
                {{if .KafkaMetrics.Topics}}
                <h3 style="margin: 2rem 0 1rem 0; color: var(--text-primary);">Topic Details</h3>
                <table class="topic-table">
                    <thead>
                        <tr>
                            <th>Topic</th>
                            <th>Partitions</th>
                            <th>Messages</th>
                            <th>Size</th>
                            <th>Produce Rate</th>
                            <th>Lag</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .KafkaMetrics.Topics}}
                        <tr>
                            <td>{{.Name}}</td>
                            <td>{{.Partitions}}</td>
                            <td>{{.MessageCount | formatNumber}}</td>
                            <td>{{.Size | formatBytes}}</td>
                            <td>{{.ProduceRate | printf "%.1f"}} msg/sec</td>
                            <td>{{.Lag | formatNumber}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
                {{end}}
            </div>
            {{end}}
            
            <!-- Flink Metrics -->
            {{if .FlinkMetrics}}
            <div class="section">
                <h2 class="section-title">⚡ Flink Metrics</h2>
                <div class="metrics-grid">
                    <div class="metric-card">
                        <div class="metric-label">JobManager Status</div>
                        <div class="metric-value">
                            <span class="status-badge status-{{.FlinkMetrics.JobManagerStatus | lower}}">{{.FlinkMetrics.JobManagerStatus}}</span>
                        </div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">Task Managers</div>
                        <div class="metric-value">{{.FlinkMetrics.TaskManagerCount}}</div>
                    </div>
                    <div class="metric-card">
                        <div class="metric-label">Active Jobs</div>
                        <div class="metric-value">{{len .FlinkMetrics.Jobs}}</div>
                    </div>
                    {{if .FlinkMetrics.ClusterMetrics}}
                    <div class="metric-card">
                        <div class="metric-label">CPU Usage</div>
                        <div class="metric-value">{{.FlinkMetrics.ClusterMetrics.CPUUsage | printf "%.1f"}}<span class="metric-unit">%</span></div>
                    </div>
                    {{end}}
                </div>
                
                {{if .FlinkMetrics.Jobs}}
                <h3 style="margin: 2rem 0 1rem 0; color: var(--text-primary);">Job Details</h3>
                <table class="topic-table">
                    <thead>
                        <tr>
                            <th>Job Name</th>
                            <th>Status</th>
                            <th>Duration</th>
                            <th>Records In</th>
                            <th>Records Out</th>
                            <th>Records/sec</th>
                        </tr>
                    </thead>
                    <tbody>
                        {{range .FlinkMetrics.Jobs}}
                        <tr>
                            <td>{{.Name}}</td>
                            <td><span class="status-badge status-{{.Status | lower}}">{{.Status}}</span></td>
                            <td>{{.Duration}}</td>
                            <td>{{.RecordsIn | formatNumber}}</td>
                            <td>{{.RecordsOut | formatNumber}}</td>
                            <td>{{.RecordsPerSec | printf "%.1f"}}</td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
                {{end}}
            </div>
            {{end}}
            
            <!-- Producer & Consumer Metrics -->
            <div class="section">
                <h2 class="section-title">📤📥 Producer & Consumer Metrics</h2>
                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 2rem;">
                    {{if .ProducerMetrics}}
                    <div>
                        <h3 style="margin-bottom: 1rem; color: var(--text-primary);">Producer</h3>
                        <div class="metrics-grid">
                            <div class="metric-card">
                                <div class="metric-label">Messages Sent</div>
                                <div class="metric-value">{{.ProducerMetrics.MessagesSent | formatNumber}}</div>
                            </div>
                            <div class="metric-card">
                                <div class="metric-label">Throughput</div>
                                <div class="metric-value">{{.ProducerMetrics.MessagesPerSec | printf "%.1f"}} <span class="metric-unit">msg/sec</span></div>
                            </div>
                            <div class="metric-card">
                                <div class="metric-label">Success Rate</div>
                                <div class="metric-value">{{.ProducerMetrics.SuccessRate | printf "%.2f"}}<span class="metric-unit">%</span></div>
                            </div>
                            <div class="metric-card">
                                <div class="metric-label">Avg Latency</div>
                                <div class="metric-value">{{.ProducerMetrics.AverageLatency}}</div>
                            </div>
                        </div>
                    </div>
                    {{end}}
                    
                    {{if .ConsumerMetrics}}
                    <div>
                        <h3 style="margin-bottom: 1rem; color: var(--text-primary);">Consumer</h3>
                        <div class="metrics-grid">
                            <div class="metric-card">
                                <div class="metric-label">Messages Consumed</div>
                                <div class="metric-value">{{.ConsumerMetrics.MessagesConsumed | formatNumber}}</div>
                            </div>
                            <div class="metric-card">
                                <div class="metric-label">Throughput</div>
                                <div class="metric-value">{{.ConsumerMetrics.MessagesPerSec | printf "%.1f"}} <span class="metric-unit">msg/sec</span></div>
                            </div>
                            <div class="metric-card">
                                <div class="metric-label">Consumer Lag</div>
                                <div class="metric-value">{{.ConsumerMetrics.Lag | formatNumber}}</div>
                            </div>
                            <div class="metric-card">
                                <div class="metric-label">Processing Time</div>
                                <div class="metric-value">{{.ConsumerMetrics.ProcessingTime}}</div>
                            </div>
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            
            <!-- Errors Section -->
            {{if .Errors}}
            <div class="section">
                <h2 class="section-title">❌ Errors & Issues</h2>
                <div class="error-list">
                    {{range .Errors}}
                    <div class="error-item">
                        <div class="error-header">
                            <span class="error-component">{{.Component}}</span>
                            <span class="error-timestamp">{{.Timestamp.Format "15:04:05"}}</span>
                        </div>
                        <div class="error-message">{{.Message}}</div>
                        {{if .Details}}
                        <div class="error-details">{{.Details}}</div>
                        {{end}}
                        {{if .Resolution}}
                        <div style="margin-top: 0.5rem; padding: 0.5rem; background: #f0f9ff; border-radius: 4px; font-size: 0.875rem; color: var(--primary-color);">
                            💡 <strong>Suggestion:</strong> {{.Resolution}}
                        </div>
                        {{end}}
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
            
        </div>
    </div>

    <!-- Footer -->
    <div class="footer">
        <div class="container">
            Report generated by PipeGen v1.0 • {{time.Now.Format "January 2, 2006 at 15:04:05"}}
        </div>
    </div>
</body>
</html>`

	// Template functions
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
		"formatNumber": func(n int64) string {
			if n >= 1_000_000 {
				return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
			} else if n >= 1_000 {
				return fmt.Sprintf("%.1fK", float64(n)/1_000)
			}
			return fmt.Sprintf("%d", n)
		},
		"formatBytes": func(bytes int64) string {
			if bytes >= 1_073_741_824 { // 1GB
				return fmt.Sprintf("%.2f GB", float64(bytes)/1_073_741_824)
			} else if bytes >= 1_048_576 { // 1MB
				return fmt.Sprintf("%.1f MB", float64(bytes)/1_048_576)
			} else if bytes >= 1_024 { // 1KB
				return fmt.Sprintf("%.1f KB", float64(bytes)/1_024)
			}
			return fmt.Sprintf("%d B", bytes)
		},
		"generatePipelineDiagram": generatePipelineDiagram,
	}

	tmpl, err := template.New("report").Funcs(funcMap).Parse(reportTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, status); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// generatePipelineDiagram creates an ASCII art diagram of the pipeline
func generatePipelineDiagram(status *PipelineStatus) string {
	diagram := `
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   📊 Producer   │───▶│  📨 Kafka Topic │───▶│  ⚡ Flink SQL   │───▶│ 📤 Output Topic │
│                 │    │                 │    │                 │    │                 │
│ Rate: %s msg/s  │    │ %s partitions   │    │ %d jobs active  │    │ %s messages     │
│ Status: %s      │    │ Health: %s      │    │ Status: %s      │    │ Lag: %s         │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
                                   │                                            │
                                   ▼                                            ▼
                       ┌─────────────────┐                          ┌─────────────────┐
                       │ 🔄 Schema Reg   │                          │ 👂 Consumer     │
                       │                 │                          │                 │
                       │ Schemas: 2      │                          │ Rate: %s msg/s │
                       │ Status: %s      │                          │ Status: %s     │
                       └─────────────────┘                          └─────────────────┘

Pipeline Flow: Producer → Topic → Flink Processing → Output → Consumer
Status: %s | Duration: %s | Messages Processed: %s`

	// Format values safely
	producerRate := "0"
	producerStatus := "STOPPED"
	if status.ProducerMetrics != nil {
		producerRate = fmt.Sprintf("%.1f", status.ProducerMetrics.MessagesPerSec)
		producerStatus = status.ProducerMetrics.Status
	}

	partitions := "1"
	kafkaHealth := "UNKNOWN"
	if status.KafkaMetrics != nil && status.Resources != nil {
		kafkaHealth = status.KafkaMetrics.ClusterHealth
		if topic, exists := status.KafkaMetrics.Topics[status.Resources.InputTopic]; exists {
			partitions = fmt.Sprintf("%d", topic.Partitions)
		}
	}

	flinkJobs := 0
	flinkStatus := "STOPPED"
	if status.FlinkMetrics != nil {
		flinkJobs = len(status.FlinkMetrics.Jobs)
		flinkStatus = status.FlinkMetrics.JobManagerStatus
	}

	outputMessages := "0"
	outputLag := "0"
	if status.KafkaMetrics != nil && status.Resources != nil {
		if topic, exists := status.KafkaMetrics.Topics[status.Resources.OutputTopic]; exists {
			outputMessages = fmt.Sprintf("%d", topic.MessageCount)
			outputLag = fmt.Sprintf("%d", topic.Lag)
		}
	}

	schemaStatus := "UNKNOWN"
	// Schema Registry status would be added here

	consumerRate := "0"
	consumerStatus := "STOPPED"
	if status.ConsumerMetrics != nil {
		consumerRate = fmt.Sprintf("%.1f", status.ConsumerMetrics.MessagesPerSec)
		consumerStatus = status.ConsumerMetrics.Status
	}

	totalMessages := "0"
	if status.ExecutionSummary != nil {
		if status.ExecutionSummary.TotalMessagesProcessed >= 1_000_000 {
			totalMessages = fmt.Sprintf("%.1fM", float64(status.ExecutionSummary.TotalMessagesProcessed)/1_000_000)
		} else if status.ExecutionSummary.TotalMessagesProcessed >= 1_000 {
			totalMessages = fmt.Sprintf("%.1fK", float64(status.ExecutionSummary.TotalMessagesProcessed)/1_000)
		} else {
			totalMessages = fmt.Sprintf("%d", status.ExecutionSummary.TotalMessagesProcessed)
		}
	}

	return fmt.Sprintf(diagram,
		producerRate, partitions, flinkJobs, outputMessages,
		producerStatus, kafkaHealth, flinkStatus, outputLag,
		schemaStatus, consumerRate, consumerStatus,
		status.Status, status.Duration, totalMessages)
}

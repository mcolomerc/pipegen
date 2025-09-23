package dashboard

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	logpkg "pipegen/internal/log"
	"pipegen/internal/pipeline"
	itemplates "pipegen/internal/templates"
)

// ExecutionReportGenerator generates HTML execution reports
type ExecutionReportGenerator struct {
	outputDir    string
	templatePath string
	logoBase64   string
	logger       logpkg.Logger
}

// NewExecutionReportGenerator creates a new execution report generator
func NewExecutionReportGenerator(outputDir string) (*ExecutionReportGenerator, error) {
	templatePath := filepath.Join("internal", "templates", "files", "execution_report.html")

	// Load and encode logo
	logoPath := filepath.Join("web", "static", "logo.png")
	logoBase64 := ""
	if _, err := os.Stat(logoPath); err == nil {
		logoData, err := os.ReadFile(logoPath)
		if err == nil {
			logoBase64 = base64.StdEncoding.EncodeToString(logoData)
		}
	}

	return &ExecutionReportGenerator{
		outputDir:    outputDir,
		templatePath: templatePath,
		logoBase64:   logoBase64,
		logger:       logpkg.Global(),
	}, nil
}

// ExecutionReportData holds all data needed for the execution report template
type ExecutionReportData struct {
	ExecutionID     string                 `json:"execution_id"`
	Timestamp       time.Time              `json:"timestamp"`
	Parameters      ExecutionParameters    `json:"parameters"`
	Metrics         ExecutionMetrics       `json:"metrics"`
	Summary         ExecutionReportSummary `json:"summary"`
	Charts          ChartData              `json:"charts"`
	Status          string                 `json:"status"`
	StatusClass     string                 `json:"status_class"`
	Duration        time.Duration          `json:"duration"`
	ExecutionTime   string                 `json:"execution_time"`
	PipelineName    string                 `json:"pipeline_name"`
	PipelineVersion string                 `json:"pipeline_version"`
	LogoBase64      string                 `json:"logo_base64"`
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
	TotalMessages     int64                   `json:"total_messages"`
	MessagesPerSecond float64                 `json:"messages_per_second"`
	BytesProcessed    int64                   `json:"bytes_processed"`
	ErrorCount        int64                   `json:"error_count"`
	SuccessRate       float64                 `json:"success_rate"`
	AvgLatency        time.Duration           `json:"avg_latency"`
	ProducerMetrics   *pipeline.ProducerStats `json:"producer_metrics,omitempty"`
	ConsumerMetrics   *pipeline.ConsumerStats `json:"consumer_metrics,omitempty"`
	FlinkMetrics      *ExecutionFlinkMetrics  `json:"flink_metrics,omitempty"`
}

// ChartData holds data for visualization charts
type ChartData struct {
	MessagesOverTime   []ChartDataPoint `json:"messages_over_time"`
	ThroughputOverTime []ChartDataPoint `json:"throughput_over_time"`
	ErrorsOverTime     []ChartDataPoint `json:"errors_over_time"`
}

// ChartDataPoint represents a point in time series data
type ChartDataPoint = TimeSeriesPoint

// GenerateReport creates an HTML execution report
func (g *ExecutionReportGenerator) GenerateReport(data *ExecutionReportData) (string, error) {
	g.logger.Debug("ensure report output directory", "dir", g.outputDir)
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		g.logger.Error("failed to create output directory", "error", err)
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	g.logger.Debug("loading report template", "path", g.templatePath)
	tmplContent, err := os.ReadFile(g.templatePath)
	if err != nil {
		// Only fallback to embedded template if this is the default internal template path
		defaultPath := filepath.Join("internal", "templates", "files", "execution_report.html")
		if (g.templatePath == defaultPath || strings.HasSuffix(g.templatePath, "/internal/templates/files/execution_report.html")) && itemplates.ExecutionReportTemplate != "" {
			g.logger.Warn("failed to read default template; using embedded fallback", "error", err)
			tmplContent = []byte(itemplates.ExecutionReportTemplate)
		} else {
			g.logger.Error("failed to read template file", "error", err)
			return "", fmt.Errorf("failed to read template file: %w", err)
		}
	}

	g.logger.Debug("parsing report template")
	tmpl, err := template.New("report").Parse(string(tmplContent))
	if err != nil {
		g.logger.Error("failed to parse template", "error", err)
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data.LogoBase64 = g.logoBase64

	// Derive CSS status class if not already provided
	if data.StatusClass == "" {
		switch strings.ToLower(data.Status) {
		case "completed", "success", "succeeded":
			data.StatusClass = "status-success"
		case "failed", "error", "errored":
			data.StatusClass = "status-error"
		case "running", "in_progress", "in-progress", "active":
			data.StatusClass = "status-running"
		default:
			data.StatusClass = "status-info"
		}
	}

	// Populate ExecutionTime string if empty
	if data.ExecutionTime == "" && data.Duration > 0 {
		data.ExecutionTime = data.Duration.String()
	}

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("pipegen-execution-report-%s.html", timestamp)
	filePath := filepath.Join(g.outputDir, filename)

	g.logger.Debug("creating report output file", "path", filePath)
	file, err := os.Create(filePath)
	if err != nil {
		g.logger.Error("failed to create output file", "error", err)
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			g.logger.Warn("failed to close report file", "error", err)
		}
	}()

	g.logger.Debug("executing report template")
	if err := tmpl.Execute(file, data); err != nil {
		g.logger.Error("failed to execute report template", "error", err)
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	g.logger.Info("execution report generated", "path", filePath)
	return filePath, nil
}

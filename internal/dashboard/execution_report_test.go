package dashboard

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewExecutionReportGenerator(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		wantErr   bool
	}{
		{
			name:      "valid output directory",
			outputDir: "/tmp/reports",
			wantErr:   false,
		},
		{
			name:      "empty output directory",
			outputDir: "",
			wantErr:   false, // Constructor doesn't validate - validation happens during generation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator, err := NewExecutionReportGenerator(tt.outputDir)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, generator)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, generator)
				assert.Equal(t, tt.outputDir, generator.outputDir)
			}
		})
	}
}

func TestExecutionReportGenerator_GenerateReport(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "pipegen-report-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create template directory and file
	templateDir := filepath.Join(tmpDir, "templates")
	err = os.MkdirAll(templateDir, 0755)
	require.NoError(t, err)

	templateContent := `<!DOCTYPE html>
<html>
<head><title>Test Report</title></head>
<body>
		<h1>{{.ExecutionID}}</h1>
<p>Duration: {{.Summary.Duration}}</p>
<p>Status: {{.Summary.Status}}</p>
</body>
</html>`

	templatePath := filepath.Join(templateDir, "execution_report.html")
	err = os.WriteFile(templatePath, []byte(templateContent), 0644)
	require.NoError(t, err)

	// Create output directory
	outputDir := filepath.Join(tmpDir, "reports")
	generator := &ExecutionReportGenerator{
		outputDir:    outputDir,
		templatePath: templatePath,
		logoBase64:   "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==",
	}

	// Test data
	testReport := &ExecutionReportData{
		ExecutionID: "test-execution-123",
		Timestamp:   time.Now(),
		Status:      "completed",
		Duration:    2*time.Minute + 30*time.Second,
		Summary: ExecutionReportSummary{
			Status:         "completed",
			StartTime:      time.Now(),
			EndTime:        time.Now().Add(2*time.Minute + 30*time.Second),
			Duration:       2*time.Minute + 30*time.Second,
			TotalMessages:  1000,
			AverageLatency: time.Millisecond * 100,
			PeakThroughput: 500.0,
			ErrorRate:      0.01,
		},
		Parameters: ExecutionParameters{
			MessageRate: 100,
			Duration:    2*time.Minute + 30*time.Second,
		},
		Charts: ChartData{
			ThroughputOverTime: []ChartDataPoint{
				{Timestamp: time.Now(), Value: 100.0},
				{Timestamp: time.Now().Add(time.Minute), Value: 150.0},
			},
		},
	}

	t.Run("successful report generation", func(t *testing.T) {
		reportPath, err := generator.GenerateReport(testReport)

		assert.NoError(t, err)
		assert.NotEmpty(t, reportPath)

		// Verify file exists
		assert.FileExists(t, reportPath)

		// Verify file content
		content, err := os.ReadFile(reportPath)
		require.NoError(t, err)

		htmlContent := string(content)
		assert.Contains(t, htmlContent, "test-execution-123")
		assert.Contains(t, htmlContent, "2m30s")
		assert.Contains(t, htmlContent, "completed")
	})

	t.Run("report with timestamp in filename", func(t *testing.T) {
		reportPath, err := generator.GenerateReport(testReport)

		assert.NoError(t, err)
		assert.Contains(t, reportPath, "pipegen-execution-report-")
		assert.Contains(t, reportPath, ".html")
	})
}

func TestExecutionReportGenerator_TemplateNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pipegen-report-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	generator := &ExecutionReportGenerator{
		outputDir:    tmpDir,
		templatePath: "/nonexistent/template.html",
		logoBase64:   "",
	}

	testReport := &ExecutionReportData{
		ExecutionID: "test-execution-123",
	}

	reportPath, err := generator.GenerateReport(testReport)

	assert.Error(t, err)
	assert.Empty(t, reportPath)
	assert.Contains(t, err.Error(), "failed to read template file")
}

func TestLogoBase64Loading(t *testing.T) {
	// Create temporary directory with logo
	tmpDir, err := os.MkdirTemp("", "pipegen-logo-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create web/static directory structure
	logoDir := filepath.Join(tmpDir, "web", "static")
	err = os.MkdirAll(logoDir, 0755)
	require.NoError(t, err)

	// Create a minimal PNG file (1x1 transparent pixel)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 dimensions
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0B, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x08, 0x1D, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x90, 0x77, 0x53, 0xDE, 0x00, 0x00, 0x00, 0x00, // IEND chunk
		0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
	}

	logoPath := filepath.Join(logoDir, "logo.png")
	err = os.WriteFile(logoPath, pngData, 0644)
	require.NoError(t, err)

	// Change working directory temporarily
	oldWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() { _ = os.Chdir(oldWd) }()

	generator, err := NewExecutionReportGenerator(tmpDir)
	require.NoError(t, err)

	// Verify logo was loaded and encoded
	assert.NotEmpty(t, generator.logoBase64)
}

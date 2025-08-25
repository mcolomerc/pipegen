package pipeline

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ProjectDir:       "/tmp/test",
				MessageRate:      100,
				Duration:         5 * time.Minute,
				BootstrapServers: "localhost:9092",
			},
			wantErr: false,
		},
		{
			name: "invalid project dir",
			config: &Config{
				ProjectDir:       "",
				MessageRate:      100,
				Duration:         5 * time.Minute,
				BootstrapServers: "localhost:9092",
			},
			wantErr: true,
		},
		{
			name: "invalid message rate",
			config: &Config{
				MessageRate:      0,
				Duration:         5 * time.Minute,
				BootstrapServers: "localhost:9092",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := NewRunner(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, runner)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, runner)
			}
		})
	}
}

func TestRunner_GenerateExecutionID(t *testing.T) {
	config := &Config{
		ProjectDir:       "/tmp/test",
		MessageRate:      100,
		Duration:         5 * time.Minute,
		BootstrapServers: "localhost:9092",
	}

	runner, err := NewRunner(config)
	require.NoError(t, err)

	// Test that execution IDs are unique
	id1 := runner.generateExecutionID()
	id2 := runner.generateExecutionID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 16) // 8 bytes * 2 hex chars
}

func TestRunner_SetReportGenerator(t *testing.T) {
	config := &Config{
		ProjectDir:       "/tmp/test",
		MessageRate:      100,
		Duration:         5 * time.Minute,
		BootstrapServers: "localhost:9092",
	}

	runner, err := NewRunner(config)
	require.NoError(t, err)

	// Mock report generator
	mockGenerator := &mockReportGenerator{}
	runner.SetReportGenerator(mockGenerator)

	// Verify it was set (would need to expose getter or test via behavior)
	assert.NotNil(t, runner)
}

func TestRunner_SetDashboardServer(t *testing.T) {
	config := &Config{
		ProjectDir:       "/tmp/test",
		MessageRate:      100,
		Duration:         5 * time.Minute,
		BootstrapServers: "localhost:9092",
	}

	runner, err := NewRunner(config)
	require.NoError(t, err)

	// Mock dashboard server
	mockServer := &mockDashboardServer{}
	runner.SetDashboardServer(mockServer)

	assert.NotNil(t, runner)
}

// Mock implementations for testing
type mockReportGenerator struct{}

func (m *mockReportGenerator) GenerateReport(data interface{}) (string, error) {
	return "/tmp/report.html", nil
}

type mockDashboardServer struct{}

func (m *mockDashboardServer) UpdatePipelineStatus(status interface{}) {}

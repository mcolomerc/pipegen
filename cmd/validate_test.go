package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateProjectStructure(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() string
		wantErr   bool
		cleanup   func(string)
	}{
		{
			name: "valid project structure",
			setupFunc: func() string {
				tmpDir, _ := os.MkdirTemp("", "pipegen-test-*")
				// Create required directories
				_ = os.MkdirAll(filepath.Join(tmpDir, "sql", "local"), 0755)
				_ = os.MkdirAll(filepath.Join(tmpDir, "sql", "cloud"), 0755)
				_ = os.MkdirAll(filepath.Join(tmpDir, "schemas"), 0755)
				_ = os.MkdirAll(filepath.Join(tmpDir, "config"), 0755)

				// Create required files
				_ = os.WriteFile(filepath.Join(tmpDir, "schemas", "input.json"), []byte(`{"type": "record"}`), 0644)
				_ = os.WriteFile(filepath.Join(tmpDir, "schemas", "output.json"), []byte(`{"type": "record"}`), 0644)
				_ = os.WriteFile(filepath.Join(tmpDir, "config", "local.yaml"), []byte("test: true"), 0644)

				return tmpDir
			},
			wantErr: false,
			cleanup: func(dir string) { _ = os.RemoveAll(dir) },
		},
		{
			name: "missing sql directory",
			setupFunc: func() string {
				tmpDir, _ := os.MkdirTemp("", "pipegen-test-*")
				// Don't create sql directory
				return tmpDir
			},
			wantErr: true,
			cleanup: func(dir string) { _ = os.RemoveAll(dir) },
		},
		{
			name: "missing schemas directory",
			setupFunc: func() string {
				tmpDir, _ := os.MkdirTemp("", "pipegen-test-*")
				_ = os.MkdirAll(filepath.Join(tmpDir, "sql", "local"), 0755)
				// Don't create schemas directory
				return tmpDir
			},
			wantErr: true,
			cleanup: func(dir string) { _ = os.RemoveAll(dir) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir := tt.setupFunc()
			defer tt.cleanup(projectDir)

			err := validateProjectStructure(projectDir)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSQLFiles(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() string
		wantErr   bool
		cleanup   func(string)
	}{
		{
			name: "valid SQL files",
			setupFunc: func() string {
				tmpDir, _ := os.MkdirTemp("", "pipegen-test-*")
				sqlDir := filepath.Join(tmpDir, "sql", "local")
				_ = os.MkdirAll(sqlDir, 0755)

				// Create valid SQL files
				validSQL := `CREATE TABLE input_events (
					id BIGINT,
					event_type STRING,
					timestamp_col TIMESTAMP(3),
					WATERMARK FOR timestamp_col AS timestamp_col - INTERVAL '5' SECOND
				) WITH (
					'connector' = 'kafka',
					'topic' = 'input-events'
				);`

				_ = os.WriteFile(filepath.Join(sqlDir, "01_create_table.sql"), []byte(validSQL), 0644)
				return tmpDir
			},
			wantErr: false,
			cleanup: func(dir string) { _ = os.RemoveAll(dir) },
		},
		{
			name: "invalid SQL syntax",
			setupFunc: func() string {
				tmpDir, _ := os.MkdirTemp("", "pipegen-test-*")
				sqlDir := filepath.Join(tmpDir, "sql", "local")
				_ = os.MkdirAll(sqlDir, 0755)

				// Create invalid SQL
				invalidSQL := `CREATE TABLE invalid syntax here;`
				_ = os.WriteFile(filepath.Join(sqlDir, "01_invalid.sql"), []byte(invalidSQL), 0644)
				return tmpDir
			},
			wantErr: true,
			cleanup: func(dir string) { _ = os.RemoveAll(dir) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir := tt.setupFunc()
			defer tt.cleanup(projectDir)

			err := validateSQLFiles(projectDir)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAVROSchemas(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func() string
		wantErr   bool
		cleanup   func(string)
	}{
		{
			name: "valid AVRO schemas",
			setupFunc: func() string {
				tmpDir, _ := os.MkdirTemp("", "pipegen-test-*")
				schemasDir := filepath.Join(tmpDir, "schemas")
				_ = os.MkdirAll(schemasDir, 0755)

				validSchema := `{
					"type": "record",
					"name": "UserEvent",
					"fields": [
						{"name": "id", "type": "long"},
						{"name": "event_type", "type": "string"},
						{"name": "timestamp", "type": "long"}
					]
				}`

				_ = os.WriteFile(filepath.Join(schemasDir, "input.json"), []byte(validSchema), 0644)
				_ = os.WriteFile(filepath.Join(schemasDir, "output.json"), []byte(validSchema), 0644)
				return tmpDir
			},
			wantErr: false,
			cleanup: func(dir string) { _ = os.RemoveAll(dir) },
		},
		{
			name: "invalid JSON schema",
			setupFunc: func() string {
				tmpDir, _ := os.MkdirTemp("", "pipegen-test-*")
				schemasDir := filepath.Join(tmpDir, "schemas")
				_ = os.MkdirAll(schemasDir, 0755)

				// Invalid JSON
				invalidSchema := `{"type": "record", "name": "Invalid", invalid json here}`
				_ = os.WriteFile(filepath.Join(schemasDir, "input.json"), []byte(invalidSchema), 0644)
				return tmpDir
			},
			wantErr: true,
			cleanup: func(dir string) { _ = os.RemoveAll(dir) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectDir := tt.setupFunc()
			defer tt.cleanup(projectDir)

			err := validateAVROSchemas(projectDir)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	// Test configuration validation
	t.Run("valid config", func(t *testing.T) {
		// This would test the validateConfig function
		// You'll need to implement this based on your config structure
		err := validateConfig()
		// For now, we'll assume it passes
		assert.NoError(t, err)
	})
}

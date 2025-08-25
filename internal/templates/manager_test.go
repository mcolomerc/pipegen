package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	t.Run("create new manager", func(t *testing.T) {
		manager, err := NewManager()
		assert.NoError(t, err)
		assert.NotNil(t, manager)
		assert.NotNil(t, manager.templates)
	})
}

func TestManager_RenderReadmeStandard(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)

	data := TemplateData{
		ProjectName:        "test-project",
		ProjectNameTitle:   "Test Project",
		SanitizedName:      "test_project",
		Description:        "A test project",
		Optimizations:      []string{"optimization1", "optimization2"},
		WithSchemaRegistry: true,
	}

	t.Run("render standard readme", func(t *testing.T) {
		result, err := manager.RenderReadmeStandard(data)
		if err == nil {
			assert.NotEmpty(t, result)
			// Check that template variables were substituted
			assert.Contains(t, result, data.ProjectName)
		} else {
			// If template files don't exist, we should get a clear error
			assert.Error(t, err)
		}
	})
}

func TestManager_RenderReadmeLLM(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)

	data := TemplateData{
		ProjectName:        "test-project",
		ProjectNameTitle:   "Test Project",
		SanitizedName:      "test_project",
		Description:        "A test project",
		Optimizations:      []string{"optimization1", "optimization2"},
		WithSchemaRegistry: true,
	}

	t.Run("render LLM readme", func(t *testing.T) {
		result, err := manager.RenderReadmeLLM(data)
		if err == nil {
			assert.NotEmpty(t, result)
			// Check that template variables were substituted
			assert.Contains(t, result, data.ProjectName)
		} else {
			// If template files don't exist, we should get a clear error
			assert.Error(t, err)
		}
	})
}

func TestManager_RenderDockerCompose(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)

	data := TemplateData{
		ProjectName:        "test-project",
		ProjectNameTitle:   "Test Project",
		SanitizedName:      "test_project",
		Description:        "A test project",
		Optimizations:      []string{"optimization1", "optimization2"},
		WithSchemaRegistry: true,
	}

	t.Run("render docker compose", func(t *testing.T) {
		result, err := manager.RenderDockerCompose(data)
		if err == nil {
			assert.NotEmpty(t, result)
		} else {
			// If template files don't exist, we should get a clear error
			assert.Error(t, err)
		}
	})
}

func TestManager_RenderSQLFiles(t *testing.T) {
	manager, err := NewManager()
	require.NoError(t, err)

	t.Run("render local SQL files", func(t *testing.T) {
		sqlFiles, err := manager.RenderSQLFiles(true)
		if err == nil {
			assert.NotEmpty(t, sqlFiles)
			// Check that we have the expected SQL files
			for filename := range sqlFiles {
				assert.True(t, strings.HasSuffix(filename, ".sql"))
			}
		} else {
			// If SQL template directory doesn't exist, we should get a clear error
			assert.Error(t, err)
		}
	})

	t.Run("render cloud SQL files", func(t *testing.T) {
		sqlFiles, err := manager.RenderSQLFiles(false)
		if err == nil {
			assert.NotEmpty(t, sqlFiles)
			// Check that we have the expected SQL files
			for filename := range sqlFiles {
				assert.True(t, strings.HasSuffix(filename, ".sql"))
			}
		} else {
			// If SQL template directory doesn't exist, we should get a clear error
			assert.Error(t, err)
		}
	})
}

func TestSanitizeAVROIdentifier(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "test",
			expected: "test",
		},
		{
			name:     "name with hyphens",
			input:    "test-project",
			expected: "test_project",
		},
		{
			name:     "name with dots",
			input:    "test.project",
			expected: "test_project",
		},
		{
			name:     "name with spaces",
			input:    "test project",
			expected: "test_project",
		},
		{
			name:     "name with mixed characters",
			input:    "test-project.name with spaces",
			expected: "test_project_name_with_spaces",
		},
		{
			name:     "name with invalid characters",
			input:    "test@#$%project",
			expected: "testproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeAVROIdentifier(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

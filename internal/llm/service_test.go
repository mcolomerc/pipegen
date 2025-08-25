package llm

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLLMService(t *testing.T) {
	service := NewLLMService()
	assert.NotNil(t, service)
}

func TestLLMService_IsEnabled(t *testing.T) {
	tests := []struct {
		name           string
		setupEnv       func()
		expectedResult bool
		cleanup        func()
	}{
		{
			name: "ollama enabled with host",
			setupEnv: func() {
				_ = os.Setenv("PIPEGEN_OLLAMA_MODEL", "llama3.1")
			},
			expectedResult: true,
			cleanup: func() {
				_ = os.Unsetenv("PIPEGEN_OLLAMA_MODEL")
			},
		},
		{
			name: "openai enabled with api key",
			setupEnv: func() {
				_ = os.Setenv("PIPEGEN_OPENAI_API_KEY", "test-key")
			},
			expectedResult: true,
			cleanup: func() {
				_ = os.Unsetenv("PIPEGEN_OPENAI_API_KEY")
			},
		},
		{
			name: "no llm provider configured",
			setupEnv: func() {
				// Ensure no env vars are set
				_ = os.Unsetenv("PIPEGEN_OLLAMA_URL")
				_ = os.Unsetenv("PIPEGEN_OLLAMA_MODEL")
				_ = os.Unsetenv("PIPEGEN_OPENAI_API_KEY")
			},
			expectedResult: false,
			cleanup:        func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanup()

			// Create service after setting environment variables
			service := NewLLMService()
			result := service.IsEnabled()
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestLLMService_GetProvider(t *testing.T) {
	tests := []struct {
		name             string
		setupEnv         func()
		expectedProvider string
		cleanup          func()
	}{
		{
			name: "ollama provider",
			setupEnv: func() {
				_ = os.Setenv("PIPEGEN_OLLAMA_MODEL", "llama3.1")
				_ = os.Unsetenv("PIPEGEN_OPENAI_API_KEY")
			},
			expectedProvider: "ollama",
			cleanup: func() {
				_ = os.Unsetenv("PIPEGEN_OLLAMA_MODEL")
			},
		},
		{
			name: "openai provider",
			setupEnv: func() {
				_ = os.Setenv("PIPEGEN_OPENAI_API_KEY", "sk-test-key")
				_ = os.Unsetenv("PIPEGEN_OLLAMA_MODEL")
				_ = os.Unsetenv("PIPEGEN_OLLAMA_URL")
			},
			expectedProvider: "openai",
			cleanup: func() {
				_ = os.Unsetenv("PIPEGEN_OPENAI_API_KEY")
			},
		},
		{
			name: "no provider",
			setupEnv: func() {
				_ = os.Unsetenv("PIPEGEN_OLLAMA_URL")
				_ = os.Unsetenv("PIPEGEN_OLLAMA_MODEL")
				_ = os.Unsetenv("PIPEGEN_OPENAI_API_KEY")
			},
			expectedProvider: "",
			cleanup:          func() {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupEnv()
			defer tt.cleanup()

			// Create service after setting environment variables
			service := NewLLMService()
			provider := service.GetProvider()
			// Convert LLMProvider to string for comparison
			providerStr := string(provider)
			assert.Equal(t, tt.expectedProvider, providerStr)
		})
	}
}

func TestLLMService_GeneratePipeline(t *testing.T) {
	t.Run("llm disabled", func(t *testing.T) {
		// Ensure no LLM providers are configured
		_ = os.Unsetenv("PIPEGEN_OLLAMA_URL")
		_ = os.Unsetenv("PIPEGEN_OLLAMA_MODEL")
		_ = os.Unsetenv("PIPEGEN_OPENAI_API_KEY")

		// Create service after clearing env vars
		service := NewLLMService()
		content, err := service.GeneratePipeline(context.Background(), "test description", "test domain")
		assert.Error(t, err)
		assert.Nil(t, content)
		assert.Contains(t, err.Error(), "LLM service not enabled")
	})

	// Note: We can't easily test actual LLM calls without mocking or integration tests
	// For unit tests, we focus on the service logic rather than external API calls
}

// Remove the TestConnection test since that method doesn't exist
func TestLLMService_ParseResponse(t *testing.T) {
	service := NewLLMService()

	tests := []struct {
		name     string
		response string
		wantErr  bool
	}{
		{
			name: "valid response format",
			response: `{
				"input_schema": "input schema here",
				"output_schema": "output schema here",
				"sql_statements": {
					"create_table": "CREATE TABLE test",
					"processing": "SELECT * FROM test"
				},
				"description": "Test explanation",
				"optimizations": ["optimization1", "optimization2"]
			}`,
			wantErr: false,
		},
		{
			name:     "invalid json response",
			response: "invalid json {",
			wantErr:  true,
		},
		{
			name:     "empty response",
			response: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := service.parseResponse(tt.response)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, content)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, content)
			}
		})
	}
}

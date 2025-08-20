package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// LLMProvider represents different LLM providers
type LLMProvider string

const (
	ProviderOpenAI LLMProvider = "openai"
	ProviderOllama LLMProvider = "ollama"
)

// LLMService handles AI-powered code generation
type LLMService struct {
	provider LLMProvider
	apiKey   string
	model    string
	baseURL  string
	enabled  bool
}

// NewLLMService creates a new LLM service (optional dependency)
func NewLLMService() *LLMService {
	// Check for Ollama first (local, no API key needed)
	if ollamaURL := os.Getenv("PIPEGEN_OLLAMA_URL"); ollamaURL != "" {
		model := os.Getenv("PIPEGEN_OLLAMA_MODEL")
		if model == "" {
			model = "llama3.1" // Default Ollama model
		}
		
		return &LLMService{
			provider: ProviderOllama,
			model:    model,
			baseURL:  ollamaURL,
			enabled:  true,
		}
	}
	
	// Default to localhost Ollama if no URL specified
	if _, exists := os.LookupEnv("PIPEGEN_OLLAMA_MODEL"); exists {
		model := os.Getenv("PIPEGEN_OLLAMA_MODEL")
		if model == "" {
			model = "llama3.1"
		}
		
		return &LLMService{
			provider: ProviderOllama,
			model:    model,
			baseURL:  "http://localhost:11434",
			enabled:  true,
		}
	}
	
	// Fall back to OpenAI
	apiKey := os.Getenv("PIPEGEN_OPENAI_API_KEY")
	if apiKey == "" {
		return &LLMService{enabled: false}
	}

	return &LLMService{
		provider: ProviderOpenAI,
		apiKey:   apiKey,
		model:    getOpenAIModel(),
		baseURL:  "https://api.openai.com/v1",
		enabled:  true,
	}
}

// IsEnabled checks if LLM service is available
func (s *LLMService) IsEnabled() bool {
	return s.enabled
}

// GetProvider returns the current LLM provider
func (s *LLMService) GetProvider() LLMProvider {
	return s.provider
}

// GeneratedContent represents LLM-generated pipeline components
type GeneratedContent struct {
	InputSchema    string            `json:"input_schema"`
	OutputSchema   string            `json:"output_schema"`
	SQLStatements  map[string]string `json:"sql_statements"`
	Description    string            `json:"description"`
	Optimizations  []string          `json:"optimizations"`
}

// GeneratePipeline creates pipeline components from natural language description
func (s *LLMService) GeneratePipeline(ctx context.Context, description, domain string) (*GeneratedContent, error) {
	if !s.enabled {
		return nil, fmt.Errorf("LLM service not enabled. Set PIPEGEN_OPENAI_API_KEY or PIPEGEN_OLLAMA_MODEL environment variable")
	}

	prompt := buildPrompt(description, domain)
	
	var response string
	var err error
	
	switch s.provider {
	case ProviderOllama:
		response, err = s.callOllama(ctx, prompt)
	case ProviderOpenAI:
		response, err = s.callOpenAI(ctx, prompt)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", s.provider)
	}
	
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	return s.parseResponse(response)
}

func getOpenAIModel() string {
	if model := os.Getenv("PIPEGEN_LLM_MODEL"); model != "" {
		return model
	}
	return "gpt-4" // Default OpenAI model
}

func buildPrompt(description, domain string) string {
	return fmt.Sprintf(`You are an expert in Apache Kafka and FlinkSQL. Generate a complete streaming pipeline based on this description:

Description: %s
Domain: %s

Generate a JSON response with:
1. input_schema: AVRO schema for input events
2. output_schema: AVRO schema for output results  
3. sql_statements: Map of filename to FlinkSQL statements
4. description: Technical summary of the pipeline
5. optimizations: Performance optimization suggestions

Focus on:
- Realistic field names and types for the domain
- Proper FlinkSQL syntax with windowing and aggregations
- Performance-optimized schema design
- Clear, maintainable SQL structure

Respond only with valid JSON.`, description, domain)
}

// OllamaRequest represents the request structure for Ollama API
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

// OllamaResponse represents the response structure from Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (s *LLMService) callOllama(ctx context.Context, prompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:  s.model,
		Prompt: prompt,
		Stream: false,
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}
	
	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w. Make sure Ollama is running at %s", err, s.baseURL)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API returned status %d. Is the model '%s' installed? Run: ollama pull %s", resp.StatusCode, s.model, s.model)
	}
	
	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode Ollama response: %w", err)
	}
	
	return ollamaResp.Response, nil
}

func (s *LLMService) callOpenAI(ctx context.Context, prompt string) (string, error) {
	// Implementation for OpenAI API call
	// This would use the official OpenAI Go client
	// For now, returning mock response for structure
	
	mockResponse := `{
		"input_schema": "{\"type\":\"record\",\"name\":\"UserEvent\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"event_type\",\"type\":\"string\"},{\"name\":\"timestamp\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}",
		"output_schema": "{\"type\":\"record\",\"name\":\"UserMetrics\",\"fields\":[{\"name\":\"user_id\",\"type\":\"string\"},{\"name\":\"event_count\",\"type\":\"long\"},{\"name\":\"window_start\",\"type\":{\"type\":\"long\",\"logicalType\":\"timestamp-millis\"}}]}",
		"sql_statements": {
			"01_create_source_table.sql": "CREATE TABLE user_events (user_id STRING, event_type STRING, timestamp_col TIMESTAMP(3)) WITH ('connector' = 'kafka', 'topic' = '${INPUT_TOPIC}', 'properties.bootstrap.servers' = 'localhost:9092', 'format' = 'avro-confluent', 'avro-confluent.url' = 'http://localhost:8082')",
			"02_create_processing.sql": "CREATE VIEW user_metrics AS SELECT user_id, COUNT(*) as event_count, TUMBLE_START(timestamp_col, INTERVAL '1' MINUTE) as window_start FROM user_events GROUP BY user_id, TUMBLE(timestamp_col, INTERVAL '1' MINUTE)",
			"03_create_output_table.sql": "CREATE TABLE output_metrics (user_id STRING, event_count BIGINT, window_start TIMESTAMP(3)) WITH ('connector' = 'kafka', 'topic' = '${OUTPUT_TOPIC}', 'properties.bootstrap.servers' = 'localhost:9092', 'format' = 'avro-confluent', 'avro-confluent.url' = 'http://localhost:8082')",
			"04_insert_results.sql": "INSERT INTO output_metrics SELECT user_id, event_count, window_start FROM user_metrics"
		},
		"description": "Real-time user activity aggregation pipeline that processes user events and calculates per-minute activity counts",
		"optimizations": ["Consider partitioning by user_id for better parallelism", "Add watermark handling for late events", "Consider using HOP windows for overlapping metrics"]
	}`
	
	return mockResponse, nil
}

func (s *LLMService) parseResponse(response string) (*GeneratedContent, error) {
	var content GeneratedContent
	if err := json.Unmarshal([]byte(response), &content); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	return &content, nil
}

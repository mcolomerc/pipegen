package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// OllamaModel represents a model in Ollama
type OllamaModel struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// OllamaModelsResponse represents the response from /api/tags
type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

// CheckOllamaConnection verifies that Ollama is running and the model is available
func (s *LLMService) CheckOllamaConnection(ctx context.Context) error {
	if s.provider != ProviderOllama {
		return nil
	}
	
	// Check if Ollama server is running
	client := &http.Client{Timeout: 5 * time.Second}
	
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/api/tags", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama is not running at %s. Start it with: ollama serve", s.baseURL)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama server returned status %d", resp.StatusCode)
	}
	
	// Check if the model is installed
	var modelsResp OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return fmt.Errorf("failed to decode models response: %w", err)
	}
	
	modelFound := false
	for _, model := range modelsResp.Models {
		if model.Name == s.model || model.Name == s.model+":latest" {
			modelFound = true
			break
		}
	}
	
	if !modelFound {
		return fmt.Errorf("model '%s' is not installed. Install it with: ollama pull %s", s.model, s.model)
	}
	
	return nil
}

// GetProviderInfo returns human-readable information about the configured provider
func (s *LLMService) GetProviderInfo() string {
	if !s.enabled {
		return "No AI provider configured"
	}
	
	switch s.provider {
	case ProviderOllama:
		return fmt.Sprintf("Ollama (local) - Model: %s, URL: %s", s.model, s.baseURL)
	case ProviderOpenAI:
		return fmt.Sprintf("OpenAI (cloud) - Model: %s", s.model)
	default:
		return "Unknown provider"
	}
}

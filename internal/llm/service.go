package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
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
	InputSchema   string            `json:"input_schema"`
	OutputSchema  string            `json:"output_schema"`
	SQLStatements map[string]string `json:"sql_statements"`
	Description   string            `json:"description"`
	Optimizations []string          `json:"optimizations"`
}

// FlexibleGeneratedContent for parsing flexible JSON responses
type FlexibleGeneratedContent struct {
	InputSchema   json.RawMessage   `json:"input_schema"`
	OutputSchema  json.RawMessage   `json:"output_schema"`
	SQLStatements map[string]string `json:"sql_statements"`
	Description   json.RawMessage   `json:"description"`
	Optimizations json.RawMessage   `json:"optimizations"`
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

	// Use longer timeout for AI generation - these requests can take 2-5 minutes
	client := &http.Client{Timeout: 300 * time.Second} // 5 minutes
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Ollama API: %w. Make sure Ollama is running at %s", err, s.baseURL)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama API returned status %d. Is the model '%s' installed? Run: ollama pull %s", resp.StatusCode, s.model, s.model)
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

// extractJSONFromMarkdown extracts JSON content from markdown-formatted responses
func extractJSONFromMarkdown(response string) string {
	// First, try to find JSON within markdown code blocks
	codeBlockPattern := regexp.MustCompile("```(?:json)?\n?([^`]+)\n?```")
	if matches := codeBlockPattern.FindStringSubmatch(response); len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// If no code blocks found, use brace counting to find complete JSON
	if strings.Contains(response, "{") && strings.Contains(response, "}") {
		start := strings.Index(response, "{")
		if start == -1 {
			return strings.TrimSpace(response)
		}

		// Count braces to find the complete JSON object
		braceCount := 0
		inString := false
		escaped := false

		for i := start; i < len(response); i++ {
			char := response[i]

			if escaped {
				escaped = false
				continue
			}

			if char == '\\' {
				escaped = true
				continue
			}

			if char == '"' && !escaped {
				inString = !inString
				continue
			}

			if !inString {
				switch char {
				case '{':
					braceCount++
				case '}':
					braceCount--
					if braceCount == 0 {
						// Found complete JSON object
						return strings.TrimSpace(response[start : i+1])
					}
				}
			}
		}

		// If we didn't find a complete object, return from start to end
		end := strings.LastIndex(response, "}") + 1
		if start < end {
			return strings.TrimSpace(response[start:end])
		}
	}

	// If no JSON patterns found, return the original response
	return strings.TrimSpace(response)
}

func (s *LLMService) parseResponse(response string) (*GeneratedContent, error) {
	// Clean the response by extracting JSON from markdown code blocks
	jsonStr := extractJSONFromMarkdown(response)

	// Try to fix common JSON issues
	jsonStr = s.fixCommonJSONIssues(jsonStr)

	// First try parsing with flexible schema fields
	var flexContent FlexibleGeneratedContent
	if err := json.Unmarshal([]byte(jsonStr), &flexContent); err != nil {
		// If parsing fails, log the full content for debugging
		fmt.Printf("⚠️  Failed to parse JSON response.\n")
		fmt.Printf("Raw LLM response:\n%s\n", response)
		fmt.Printf("Extracted JSON:\n%s\n", jsonStr)
		fmt.Printf("Parse error: %v\n", err)
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	// Convert to final content, handling both string and object formats for schemas
	content := &GeneratedContent{
		SQLStatements: flexContent.SQLStatements,
	}

	// Handle InputSchema - could be string or JSON object
	if err := s.convertSchemaField(flexContent.InputSchema, &content.InputSchema); err != nil {
		return nil, fmt.Errorf("failed to convert input_schema: %w", err)
	}

	// Handle OutputSchema - could be string or JSON object
	if err := s.convertSchemaField(flexContent.OutputSchema, &content.OutputSchema); err != nil {
		return nil, fmt.Errorf("failed to convert output_schema: %w", err)
	}

	// Handle Description - could be string or object
	if err := s.convertDescriptionField(flexContent.Description, &content.Description); err != nil {
		return nil, fmt.Errorf("failed to convert description: %w", err)
	}

	// Handle Optimizations - could be array of strings or array of objects
	if err := s.convertOptimizationsField(flexContent.Optimizations, &content.Optimizations); err != nil {
		return nil, fmt.Errorf("failed to convert optimizations: %w", err)
	}

	return content, nil
}

// fixCommonJSONIssues attempts to fix common JSON formatting issues in LLM responses
func (s *LLMService) fixCommonJSONIssues(jsonStr string) string {
	// Convert template literals (backticks) to proper JSON strings
	// This handles patterns like: "key": `value` -> "key": "value"
	jsonStr = s.convertTemplateLiterals(jsonStr)

	// Fix multi-line string literals that aren't properly escaped
	jsonStr = s.fixMultiLineStrings(jsonStr)

	// Remove any opening braces followed immediately by comma: {, -> {
	jsonStr = regexp.MustCompile(`\{\s*,`).ReplaceAllString(jsonStr, `{`)

	// Remove any opening brackets followed immediately by comma: [, -> [
	jsonStr = regexp.MustCompile(`\[\s*,`).ReplaceAllString(jsonStr, `[`)

	// Remove any trailing commas before closing braces or brackets
	jsonStr = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(jsonStr, "$1")

	// Fix malformed quotes in strings (common LLM issue)
	// Pattern like: "name": ""UserEvent\", \"fields\" -> "name": "UserEvent", "fields"
	jsonStr = regexp.MustCompile(`""\s*([^"]+)\\"?,\s*\\?"([^"]+)"`).ReplaceAllString(jsonStr, `"$1", "$2"`)

	// Remove any duplicate quotes at the start of values
	jsonStr = regexp.MustCompile(`":\s*""`).ReplaceAllString(jsonStr, `": "`)

	// Fix escaped quotes that shouldn't be escaped in JSON keys/values
	// But be careful not to break intentionally escaped quotes in content
	jsonStr = regexp.MustCompile(`([^\\])\\"`).ReplaceAllString(jsonStr, `$1"`)

	// Ensure proper spacing around colons
	jsonStr = regexp.MustCompile(`"(\s*):(\s*)`).ReplaceAllString(jsonStr, `"$1: `)

	// DON'T try to fix missing commas automatically - this was causing the {, issue
	// The extracted JSON should already be valid, and adding commas can break things
	// jsonStr = regexp.MustCompile(`(":\s*[^,}\]]+)\s*\n\s*(")`).ReplaceAllString(jsonStr, `$1,`+"\n"+`$2`)

	// Final cleanup: remove extra commas at the end of arrays or objects
	jsonStr = regexp.MustCompile(`,(\s*[}\]])`).ReplaceAllString(jsonStr, "$1")

	return strings.TrimSpace(jsonStr)
}

// fixStringConcatenation fixes JavaScript-style string concatenation in JSON
func (s *LLMService) fixStringConcatenation(jsonStr string) string {
	// Look for patterns like: "string1"\n + "string2"\n + "string3"
	// and convert them to: "string1string2string3"

	// Use regex to find and replace JavaScript-style concatenation
	// Pattern: "text"\s*\+\s*"text" -> "texttext"
	for {
		// Find pattern: "something" + "something else"
		pattern := regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"\s*\+\s*"([^"\\]*(\\.[^"\\]*)*)"`)
		if !pattern.MatchString(jsonStr) {
			break
		}
		// Replace with concatenated string, preserving escape sequences
		jsonStr = pattern.ReplaceAllString(jsonStr, `"$1$3"`)
	}

	// Also handle patterns where there are newlines and extra whitespace
	// Pattern: "text"\n                        + "text" -> "texttext"
	pattern2 := regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"\s*\n\s*\+\s*"([^"\\]*(\\.[^"\\]*)*)"`)
	for pattern2.MatchString(jsonStr) {
		jsonStr = pattern2.ReplaceAllString(jsonStr, `"$1$3"`)
	}

	return jsonStr
}

// fixSQLStatements fixes SQL statements that are incorrectly formatted as arrays instead of strings
func (s *LLMService) fixSQLStatements(jsonStr string) string {
	// Look for patterns like: "filename": { "statement1", "statement2" }
	// and convert them to: "filename": "statement1 statement2"

	// Pattern to match SQL statement objects with comma-separated strings
	pattern := regexp.MustCompile(`("[\w.-]+\.sql")\s*:\s*\{\s*("[^"]*"(?:\s*,\s*"[^"]*")*)\s*\}`)

	for pattern.MatchString(jsonStr) {
		jsonStr = pattern.ReplaceAllStringFunc(jsonStr, func(match string) string {
			// Extract filename and statements
			subMatches := pattern.FindStringSubmatch(match)
			if len(subMatches) != 3 {
				return match
			}

			filename := subMatches[1]
			statements := subMatches[2]

			// Split the comma-separated strings and join them
			re := regexp.MustCompile(`"([^"]*)"`)
			stmtMatches := re.FindAllStringSubmatch(statements, -1)

			var joinedStatements []string
			for _, stmtMatch := range stmtMatches {
				if len(stmtMatch) > 1 {
					joinedStatements = append(joinedStatements, stmtMatch[1])
				}
			}

			// Join all statements with a space and return as a proper JSON string
			combined := strings.Join(joinedStatements, " ")
			return fmt.Sprintf(`%s: "%s"`, filename, combined)
		})
	}

	// Also handle arrays: "filename": ["statement1", "statement2"]
	arrayPattern := regexp.MustCompile(`("[\w.-]+\.sql")\s*:\s*\[\s*("[^"]*"(?:\s*,\s*"[^"]*")*)\s*\]`)

	for arrayPattern.MatchString(jsonStr) {
		jsonStr = arrayPattern.ReplaceAllStringFunc(jsonStr, func(match string) string {
			// Extract filename and statements
			subMatches := arrayPattern.FindStringSubmatch(match)
			if len(subMatches) != 3 {
				return match
			}

			filename := subMatches[1]
			statements := subMatches[2]

			// Split the comma-separated strings and join them
			re := regexp.MustCompile(`"([^"]*)"`)
			stmtMatches := re.FindAllStringSubmatch(statements, -1)

			var joinedStatements []string
			for _, stmtMatch := range stmtMatches {
				if len(stmtMatch) > 1 {
					joinedStatements = append(joinedStatements, stmtMatch[1])
				}
			}

			// Join all statements with a newline and return as a proper JSON string
			combined := strings.Join(joinedStatements, " ")
			return fmt.Sprintf(`%s: "%s"`, filename, combined)
		})
	}

	return jsonStr
}

// fixMultiLineStrings fixes JSON strings that contain unescaped newlines
func (s *LLMService) fixMultiLineStrings(jsonStr string) string {
	// First, fix JavaScript-style string concatenation
	jsonStr = s.fixStringConcatenation(jsonStr)

	// Fix SQL statements that are incorrectly formatted as objects
	jsonStr = s.fixSQLStatements(jsonStr)

	// Use a state machine to properly handle multi-line strings in JSON
	result := strings.Builder{}
	inString := false
	escaped := false

	for i := 0; i < len(jsonStr); i++ {
		char := jsonStr[i]

		if escaped {
			result.WriteByte(char)
			escaped = false
			continue
		}

		if char == '\\' {
			result.WriteByte(char)
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			result.WriteByte(char)
			continue
		}

		if inString {
			// Inside a JSON string, escape newlines and tabs
			switch char {
			case '\n':
				result.WriteString(`\n`)
			case '\r':
				result.WriteString(`\r`)
			case '\t':
				result.WriteString(`\t`)
			default:
				result.WriteByte(char)
			}
		} else {
			// Outside of strings, keep everything as-is
			result.WriteByte(char)
		}
	}

	return result.String()
}

// convertTemplateLiterals converts template literals (backticks) and triple quotes to proper JSON strings
func (s *LLMService) convertTemplateLiterals(jsonStr string) string {
	// First handle triple quotes
	jsonStr = s.convertTripleQuotes(jsonStr)

	// Then handle backticks
	// Find patterns like: "key": `value` and convert to "key": "value"
	// We need to be careful about nested quotes within the template literal

	// Use a state machine approach to handle this properly
	result := strings.Builder{}
	inBackticks := false
	inQuotes := false
	escaped := false

	for i := 0; i < len(jsonStr); i++ {
		char := jsonStr[i]

		if escaped {
			result.WriteByte(char)
			escaped = false
			continue
		}

		if char == '\\' {
			result.WriteByte(char)
			escaped = true
			continue
		}

		if char == '"' && !inBackticks {
			inQuotes = !inQuotes
			result.WriteByte(char)
			continue
		}

		if char == '`' && !inQuotes {
			if !inBackticks {
				// Start of template literal, replace with quote
				result.WriteByte('"')
				inBackticks = true
			} else {
				// End of template literal, replace with quote
				result.WriteByte('"')
				inBackticks = false
			}
			continue
		}

		if inBackticks {
			// Inside template literal, escape any quotes
			switch char {
			case '"':
				result.WriteString(`\"`)
			case '\n':
				result.WriteString(`\n`)
			case '\t':
				result.WriteString(`\t`)
			default:
				result.WriteByte(char)
			}
		} else {
			result.WriteByte(char)
		}
	}

	return result.String()
}

// convertTripleQuotes converts triple quotes to proper JSON strings
func (s *LLMService) convertTripleQuotes(jsonStr string) string {
	// Find patterns like: "key": """value""" and convert to "key": "value"
	result := strings.Builder{}
	i := 0

	for i < len(jsonStr) {
		// Look for triple quotes
		if i+2 < len(jsonStr) && jsonStr[i:i+3] == `"""` {
			// Found start of triple quote, replace with single quote
			result.WriteByte('"')
			i += 3

			// Find the closing triple quotes
			for i < len(jsonStr) {
				if i+2 < len(jsonStr) && jsonStr[i:i+3] == `"""` {
					// Found end of triple quote, replace with single quote
					result.WriteByte('"')
					i += 3
					break
				} else {
					// Inside triple quote, escape any quotes and handle newlines
					char := jsonStr[i]
					switch char {
					case '"':
						result.WriteString(`\"`)
					case '\n':
						result.WriteString(`\n`)
					case '\t':
						result.WriteString(`\t`)
					default:
						result.WriteByte(char)
					}
					i++
				}
			}
		} else {
			result.WriteByte(jsonStr[i])
			i++
		}
	}

	return result.String()
}

// convertSchemaField converts a JSON RawMessage to a string, handling both string and object formats
func (s *LLMService) convertSchemaField(raw json.RawMessage, target *string) error {
	if len(raw) == 0 {
		*target = ""
		return nil
	}

	// First try to unmarshal as a string
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		*target = str
		return nil
	}

	// If that fails, treat it as a JSON object and convert to string
	// This handles cases where the AI returns the schema as a JSON object
	var obj interface{}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return fmt.Errorf("schema field is neither string nor valid JSON object: %w", err)
	}

	// Convert the object back to a formatted JSON string
	schemaBytes, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to convert schema object to string: %w", err)
	}

	*target = string(schemaBytes)
	return nil
}

// convertDescriptionField converts description from various formats to string
func (s *LLMService) convertDescriptionField(raw json.RawMessage, target *string) error {
	if len(raw) == 0 {
		*target = ""
		return nil
	}

	// First try to unmarshal as a string
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		*target = str
		return nil
	}

	// Try to unmarshal as an object and extract meaningful text
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		var parts []string

		// Try to extract common fields in a meaningful order
		if title, ok := obj["title"].(string); ok {
			parts = append(parts, title)
		}
		if summary, ok := obj["summary"].(string); ok {
			parts = append(parts, summary)
		}
		if desc, ok := obj["description"].(string); ok {
			parts = append(parts, desc)
		}

		// Add any other string fields
		for key, value := range obj {
			if key != "title" && key != "summary" && key != "description" {
				if strVal, ok := value.(string); ok {
					parts = append(parts, fmt.Sprintf("%s: %s", key, strVal))
				}
			}
		}

		if len(parts) > 0 {
			*target = strings.Join(parts, ". ")
			return nil
		}

		// Fallback: convert entire object to JSON string
		objBytes, _ := json.Marshal(obj)
		*target = string(objBytes)
		return nil
	}

	return fmt.Errorf("description field is not a recognized format (expected string or object)")
}

// convertOptimizationsField converts optimizations from various formats to []string
func (s *LLMService) convertOptimizationsField(raw json.RawMessage, target *[]string) error {
	if len(raw) == 0 {
		*target = []string{}
		return nil
	}

	// First try to unmarshal as array of strings
	var strArray []string
	if err := json.Unmarshal(raw, &strArray); err == nil {
		*target = strArray
		return nil
	}

	// Try to unmarshal as array of objects with name/description
	var objArray []map[string]interface{}
	if err := json.Unmarshal(raw, &objArray); err == nil {
		var result []string
		for _, obj := range objArray {
			// Try to extract meaningful text from the object
			var text string
			if name, ok := obj["name"].(string); ok {
				text = name
				if desc, ok := obj["description"].(string); ok {
					text += ": " + desc
				}
			} else if desc, ok := obj["description"].(string); ok {
				text = desc
			} else {
				// Fallback: convert entire object to JSON string
				objBytes, _ := json.Marshal(obj)
				text = string(objBytes)
			}
			result = append(result, text)
		}
		*target = result
		return nil
	}

	// Try to unmarshal as a single object (key-value pairs)
	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil {
		var result []string
		for key, value := range obj {
			if strVal, ok := value.(string); ok {
				result = append(result, fmt.Sprintf("%s: %s", key, strVal))
			} else {
				// Convert non-string values to JSON
				valBytes, _ := json.Marshal(value)
				result = append(result, fmt.Sprintf("%s: %s", key, string(valBytes)))
			}
		}
		*target = result
		return nil
	}

	// Fallback: try as single string
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		*target = []string{str}
		return nil
	}

	return fmt.Errorf("optimizations field is not a recognized format (expected []string, []object, or object)")
}

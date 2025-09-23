package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	logpkg "pipegen/internal/log"
)

// Schema represents an AVRO schema
type Schema struct {
	Name      string        `json:"name"`
	Namespace string        `json:"namespace"`
	Type      string        `json:"type"`
	Content   string        `json:"-"` // Raw schema content
	Fields    []SchemaField `json:"fields,omitempty"`
	FilePath  string        `json:"-"`
}

// SchemaField represents a field in an AVRO schema
type SchemaField struct {
	Name string      `json:"name"`
	Type interface{} `json:"type"`
	Doc  string      `json:"doc,omitempty"`
}

// SchemaLoader handles loading and parsing AVRO schemas from files
type SchemaLoader struct {
	projectDir string
	logger     logpkg.Logger
}

// NewSchemaLoader creates a new schema loader
func NewSchemaLoader(projectDir string) *SchemaLoader {
	return &SchemaLoader{
		projectDir: projectDir,
		logger:     logpkg.Global(),
	}
}

// LoadSchemas loads all AVRO schemas from the schemas/ directory
func (loader *SchemaLoader) LoadSchemas() (map[string]*Schema, error) {
	schemaDir := filepath.Join(loader.projectDir, "schemas")

	// Check if schemas directory exists
	if _, err := os.Stat(schemaDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("schemas directory not found: %s", schemaDir)
	}

	// Read all schema files
	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read schemas directory: %w", err)
	}

	schemas := make(map[string]*Schema)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Support .avsc and .json files
		if !strings.HasSuffix(entry.Name(), ".avsc") && !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(schemaDir, entry.Name())
		schema, err := loader.loadSchema(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load schema from %s: %w", entry.Name(), err)
		}

		// Use filename without extension as key
		key := loader.getSchemaKey(entry.Name())
		schemas[key] = schema
	}

	if len(schemas) == 0 {
		return nil, fmt.Errorf("no AVRO schema files found in %s", schemaDir)
	}

	if loader.logger != nil {
		loader.logger.Info("loaded avro schemas", "count", len(schemas), "dir", schemaDir)
		for key, schema := range schemas {
			loader.logger.Debug("schema", "key", key, "namespace", schema.Namespace, "name", schema.Name)
		}
	}

	return schemas, nil
}

// loadSchema loads a single AVRO schema from a file
func (loader *SchemaLoader) loadSchema(filePath string) (*Schema, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	// Parse JSON schema
	var schemaData map[string]interface{}
	if err := json.Unmarshal(content, &schemaData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON schema: %w", err)
	}

	// Create Schema object
	schema := &Schema{
		Content:  string(content),
		FilePath: filePath,
	}

	// Extract basic schema information
	if name, ok := schemaData["name"].(string); ok {
		schema.Name = name
	}

	if namespace, ok := schemaData["namespace"].(string); ok {
		schema.Namespace = namespace
	}

	if schemaType, ok := schemaData["type"].(string); ok {
		schema.Type = schemaType
	}

	// Extract fields if present
	if fieldsData, ok := schemaData["fields"].([]interface{}); ok {
		for _, fieldData := range fieldsData {
			if fieldMap, ok := fieldData.(map[string]interface{}); ok {
				field := SchemaField{}

				if name, ok := fieldMap["name"].(string); ok {
					field.Name = name
				}

				if fieldType, ok := fieldMap["type"]; ok {
					field.Type = fieldType
				}

				if doc, ok := fieldMap["doc"].(string); ok {
					field.Doc = doc
				}

				schema.Fields = append(schema.Fields, field)
			}
		}
	}

	// Validate schema
	if err := loader.validateSchema(schema); err != nil {
		return nil, fmt.Errorf("schema validation failed: %w", err)
	}

	return schema, nil
}

// validateSchema performs basic validation on an AVRO schema
func (loader *SchemaLoader) validateSchema(schema *Schema) error {
	// Check required fields
	if schema.Name == "" {
		return fmt.Errorf("schema must have a name")
	}

	if schema.Type == "" {
		return fmt.Errorf("schema must have a type")
	}

	if schema.Type != "record" && schema.Type != "array" && schema.Type != "map" {
		return fmt.Errorf("unsupported schema type: %s", schema.Type)
	}

	// For record types, validate fields
	if schema.Type == "record" {
		if len(schema.Fields) == 0 {
			return fmt.Errorf("record schema must have fields")
		}

		// Check for duplicate field names
		fieldNames := make(map[string]bool)
		for _, field := range schema.Fields {
			if field.Name == "" {
				return fmt.Errorf("field must have a name")
			}

			if fieldNames[field.Name] {
				return fmt.Errorf("duplicate field name: %s", field.Name)
			}
			fieldNames[field.Name] = true
		}
	}

	// Validate JSON syntax by re-parsing
	var temp interface{}
	if err := json.Unmarshal([]byte(schema.Content), &temp); err != nil {
		return fmt.Errorf("invalid JSON syntax: %w", err)
	}

	return nil
}

// getSchemaKey extracts a key from the schema filename
func (loader *SchemaLoader) getSchemaKey(filename string) string {
	// Remove file extension
	key := strings.TrimSuffix(filename, ".avsc")
	key = strings.TrimSuffix(key, ".json")

	// Convert to consistent format
	key = strings.ToLower(key)
	key = strings.ReplaceAll(key, "_", "")
	key = strings.ReplaceAll(key, "-", "")

	// Map common patterns to standard keys
	if strings.Contains(key, "input") || strings.Contains(key, "event") {
		return "input"
	}
	if strings.Contains(key, "output") || strings.Contains(key, "result") {
		return "output"
	}

	return key
}

// GetSchemaSubjects returns Schema Registry subjects for the schemas
func (loader *SchemaLoader) GetSchemaSubjects(schemas map[string]*Schema, topicPrefix string) map[string]string {
	subjects := make(map[string]string)

	for key := range schemas {
		switch key {
		case "input":
			subjects[key] = fmt.Sprintf("%s-input-value", topicPrefix)
		case "output":
			subjects[key] = fmt.Sprintf("%s-output-value", topicPrefix)
		default:
			subjects[key] = fmt.Sprintf("%s-%s-value", topicPrefix, key)
		}
	}

	return subjects
}

// SchemaRegistry represents a connection to Confluent Schema Registry
type SchemaRegistry struct {
	URL       string
	APIKey    string
	APISecret string
}

// NewSchemaRegistry creates a new Schema Registry client
func NewSchemaRegistry(url, apiKey, apiSecret string) *SchemaRegistry {
	return &SchemaRegistry{
		URL:       url,
		APIKey:    apiKey,
		APISecret: apiSecret,
	}
}

// RegisterSchema registers a schema in Schema Registry
func (sr *SchemaRegistry) RegisterSchema(subject string, schema *Schema) (int, error) {
	// TODO: Implement actual Schema Registry API call
	// This is a placeholder for the actual implementation

	// Logging could be added by passing a logger; omitted to keep SchemaRegistry decoupled.

	// Here you would use the Schema Registry HTTP API to:
	// 1. Check compatibility with existing schemas
	// 2. Register the new schema
	// 3. Return the schema ID

	return 1, nil // Simulated schema ID
}

// GetSchema retrieves a schema from Schema Registry
func (sr *SchemaRegistry) GetSchema(subject string, version string) (*Schema, error) {
	// TODO: Implement actual Schema Registry API call
	return nil, fmt.Errorf("not implemented")
}

// CheckCompatibility checks if a schema is compatible with existing versions
func (sr *SchemaRegistry) CheckCompatibility(subject string, schema *Schema) (bool, error) {
	// TODO: Implement actual Schema Registry API call
	return true, nil // Simulated compatibility check
}

// SchemaReference represents a reference to a registered schema
type SchemaReference struct {
	Subject string `json:"subject"`
	Version int    `json:"version"`
	ID      int    `json:"id"`
}

// GenerateJavaClass generates Java POJO classes from AVRO schemas
func (schema *Schema) GenerateJavaClass() (string, error) {
	// TODO: Implement Java class generation from AVRO schema
	return "", fmt.Errorf("java class generation not implemented")
}

// GeneratePythonClass generates Python dataclasses from AVRO schemas
func (schema *Schema) GeneratePythonClass() (string, error) {
	// TODO: Implement Python class generation from AVRO schema
	return "", fmt.Errorf("python class generation not implemented")
}

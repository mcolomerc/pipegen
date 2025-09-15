package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pipegen/internal/templates"
)

// LLMContent represents AI-generated pipeline components
type LLMContent struct {
	InputSchema   string            `json:"input_schema"`
	OutputSchema  string            `json:"output_schema"`
	SQLStatements map[string]string `json:"sql_statements"`
	Description   string            `json:"description"`
	Optimizations []string          `json:"optimizations"`
}

// LLMProjectGenerator extends ProjectGenerator with AI-generated content
type LLMProjectGenerator struct {
	*ProjectGenerator
	llmContent *LLMContent
}

// NewProjectGeneratorWithLLM creates a generator with LLM-generated content
func NewProjectGeneratorWithLLM(projectName, projectPath string, localMode bool, content *LLMContent) (*LLMProjectGenerator, error) {
	base, err := NewProjectGenerator(projectName, projectPath, localMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create base project generator: %w", err)
	}

	return &LLMProjectGenerator{
		ProjectGenerator: base,
		llmContent: &LLMContent{
			InputSchema:   content.InputSchema,
			OutputSchema:  content.OutputSchema,
			SQLStatements: content.SQLStatements,
			Description:   content.Description,
			Optimizations: content.Optimizations,
		},
	}, nil
}

// Generate creates the project structure with LLM-generated content
func (g *LLMProjectGenerator) Generate() error {
	// Create project directory
	if err := os.MkdirAll(g.ProjectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate configuration (use base implementation)
	if err := g.generateConfig(); err != nil {
		return err
	}

	// Generate LLM-powered components (instead of base schemas and SQL)
	if err := g.generateLLMSchemas(); err != nil {
		return err
	}

	if err := g.generateLLMSQL(); err != nil {
		return err
	}

	// Generate Docker and Flink configuration files for local development
	if g.LocalMode {
		if err := g.generateDockerFiles(); err != nil {
			return err
		}
	}

	// Generate comprehensive README.md with AI-generated content
	if err := g.generateLLMREADME(); err != nil {
		return err
	}

	return nil
}

func (g *LLMProjectGenerator) generateLLMSchemas() error {
	schemasDir := filepath.Join(g.ProjectPath, "schemas")

	// Create schemas directory if it doesn't exist
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		return fmt.Errorf("failed to create schemas directory %s: %w", schemasDir, err)
	}

	// Verify directory was created
	if _, err := os.Stat(schemasDir); os.IsNotExist(err) {
		return fmt.Errorf("schemas directory %s was not created successfully", schemasDir)
	} else if err != nil {
		return fmt.Errorf("error checking schemas directory %s: %w", schemasDir, err)
	}

	// Determine which input schema to write: prefer explicit InputSchemaContent when set
	inputSchemaContent := g.llmContent.InputSchema
	if strings.TrimSpace(g.InputSchemaContent) != "" {
		inputSchemaContent = g.InputSchemaContent
	}

	// Check schema content before proceeding
	if strings.TrimSpace(inputSchemaContent) == "" {
		return fmt.Errorf("input schema content is empty - cannot generate schema file")
	}
	if strings.TrimSpace(g.llmContent.OutputSchema) == "" {
		return fmt.Errorf("output schema content is empty - cannot generate schema file")
	}

	// Write input schema from LLM (standardized to input.avsc)
	inputPath := filepath.Join(schemasDir, "input.avsc")
	if err := g.writeLLMSchema(inputPath, inputSchemaContent); err != nil {
		return fmt.Errorf("failed to write LLM input schema: %w", err)
	}

	// Write output schema from LLM (keep existing naming to avoid breaking downstream)
	outputPath := filepath.Join(schemasDir, "output_result.avsc")
	if err := g.writeLLMSchema(outputPath, g.llmContent.OutputSchema); err != nil {
		return fmt.Errorf("failed to write LLM output schema: %w", err)
	}

	return nil
}

func (g *LLMProjectGenerator) writeLLMSchema(path, schemaJSON string) error {
	// Check if schema content is empty
	if strings.TrimSpace(schemaJSON) == "" {
		return fmt.Errorf("schema content is empty for path: %s", path)
	}

	// Ensure the directory exists (defensive programming)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to ensure directory exists for %s: %w", path, err)
	}

	// Parse and pretty-print the schema JSON
	var schema interface{}
	if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
		return fmt.Errorf("invalid schema JSON from LLM for path %s: %w\nSchema content: %s", path, err, schemaJSON)
	}

	prettyJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format schema JSON for path %s: %w", path, err)
	}

	// Use absolute path to avoid any working directory issues
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}

	if err := os.WriteFile(absPath, prettyJSON, 0644); err != nil {
		return fmt.Errorf("failed to write schema file %s (abs: %s): %w", path, absPath, err)
	}

	return nil
}

// Helper function to truncate strings for debugging (add this if it doesn't exist)
func (g *LLMProjectGenerator) generateLLMSQL() error {
	sqlDir := filepath.Join(g.ProjectPath, "sql")

	// Create sql directory if it doesn't exist
	if err := os.MkdirAll(sqlDir, 0755); err != nil {
		return fmt.Errorf("failed to create sql directory: %w", err)
	}

	for filename, content := range g.llmContent.SQLStatements {
		filePath := filepath.Join(sqlDir, filename)

		// Add comment header to each SQL file
		sqlWithHeader := fmt.Sprintf(`-- %s
-- Generated by PipeGen AI
-- Description: %s
-- 
-- This SQL statement was generated based on your description.
-- Feel free to modify it according to your specific requirements.

%s`, filename, g.llmContent.Description, content)

		if err := writeFile(filePath, sqlWithHeader); err != nil {
			return fmt.Errorf("failed to write SQL file %s: %w", filename, err)
		}
	}

	// Create an additional file with optimization suggestions
	if len(g.llmContent.Optimizations) > 0 {
		optimizationsPath := filepath.Join(sqlDir, "OPTIMIZATIONS.md")
		optimizationsContent := g.buildOptimizationsDoc()
		if err := writeFile(optimizationsPath, optimizationsContent); err != nil {
			return fmt.Errorf("failed to write optimizations file: %w", err)
		}
	}

	return nil
}

func (g *LLMProjectGenerator) buildOptimizationsDoc() string {
	content := fmt.Sprintf(`# AI-Generated Optimization Suggestions

This file contains performance and design optimization suggestions generated by AI for your streaming pipeline.

## Pipeline Description
%s

## Optimization Recommendations

`, g.llmContent.Description)

	for i, opt := range g.llmContent.Optimizations {
		content += fmt.Sprintf("%d. %s\n\n", i+1, opt)
	}

	content += `## Next Steps

1. Review each SQL file in this directory
2. Test the generated pipeline with sample data
3. Apply optimizations based on your specific use case
4. Monitor performance and adjust as needed

## Resources

- [Apache Flink SQL Documentation](https://nightlies.apache.org/flink/flink-docs-release-1.17/docs/dev/table/sql/overview/)
- [Kafka Connector Documentation](https://nightlies.apache.org/flink/flink-docs-release-1.17/docs/connectors/table/kafka/)
- [PipeGen Documentation](../README.md)
`

	return content
}

// generateLLMREADME creates comprehensive documentation with AI-generated content details
func (g *LLMProjectGenerator) generateLLMREADME() error {
	templateData := templates.TemplateData{
		ProjectName:      g.ProjectName,
		ProjectNameTitle: toTitle(strings.ReplaceAll(g.ProjectName, "-", " ")),
		SanitizedName:    templates.SanitizeAVROIdentifier(g.ProjectName),
	}

	// Add LLM-specific data
	if g.llmContent != nil {
		templateData.Description = g.llmContent.Description
		templateData.Optimizations = g.llmContent.Optimizations
	}

	readmeContent, err := g.templateManager.RenderReadmeLLM(templateData)
	if err != nil {
		return fmt.Errorf("failed to render LLM README template: %w", err)
	}

	readmePath := filepath.Join(g.ProjectPath, "README.md")
	return os.WriteFile(readmePath, []byte(readmeContent), 0644)
}

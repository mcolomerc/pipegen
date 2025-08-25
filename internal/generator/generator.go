package generator

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"pipegen/internal/templates"
)

// sanitizeAVROIdentifier converts a string to a valid AVRO identifier
// AVRO identifiers must match [A-Za-z_][A-Za-z0-9_]*
func sanitizeAVROIdentifier(s string) string {
	// Replace hyphens and other invalid chars with underscores
	re := regexp.MustCompile(`[^A-Za-z0-9_]`)
	sanitized := re.ReplaceAllString(s, "_")

	// Ensure it starts with a letter or underscore
	if len(sanitized) > 0 && !regexp.MustCompile(`^[A-Za-z_]`).MatchString(sanitized) {
		sanitized = "_" + sanitized
	}

	// If empty or only invalid chars, provide a default
	if sanitized == "" || sanitized == "_" {
		sanitized = "pipeline"
	}

	return sanitized
}

// ProjectGenerator handles the creation of new streaming pipeline projects
type ProjectGenerator struct {
	ProjectName     string
	ProjectPath     string
	LocalMode       bool
	InputSchemaPath string
	templateManager *templates.Manager
}

// NewProjectGenerator creates a new project generator instance
func NewProjectGenerator(name, path string, localMode bool) (*ProjectGenerator, error) {
	templateManager, err := templates.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create template manager: %w", err)
	}

	return &ProjectGenerator{
		ProjectName:     name,
		ProjectPath:     path,
		LocalMode:       localMode,
		templateManager: templateManager,
	}, nil
}

// SetInputSchemaPath sets the path to a user-provided input schema
func (g *ProjectGenerator) SetInputSchemaPath(path string) {
	g.InputSchemaPath = path
}

// Generate creates the complete project structure
func (g *ProjectGenerator) Generate() error {
	// Create project directory
	if err := os.MkdirAll(g.ProjectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate all components
	if err := g.generateSQLStatements(); err != nil {
		return err
	}

	if err := g.generateAVROSchemas(); err != nil {
		return err
	}

	if err := g.generateConfig(); err != nil {
		return err
	}

	// Generate Docker and Flink configuration files for local development
	if g.LocalMode {
		if err := g.generateDockerFiles(); err != nil {
			return err
		}
	}

	// Generate README.md documentation
	if err := g.generateREADME(); err != nil {
		return err
	}

	return nil
}

func (g *ProjectGenerator) generateSQLStatements() error {
	sqlDir := filepath.Join(g.ProjectPath, "sql")
	if err := os.MkdirAll(sqlDir, 0755); err != nil {
		return fmt.Errorf("failed to create SQL directory: %w", err)
	}

	// Generate SQL statements based on mode
	sqlTemplates := g.getSQLTemplates()

	for filename, content := range sqlTemplates {
		filePath := filepath.Join(sqlDir, filename)
		if err := writeFile(filePath, content); err != nil {
			return err
		}
	}

	return nil
}

func (g *ProjectGenerator) generateAVROSchemas() error {
	schemasDir := filepath.Join(g.ProjectPath, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		return fmt.Errorf("failed to create schemas directory: %w", err)
	}

	// Handle input schema
	if g.InputSchemaPath != "" {
		// Copy user-provided schema
		if err := g.copyInputSchema(schemasDir); err != nil {
			return err
		}
	} else {
		// Generate default input schema
		if err := g.generateDefaultInputSchema(schemasDir); err != nil {
			return err
		}
	}

	// Always generate output schema
	if err := g.generateOutputSchema(schemasDir); err != nil {
		return err
	}

	return nil
}

func (g *ProjectGenerator) copyInputSchema(schemasDir string) error {
	// Read the user-provided schema
	inputFile, err := os.Open(g.InputSchemaPath)
	if err != nil {
		return fmt.Errorf("failed to open input schema: %w", err)
	}
	defer func() { _ = inputFile.Close() }()

	// Copy to input_event.avsc
	outputPath := filepath.Join(schemasDir, "input_event.avsc")
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create input schema file: %w", err)
	}
	defer func() { _ = outputFile.Close() }()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return fmt.Errorf("failed to copy schema file: %w", err)
	}

	fmt.Printf("📋 Using provided input schema: %s\n", g.InputSchemaPath)
	return nil
}

func (g *ProjectGenerator) generateDefaultInputSchema(schemasDir string) error {
	// Sanitize project name for AVRO namespace
	sanitizedName := sanitizeAVROIdentifier(g.ProjectName)

	templateData := templates.TemplateData{
		ProjectName:   g.ProjectName,
		SanitizedName: sanitizedName,
	}

	inputSchema, err := g.templateManager.RenderInputSchema(templateData)
	if err != nil {
		return fmt.Errorf("failed to render input schema template: %w", err)
	}

	inputSchemaPath := filepath.Join(schemasDir, "input.json")
	return writeFile(inputSchemaPath, inputSchema)
}

func (g *ProjectGenerator) generateOutputSchema(schemasDir string) error {
	// Sanitize project name for AVRO namespace
	sanitizedName := sanitizeAVROIdentifier(g.ProjectName)

	templateData := templates.TemplateData{
		ProjectName:   g.ProjectName,
		SanitizedName: sanitizedName,
	}

	outputSchema, err := g.templateManager.RenderOutputSchema(templateData)
	if err != nil {
		return fmt.Errorf("failed to render output schema template: %w", err)
	}

	outputSchemaPath := filepath.Join(schemasDir, "output.json")
	return writeFile(outputSchemaPath, outputSchema)
}

func (g *ProjectGenerator) getSQLTemplates() map[string]string {
	if g.LocalMode {
		return g.getLocalSQLTemplates()
	}
	return g.getCloudSQLTemplates()
}

func (g *ProjectGenerator) getLocalSQLTemplates() map[string]string {
	sqlTemplates, err := g.templateManager.RenderSQLFiles(true)
	if err != nil {
		// Fallback to empty map if template loading fails
		return make(map[string]string)
	}
	return sqlTemplates
}

func (g *ProjectGenerator) getCloudSQLTemplates() map[string]string {
	sqlTemplates, err := g.templateManager.RenderSQLFiles(false)
	if err != nil {
		// Fallback to empty map if template loading fails
		return make(map[string]string)
	}
	return sqlTemplates
}

func (g *ProjectGenerator) generateConfig() error {
	templateData := templates.TemplateData{
		ProjectName: g.ProjectName,
	}

	var configContent string
	var err error

	if g.LocalMode {
		configContent, err = g.templateManager.RenderLocalConfig(templateData)
	} else {
		configContent, err = g.templateManager.RenderCloudConfig(templateData)
	}

	if err != nil {
		return fmt.Errorf("failed to render config template: %w", err)
	}

	configPath := filepath.Join(g.ProjectPath, ".pipegen.yaml")
	return writeFile(configPath, configContent)
}

func (g *ProjectGenerator) generateDockerFiles() error {
	// Generate docker-compose.yml
	if err := g.generateDockerCompose(); err != nil {
		return err
	}

	// Generate flink-conf.yaml
	if err := g.generateFlinkConfig(); err != nil {
		return err
	}

	return nil
}

func (g *ProjectGenerator) generateDockerCompose() error {
	templateData := templates.TemplateData{
		WithSchemaRegistry: true, // Include Schema Registry by default
	}

	composeContent, err := g.templateManager.RenderDockerCompose(templateData)
	if err != nil {
		return fmt.Errorf("failed to render docker-compose template: %w", err)
	}

	composePath := filepath.Join(g.ProjectPath, "docker-compose.yml")
	return writeFile(composePath, composeContent)
}

func (g *ProjectGenerator) generateFlinkConfig() error {
	templateData := templates.TemplateData{}

	flinkConfig, err := g.templateManager.RenderFlinkConfig(templateData)
	if err != nil {
		return fmt.Errorf("failed to render Flink config template: %w", err)
	}

	flinkConfPath := filepath.Join(g.ProjectPath, "flink-conf.yaml")
	return writeFile(flinkConfPath, flinkConfig)
}

// generateREADME creates comprehensive documentation for the generated project
func (g *ProjectGenerator) generateREADME() error {
	templateData := templates.TemplateData{
		ProjectName:      g.ProjectName,
		ProjectNameTitle: toTitle(strings.ReplaceAll(g.ProjectName, "-", " ")),
		SanitizedName:    templates.SanitizeAVROIdentifier(g.ProjectName),
	}

	readmeContent, err := g.templateManager.RenderReadmeStandard(templateData)
	if err != nil {
		return fmt.Errorf("failed to render README template: %w", err)
	}

	readmePath := filepath.Join(g.ProjectPath, "README.md")
	return writeFile(readmePath, readmeContent)
}

// Helper function to write content to file
func writeFile(filePath, content string) error {
	return os.WriteFile(filePath, []byte(content), 0644)
}

// toTitle capitalizes the first letter of each word, replacing deprecated strings.Title
func toTitle(s string) string {
	if s == "" {
		return s
	}
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

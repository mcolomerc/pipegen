package generator

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

	// Note: Output schema is not generated as it will be registered automatically by Flink CREATE TABLE
	// This avoids double schema registration conflicts

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

	inputSchemaPath := filepath.Join(schemasDir, "input.avsc")
	return writeFile(inputSchemaPath, inputSchema)
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

	// Only use local config for now (no cloud integration)
	configContent, err := g.templateManager.RenderLocalConfig(templateData)
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

	// Generate flink-entrypoint.sh
	if err := g.generateFlinkEntrypoint(); err != nil {
		return err
	}

	// Download Flink connectors
	if err := g.generateConnectors(); err != nil {
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

func (g *ProjectGenerator) generateFlinkEntrypoint() error {
	templateData := templates.TemplateData{}

	flinkEntrypoint, err := g.templateManager.RenderFlinkEntrypoint(templateData)
	if err != nil {
		return fmt.Errorf("failed to render Flink entrypoint template: %w", err)
	}

	flinkEntrypointPath := filepath.Join(g.ProjectPath, "flink-entrypoint.sh")
	if err := writeFile(flinkEntrypointPath, flinkEntrypoint); err != nil {
		return err
	}

	// Make the script executable
	return os.Chmod(flinkEntrypointPath, 0755)
}

// generateConnectors downloads Flink connector JAR files from the connectors.txt template
func (g *ProjectGenerator) generateConnectors() error {
	connectorsDir := filepath.Join(g.ProjectPath, "connectors")
	if err := os.MkdirAll(connectorsDir, 0755); err != nil {
		return fmt.Errorf("failed to create connectors directory: %w", err)
	}

	// Get connectors.txt content from template
	connectorsContent, err := g.templateManager.RenderConnectors()
	if err != nil {
		return fmt.Errorf("failed to render connectors template: %w", err)
	}

	// Parse URLs from content
	urls := g.parseConnectorURLs(connectorsContent)
	if len(urls) == 0 {
		fmt.Println("⚠️  No connector URLs found in template")
		return nil
	}

	fmt.Printf("📥 Downloading %d Flink connectors...\n", len(urls))

	// Download each connector
	for i, connectorURL := range urls {
		if err := g.downloadConnector(connectorsDir, connectorURL, i+1, len(urls)); err != nil {
			fmt.Printf("⚠️  Failed to download connector %d/%d: %v\n", i+1, len(urls), err)
			continue
		}
	}

	fmt.Println("✅ Connector downloads completed")
	return nil
}

// parseConnectorURLs parses URLs from the connectors.txt content
func (g *ProjectGenerator) parseConnectorURLs(content string) []string {
	var urls []string
	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}

	return urls
}

// downloadConnector downloads a single connector JAR file
func (g *ProjectGenerator) downloadConnector(connectorsDir, connectorURL string, current, total int) error {
	// Parse URL to get filename
	parsedURL, err := url.Parse(connectorURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Extract filename from URL path
	filename := filepath.Base(parsedURL.Path)
	if filename == "" || filename == "." {
		return fmt.Errorf("could not extract filename from URL")
	}

	targetPath := filepath.Join(connectorsDir, filename)

	// Check if file already exists
	if _, err := os.Stat(targetPath); err == nil {
		fmt.Printf("  📦 %d/%d: %s (already exists)\n", current, total, filename)
		return nil
	}

	// Download the file
	fmt.Printf("  📥 %d/%d: %s\n", current, total, filename)

	resp, err := http.Get(connectorURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			fmt.Printf("failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create the target file
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("failed to close file: %v\n", err)
		}
	}()

	// Copy the response body to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
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

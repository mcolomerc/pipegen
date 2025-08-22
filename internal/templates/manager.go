package templates

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

//go:embed files
var templatesFS embed.FS

// TemplateData contains data for template rendering
type TemplateData struct {
	ProjectName        string
	ProjectNameTitle   string
	SanitizedName      string
	Description        string
	Optimizations      []string
	WithSchemaRegistry bool
}

// Manager handles template rendering
type Manager struct {
	templates map[string]*template.Template
}

// NewManager creates a new template manager
func NewManager() (*Manager, error) {
	m := &Manager{
		templates: make(map[string]*template.Template),
	}

	// Create custom template functions
	funcMap := template.FuncMap{
		"add": func(a, b int) int {
			return a + b
		},
	}

	// Load templates
	templatePaths := []struct {
		name string
		path string
	}{
		{"readme_standard", "files/readme/standard.md"},
		{"readme_llm", "files/readme/llm.md"},
		{"docker_compose", "files/docker/compose.yml"},
		{"flink_config", "files/config/flink-conf.yaml"},
		{"local_config", "files/config/local.yaml"},
		{"cloud_config", "files/config/cloud.yaml"},
		{"input_schema", "files/schemas/input.json"},
		{"output_schema", "files/schemas/output.json"},
	}

	for _, tp := range templatePaths {
		content, err := templatesFS.ReadFile(tp.path)
		if err != nil {
			return nil, fmt.Errorf("failed to read template %s: %w", tp.name, err)
		}

		tmpl, err := template.New(tp.name).Funcs(funcMap).Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template %s: %w", tp.name, err)
		}

		m.templates[tp.name] = tmpl
	}

	return m, nil
}

// RenderReadmeStandard renders the standard README template
func (m *Manager) RenderReadmeStandard(data TemplateData) (string, error) {
	return m.render("readme_standard", data)
}

// RenderReadmeLLM renders the LLM README template
func (m *Manager) RenderReadmeLLM(data TemplateData) (string, error) {
	return m.render("readme_llm", data)
}

// RenderDockerCompose renders the docker-compose template
func (m *Manager) RenderDockerCompose(data TemplateData) (string, error) {
	return m.render("docker_compose", data)
}

// RenderFlinkConfig renders the Flink configuration template
func (m *Manager) RenderFlinkConfig(data TemplateData) (string, error) {
	return m.render("flink_config", data)
}

// RenderLocalConfig renders the local configuration template
func (m *Manager) RenderLocalConfig(data TemplateData) (string, error) {
	return m.render("local_config", data)
}

// RenderCloudConfig renders the cloud configuration template
func (m *Manager) RenderCloudConfig(data TemplateData) (string, error) {
	return m.render("cloud_config", data)
}

// RenderInputSchema renders the input AVRO schema template
func (m *Manager) RenderInputSchema(data TemplateData) (string, error) {
	return m.render("input_schema", data)
}

// RenderOutputSchema renders the output AVRO schema template
func (m *Manager) RenderOutputSchema(data TemplateData) (string, error) {
	return m.render("output_schema", data)
}

// RenderSQLFiles renders SQL templates from the SQL directory
func (m *Manager) RenderSQLFiles(localMode bool) (map[string]string, error) {
	var sqlDir string
	if localMode {
		sqlDir = "files/sql/local"
	} else {
		sqlDir = "files/sql/cloud"
	}

	// Read all SQL files from the directory
	entries, err := templatesFS.ReadDir(sqlDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read SQL directory %s: %w", sqlDir, err)
	}

	sqlTemplates := make(map[string]string)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		filePath := fmt.Sprintf("%s/%s", sqlDir, entry.Name())
		content, err := templatesFS.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read SQL file %s: %w", filePath, err)
		}

		sqlTemplates[entry.Name()] = string(content)
	}

	return sqlTemplates, nil
}

// render executes a template with the given data
func (m *Manager) render(templateName string, data TemplateData) (string, error) {
	tmpl, exists := m.templates[templateName]
	if !exists {
		return "", fmt.Errorf("template %s not found", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return buf.String(), nil
}

// SanitizeAVROIdentifier sanitizes a string for use as an AVRO identifier
func SanitizeAVROIdentifier(name string) string {
	// Replace hyphens and other invalid characters with underscores
	sanitized := strings.ReplaceAll(name, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	// Remove any characters that aren't alphanumeric or underscores
	var result strings.Builder
	for _, r := range sanitized {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

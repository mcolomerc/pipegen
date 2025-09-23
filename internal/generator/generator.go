package generator

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"pipegen/internal/pipeline"
	"pipegen/internal/templates"
	"pipegen/internal/version"
)

// ProjectGenerator handles the creation of new streaming pipeline projects
type ProjectGenerator struct {
	ProjectName        string
	ProjectPath        string
	LocalMode          bool
	InputSchemaPath    string
	InputSchemaContent string
	InputCSVPath       string
	templateManager    *templates.Manager
}

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

// SetInputSchemaContent sets the raw AVRO schema JSON content to be written as input schema
func (g *ProjectGenerator) SetInputSchemaContent(content string) {
	g.InputSchemaContent = content
}

// SetInputCSVPath sets the path to a user-provided CSV file for schema inference
func (g *ProjectGenerator) SetInputCSVPath(path string) {
	g.InputCSVPath = path
}

// Generate creates the complete project structure
func (g *ProjectGenerator) Generate() error {
	// Create project directory
	if err := os.MkdirAll(g.ProjectPath, 0755); err != nil {
		return fmt.Errorf("failed to create project directory: %w", err)
	}

	// Generate all components
	// Generate schemas first (especially important for CSV mode where schema inference happens)
	if err := g.generateAVROSchemas(); err != nil {
		return err
	}

	// Then generate SQL statements (CSV source table generation depends on inferred schema)
	if err := g.generateSQLStatements(); err != nil {
		return err
	}

	// If an input schema is provided (path or content), generate a baseline source table DDL from it
	if g.InputSchemaPath != "" || strings.TrimSpace(g.InputSchemaContent) != "" {
		if err := g.generateSourceTableFromSchema(); err != nil {
			// Don't fail the whole generation on DDL synthesis issues; warn instead
			fmt.Printf("⚠️  Failed to generate source table from schema: %v\n", err)
		}
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

	// If CSV path provided, synthesize a filesystem source table (override default 01_create_source_table.sql)
	if g.InputCSVPath != "" {
		if err := g.generateCSVSourceTable(sqlDir); err != nil {
			fmt.Printf("⚠️  Failed to generate CSV source table: %v\n", err)
		}
	}

	return nil
}

// generateCSVSourceTable creates a Flink filesystem connector table referencing the provided CSV file.
func (g *ProjectGenerator) generateCSVSourceTable(sqlDir string) error {
	// We rely on the previously generated (or inferred) input.avsc schema to build the column list.
	loader := pipeline.NewSchemaLoader(g.ProjectPath)
	schemas, err := loader.LoadSchemas()
	if err != nil {
		return fmt.Errorf("load schemas: %w", err)
	}
	inputSchema := schemas["input"]
	if inputSchema == nil {
		return fmt.Errorf("input schema not found for CSV source table generation")
	}

	var cols []string
	for _, f := range inputSchema.Fields {
		// Basic mapping (reuse existing flink type mapper indirectly via avro types we already have)
		// We'll parse field.Type JSON representation; easier: call through existing helper by re-marshalling
		avroType := f.Type
		flinkType := g.flinkTypeFromAvroType(avroType)
		cols = append(cols, fmt.Sprintf("  `%s` %s", f.Name, flinkType))
	}

	// Derive a stable table name
	sanitized := sanitizeAVROIdentifier(g.ProjectName)
	tableName := fmt.Sprintf("%s_input_csv", strings.ToLower(sanitized))

	// Generate container path for Docker environment
	// Users should place their CSV file in the ./data/ directory of their project
	csvFileName := filepath.Base(g.InputCSVPath)
	containerPath := fmt.Sprintf("/opt/flink/data/input/%s", csvFileName)

	ddl := fmt.Sprintf(`-- Auto-generated CSV filesystem source table
-- NOTE: Place your CSV file in the ./data/ directory for Docker access
-- Host path: ./data/%s
-- Container path: %s
CREATE TABLE %s (
%s
) WITH (
  'connector' = 'filesystem',
  'path' = '%s',
  'format' = 'csv',
  'csv.ignore-parse-errors' = 'true'
);
`, csvFileName, containerPath, tableName, strings.Join(cols, ",\n"), containerPath)

	filePath := filepath.Join(sqlDir, "01_create_source_table.sql")
	if err := os.WriteFile(filePath, []byte(ddl), 0644); err != nil {
		return fmt.Errorf("write csv source table: %w", err)
	}
	fmt.Printf("🧩 Generated CSV filesystem source table DDL at %s\n", filePath)

	// Create data directory and copy CSV file for Docker access
	if err := g.setupCSVDataDirectory(csvFileName); err != nil {
		fmt.Printf("⚠️  Warning: Failed to setup data directory: %v\n", err)
		fmt.Printf("💡 Manual step: Copy your CSV file to ./data/%s before running\n", csvFileName)
	}

	return nil
}

// setupCSVDataDirectory creates the data directory and copies the CSV file
func (g *ProjectGenerator) setupCSVDataDirectory(csvFileName string) error {
	dataDir := filepath.Join(g.ProjectPath, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	// Copy the original CSV file to the project data directory
	srcFile, err := os.Open(g.InputCSVPath)
	if err != nil {
		return fmt.Errorf("open source CSV: %w", err)
	}
	defer func() {
		if closeErr := srcFile.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close source file: %v\n", closeErr)
		}
	}()

	dstPath := filepath.Join(dataDir, csvFileName)
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create destination CSV: %w", err)
	}
	defer func() {
		if closeErr := dstFile.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close destination file: %v\n", closeErr)
		}
	}()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy CSV file: %w", err)
	}

	fmt.Printf("📁 Copied CSV file to ./data/%s for Docker access\n", csvFileName)
	return nil
}

func (g *ProjectGenerator) generateAVROSchemas() error {
	schemasDir := filepath.Join(g.ProjectPath, "schemas")
	if err := os.MkdirAll(schemasDir, 0755); err != nil {
		return fmt.Errorf("failed to create schemas directory: %w", err)
	}

	// Handle input schema
	switch {
	case strings.TrimSpace(g.InputSchemaContent) != "":
		if err := g.writeInputSchemaContent(schemasDir, g.InputSchemaContent); err != nil {
			return err
		}
	case g.InputSchemaPath != "":
		// Copy user-provided schema
		if err := g.copyInputSchema(schemasDir); err != nil {
			return err
		}
	case g.InputCSVPath != "":
		if err := g.generateSchemaFromCSV(schemasDir); err != nil {
			return fmt.Errorf("failed to infer schema from CSV: %w", err)
		}
	default:
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

	// Copy to input.avsc for consistency
	outputPath := filepath.Join(schemasDir, "input.avsc")
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

// writeInputSchemaContent writes raw schema JSON content to schemas/input.avsc
func (g *ProjectGenerator) writeInputSchemaContent(schemasDir string, content string) error {
	outputPath := filepath.Join(schemasDir, "input.avsc")
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write input schema content: %w", err)
	}
	fmt.Printf("📋 Wrote provided input schema content to %s\n", outputPath)
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

// generateSchemaFromCSV performs a lightweight streaming analysis of the CSV file to infer an AVRO schema.
// This is intentionally conservative: it samples up to sampleLimit rows and infers primitive types only.
func (g *ProjectGenerator) generateSchemaFromCSV(schemasDir string) error {
	const sampleLimit = 500

	f, err := os.Open(g.InputCSVPath)
	if err != nil {
		return fmt.Errorf("open CSV: %w", err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.ReuseRecord = true
	r.FieldsPerRecord = -1 // allow variable

	// Read header
	header, err := r.Read()
	if err != nil {
		return fmt.Errorf("read header: %w", err)
	}
	if len(header) == 0 {
		return fmt.Errorf("empty CSV header")
	}

	colCount := len(header)
	// State trackers
	type colState struct {
		name     string
		isInt    bool
		isFloat  bool
		isBool   bool
		nullable bool
		sampled  int
	}
	cols := make([]*colState, colCount)
	for i, h := range header {
		name := strings.TrimSpace(h)
		if name == "" {
			name = fmt.Sprintf("col_%d", i+1)
		}
		// sanitize for Avro (reuse sanitizeAVROIdentifier but allow upper -> lower)
		name = sanitizeAVROIdentifier(strings.ToLower(name))
		cols[i] = &colState{name: name, isInt: true, isFloat: true, isBool: true}
	}

	for row := 0; row < sampleLimit; row++ {
		rec, e := r.Read()
		if e != nil {
			if e == io.EOF {
				break
			}
			// Any parse error: stop sampling early (best-effort inference)
			break
		}
		if len(rec) == 0 {
			break
		}
		for i := 0; i < colCount && i < len(rec); i++ {
			v := strings.TrimSpace(rec[i])
			if v == "" {
				cols[i].nullable = true
				continue
			}
			cols[i].sampled++
			if cols[i].isInt {
				if _, err := strconv.ParseInt(v, 10, 64); err != nil {
					cols[i].isInt = false
				}
			}
			if cols[i].isFloat {
				if _, err := strconv.ParseFloat(v, 64); err != nil {
					cols[i].isFloat = false
				}
			}
			if cols[i].isBool {
				lower := strings.ToLower(v)
				if lower != "true" && lower != "false" && lower != "0" && lower != "1" {
					cols[i].isBool = false
				}
			}
		}
	}

	// Build AVRO schema
	record := map[string]interface{}{
		"type":      "record",
		"name":      sanitizeAVROIdentifier(g.ProjectName) + "_input",
		"namespace": "pipegen.generated",
	}
	var fields []map[string]interface{}
	for _, c := range cols {
		avroType := "string"
		if c.isInt && c.sampled > 0 {
			avroType = "int"
		} else if c.isFloat && !c.isInt && c.sampled > 0 { // float only if not pure int
			avroType = "double"
		} else if c.isBool && c.sampled > 0 {
			avroType = "boolean"
		}
		var fieldType interface{}
		if c.nullable {
			fieldType = []interface{}{"null", avroType}
		} else {
			fieldType = avroType
		}
		fields = append(fields, map[string]interface{}{
			"name": c.name,
			"type": fieldType,
		})
	}
	record["fields"] = fields

	jsonBytes, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal inferred schema: %w", err)
	}

	outputPath := filepath.Join(schemasDir, "input.avsc")
	if err := os.WriteFile(outputPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("write inferred schema: %w", err)
	}

	fmt.Printf("🧪 Inferred AVRO schema from CSV (%d columns) -> %s\n", len(cols), outputPath)
	return nil
}

// generateSourceTableFromSchema synthesizes a baseline Flink SQL source table from the input AVRO schema
func (g *ProjectGenerator) generateSourceTableFromSchema() error {
	// Load schemas from project dir to reuse existing loader and flexible keying
	loader := pipeline.NewSchemaLoader(g.ProjectPath)
	schemas, err := loader.LoadSchemas()
	if err != nil {
		return fmt.Errorf("failed to load schemas: %w", err)
	}

	// Prefer the schema keyed as "input"; otherwise take any
	var schema *pipeline.Schema
	if s, ok := schemas["input"]; ok {
		schema = s
	} else {
		for _, s := range schemas {
			schema = s
			break
		}
	}
	if schema == nil {
		return fmt.Errorf("no input schema found to generate source table")
	}

	// Build columns DDL
	var cols []string
	for _, f := range schema.Fields {
		colType := g.flinkTypeFromAvroType(f.Type)
		// Quote column names with backticks
		cols = append(cols, fmt.Sprintf("  `%s` %s", f.Name, colType))
	}

	// Derive table and topic names from project
	sanitized := sanitizeAVROIdentifier(g.ProjectName)
	tableName := fmt.Sprintf("%s_input", strings.ToLower(sanitized))
	topicName := fmt.Sprintf("%s-input", strings.ToLower(sanitized))

	ddl := fmt.Sprintf(`-- Auto-generated from AVRO schema
CREATE TABLE %s (
%s
) WITH (
  'connector' = 'kafka',
  'topic' = '%s',
  'properties.bootstrap.servers' = 'broker:29092',
  'scan.startup.mode' = 'earliest-offset',
  'format' = 'avro-confluent',
  'avro-confluent.url' = 'http://schema-registry:8082'
);
`, tableName, strings.Join(cols, ",\n"), topicName)

	// Write to sql/01_create_source_table.sql (override existing template)
	sqlDir := filepath.Join(g.ProjectPath, "sql")
	if err := os.MkdirAll(sqlDir, 0755); err != nil {
		return fmt.Errorf("failed to ensure sql dir: %w", err)
	}
	target := filepath.Join(sqlDir, "01_create_source_table.sql")
	if err := os.WriteFile(target, []byte(ddl), 0644); err != nil {
		return fmt.Errorf("failed to write source table DDL: %w", err)
	}

	fmt.Printf("🧩 Generated source table DDL at %s\n", target)
	return nil
}

// flinkTypeFromAvroType maps AVRO field types to Flink SQL types (best-effort)
func (g *ProjectGenerator) flinkTypeFromAvroType(t interface{}) string {
	switch v := t.(type) {
	case string:
		switch v {
		case "string":
			return "STRING"
		case "int":
			return "INT"
		case "long":
			return "BIGINT"
		case "float":
			return "FLOAT"
		case "double":
			return "DOUBLE"
		case "boolean":
			return "BOOLEAN"
		case "bytes":
			return "BYTES"
		default:
			return "STRING"
		}
	case []interface{}:
		// Union types: pick first non-null
		for _, u := range v {
			if s, ok := u.(string); ok && s != "null" {
				return g.flinkTypeFromAvroType(s)
			}
			if m, ok := u.(map[string]interface{}); ok {
				return g.flinkTypeFromAvroType(m)
			}
		}
		return "STRING"
	case map[string]interface{}:
		// Handle logical types and complex types
		if lt, ok := v["logicalType"].(string); ok {
			switch lt {
			case "date":
				return "DATE"
			case "timestamp-millis", "timestamp-micros":
				return "TIMESTAMP(3)"
			case "time-millis", "time-micros":
				return "TIME(3)"
			}
		}
		if typ, ok := v["type"].(string); ok {
			switch typ {
			case "record":
				return "STRING"
			case "array":
				return "ARRAY<STRING>"
			case "map":
				return "MAP<STRING, STRING>"
			case "enum":
				return "STRING"
			default:
				return g.flinkTypeFromAvroType(typ)
			}
		}
		return "STRING"
	default:
		return "STRING"
	}
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

	// Build list dynamically from version constants.
	base := "https://repo1.maven.org/maven2"
	flinkFormat := version.FlinkFormatVersion
	compat := version.FlinkConnectorCompat
	kafkaConnVer := version.KafkaConnectorVersion

	urls := []string{
		fmt.Sprintf("%s/org/apache/flink/flink-connector-kafka/%s-%s/flink-connector-kafka-%s-%s.jar", base, kafkaConnVer, compat, kafkaConnVer, compat),
		fmt.Sprintf("%s/org/apache/flink/flink-sql-connector-kafka/%s-%s/flink-sql-connector-kafka-%s-%s.jar", base, kafkaConnVer, compat, kafkaConnVer, compat),
		fmt.Sprintf("%s/org/apache/flink/flink-sql-avro-confluent-registry/%s/flink-sql-avro-confluent-registry-%s.jar", base, version.AvroConfluentRegistryVersion, version.AvroConfluentRegistryVersion),
		fmt.Sprintf("%s/org/apache/kafka/kafka-clients/%s/kafka-clients-%s.jar", base, version.KafkaClientsVersion, version.KafkaClientsVersion),
		fmt.Sprintf("%s/com/fasterxml/jackson/core/jackson-core/%s/jackson-core-%s.jar", base, version.JacksonVersion, version.JacksonVersion),
		fmt.Sprintf("%s/com/fasterxml/jackson/core/jackson-databind/%s/jackson-databind-%s.jar", base, version.JacksonVersion, version.JacksonVersion),
		fmt.Sprintf("%s/com/fasterxml/jackson/core/jackson-annotations/%s/jackson-annotations-%s.jar", base, version.JacksonVersion, version.JacksonVersion),
		fmt.Sprintf("%s/com/google/guava/guava/%s/guava-%s.jar", base, version.GuavaVersion, version.GuavaVersion),
		fmt.Sprintf("%s/org/apache/flink/flink-json/%s/flink-json-%s.jar", base, flinkFormat, flinkFormat),
		fmt.Sprintf("%s/org/apache/flink/flink-csv/%s/flink-csv-%s.jar", base, flinkFormat, flinkFormat),
	}

	fmt.Printf("📥 Downloading %d Flink connectors (generated from version constants)...\n", len(urls))
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
// parseConnectorURLs removed since connectors are now generated dynamically from version constants

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

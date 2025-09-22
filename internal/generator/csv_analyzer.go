package generator

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// CSVAnalyzer performs streaming profiling and type inference on a CSV file.
type CSVAnalyzer struct {
	Path       string
	MaxRows    int
	MaxSamples int // number of example values to retain per column
	Delimiter  rune
	HasHeader  bool
	Timezone   *time.Location
}

// ColumnProfile holds inferred stats for a single column.
type ColumnProfile struct {
	Name         string
	Type         string // inferred canonical type: int, long, double, boolean, date, timestamp, string
	Nullable     bool
	RowCount     int
	NonNullCount int
	Samples      []string
	DistinctCap  int
	Distinct     map[string]int // value -> count (capped)
}

// AnalysisResult contains all columns and metadata.
type AnalysisResult struct {
	Columns       []*ColumnProfile
	TotalRows     int
	HeaderPresent bool
}

var (
	dateLayouts = []string{
		"2006-01-02",
		"02/01/2006",
		"01/02/2006",
	}
	timestampLayouts = []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02T15:04:05",
	}
	boolPattern = regexp.MustCompile(`^(?i:true|false|0|1|yes|no)$`)
)

// NewCSVAnalyzer creates a configured analyzer.
func NewCSVAnalyzer(path string) *CSVAnalyzer {
	return &CSVAnalyzer{
		Path:       path,
		MaxRows:    5000,
		MaxSamples: 5,
		Delimiter:  ',',
		HasHeader:  true,
		Timezone:   time.UTC,
	}
}

// Analyze performs streaming inference.
func (a *CSVAnalyzer) Analyze() (*AnalysisResult, error) {
	file, err := os.Open(a.Path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := file.Close(); cerr != nil {
			fmt.Printf("⚠️  failed to close CSV file: %v\n", cerr)
		}
	}()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.ReuseRecord = true
	reader.FieldsPerRecord = -1
	reader.Comma = a.Delimiter

	rowIndex := 0
	var header []string
	var cols []*ColumnProfile

	for {
		rec, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if rowIndex == 0 && a.HasHeader {
			header = rec
			cols = make([]*ColumnProfile, len(header))
			for i, h := range header {
				name := strings.TrimSpace(h)
				if name == "" {
					name = fmt.Sprintf("col_%d", i+1)
				}
				cols[i] = &ColumnProfile{Name: sanitizeAVROIdentifier(strings.ToLower(name)), Type: "int", Distinct: make(map[string]int), DistinctCap: 100}
			}
			rowIndex++
			continue
		}
		if cols == nil { // no header case
			cols = make([]*ColumnProfile, len(rec))
			for i := range rec {
				cols[i] = &ColumnProfile{Name: fmt.Sprintf("col_%d", i+1), Type: "int", Distinct: make(map[string]int), DistinctCap: 100}
			}
		}

		for i, v := range rec {
			if i >= len(cols) { // ignore extra cells if rows become longer
				continue
			}
			c := cols[i]
			c.RowCount++
			v = strings.TrimSpace(v)
			if v == "" {
				c.Nullable = true
				continue
			}
			c.NonNullCount++
			if len(c.Samples) < a.MaxSamples {
				c.Samples = append(c.Samples, v)
			}
			if len(c.Distinct) < c.DistinctCap {
				c.Distinct[v]++
			}
			inferColumnType(c, v, a.Timezone)
		}

		rowIndex++
		if rowIndex >= a.MaxRows {
			break
		}
	}

	res := &AnalysisResult{Columns: cols, TotalRows: rowIndex, HeaderPresent: a.HasHeader && len(header) > 0}
	finalizeTypes(res)
	return res, nil
}

func inferColumnType(c *ColumnProfile, v string, tz *time.Location) {
	// Escalation model: start at int; widen as needed.
	switch c.Type {
	case "int":
		if _, err := strconv.ParseInt(v, 10, 32); err == nil {
			return
		}
		if _, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.Type = "long"
			return
		}
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			c.Type = "double"
			return
		}
		if boolPattern.MatchString(v) {
			c.Type = "boolean"
			return
		}
		if parseDate(v) != nil {
			c.Type = "date"
			return
		}
		if parseTimestamp(v, tz) != nil {
			c.Type = "timestamp"
			return
		}
		c.Type = "string"
	case "long":
		if _, err := strconv.ParseInt(v, 10, 64); err == nil {
			return
		}
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			c.Type = "double"
			return
		}
		if boolPattern.MatchString(v) {
			c.Type = "string"
			return
		}
		if parseDate(v) != nil {
			c.Type = "string"
			return
		}
		if parseTimestamp(v, tz) != nil {
			c.Type = "string"
			return
		}
		c.Type = "string"
	case "double":
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			return
		}
		if boolPattern.MatchString(v) {
			c.Type = "string"
			return
		}
		if parseDate(v) != nil {
			c.Type = "string"
			return
		}
		if parseTimestamp(v, tz) != nil {
			c.Type = "string"
			return
		}
		c.Type = "string"
	case "boolean":
		if boolPattern.MatchString(v) {
			return
		}
		if parseDate(v) != nil || parseTimestamp(v, tz) != nil {
			c.Type = "string"
			return
		}
		if _, err := strconv.ParseFloat(v, 64); err == nil {
			c.Type = "double"
			return
		}
		c.Type = "string"
	case "date":
		if parseDate(v) != nil {
			return
		}
		if parseTimestamp(v, tz) != nil {
			c.Type = "timestamp"
			return
		}
		c.Type = "string"
	case "timestamp":
		if parseTimestamp(v, tz) != nil {
			return
		}
		c.Type = "string"
	case "string":
		return
	}
}

func parseDate(v string) *time.Time {
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return &t
		}
	}
	return nil
}

func parseTimestamp(v string, tz *time.Location) *time.Time {
	for _, layout := range timestampLayouts {
		if t, err := time.Parse(layout, v); err == nil {
			return &t
		}
		// Try with location
		if t, err := time.ParseInLocation(layout, v, tz); err == nil {
			return &t
		}
	}
	return nil
}

func finalizeTypes(res *AnalysisResult) {
	for _, c := range res.Columns {
		// If no non-null samples default to string nullable
		if c.NonNullCount == 0 {
			c.Type = "string"
			c.Nullable = true
		}
	}
}

// ExportAnalysisAsMarkdown returns a markdown summary for docs or prompts.
func ExportAnalysisAsMarkdown(res *AnalysisResult) string {
	var b strings.Builder
	b.WriteString("| Column | Type | Nullable | Non-Null | Distinct (<=10) | Samples |\n")
	b.WriteString("|--------|------|----------|----------|-----------------|---------|\n")
	for _, c := range res.Columns {
		var distinctVals []string
		count := 0
		for v := range c.Distinct {
			if count >= 10 {
				break
			}
			distinctVals = append(distinctVals, v)
			count++
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %t | %d | %s | %s |\n", c.Name, c.Type, c.Nullable, c.NonNullCount, strings.Join(distinctVals, ","), strings.Join(c.Samples, ",")))
	}
	return b.String()
}

// ExportAnalysisForPrompt produces a compact table suited for LLM prompts (limits columns and truncates samples).
func ExportAnalysisForPrompt(res *AnalysisResult, maxCols int) string {
	if maxCols <= 0 || maxCols > len(res.Columns) {
		maxCols = len(res.Columns)
	}
	var b strings.Builder
	b.WriteString("Columns (up to ")
	b.WriteString(strconv.Itoa(maxCols))
	b.WriteString("):\n")
	b.WriteString("name,type,nullable,sample_values\n")
	for i := 0; i < maxCols; i++ {
		c := res.Columns[i]
		// Combine up to 3 samples
		samples := c.Samples
		if len(samples) > 3 {
			samples = samples[:3]
		}
		sampleStr := strings.Join(samples, "|")
		// Sanitize commas/newlines
		sampleStr = strings.ReplaceAll(sampleStr, ",", " ")
		sampleStr = strings.ReplaceAll(sampleStr, "\n", " ")
		b.WriteString(fmt.Sprintf("%s,%s,%t,%s\n", c.Name, c.Type, c.Nullable, sampleStr))
	}
	if maxCols < len(res.Columns) {
		b.WriteString(fmt.Sprintf("... %d more columns omitted for brevity\n", len(res.Columns)-maxCols))
	}
	return b.String()
}

// GenerateAVROFromAnalysis maps the analysis result to an AVRO schema JSON (bytes)
func GenerateAVROFromAnalysis(projectName string, res *AnalysisResult) ([]byte, error) {
	var fields []map[string]interface{}
	for _, c := range res.Columns {
		avroType := mapInferredToAvro(c.Type)
		var fieldType interface{}
		if c.Nullable {
			fieldType = []interface{}{"null", avroType}
		} else {
			fieldType = avroType
		}
		fields = append(fields, map[string]interface{}{
			"name": c.Name,
			"type": fieldType,
		})
	}
	name := sanitizeAVROIdentifier(projectName) + "_input"
	record := map[string]interface{}{
		"type":      "record",
		"name":      name,
		"namespace": "pipegen.generated",
		"fields":    fields,
	}
	return jsonMarshalIndent(record)
}

func mapInferredToAvro(t string) string {
	switch t {
	case "int":
		return "int"
	case "long":
		return "long"
	case "double":
		return "double"
	case "boolean":
		return "boolean"
	case "date":
		return "string" // Could be logicalType date with extra structure; keep simple for now
	case "timestamp":
		return "string" // Could map to logical timestamp-millis; keep simple initially
	default:
		return "string"
	}
}

// jsonMarshalIndent local helper to avoid bringing encoding/json here earlier than needed
func jsonMarshalIndent(v interface{}) ([]byte, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	return data, err
}

// Integrate with existing generator inference: if future advanced inference is needed, we can replace existing logic
// by calling CSVAnalyzer then GenerateAVROFromAnalysis.

// Optionally provide a convenience wrapper.
func InferAVROFromCSV(projectName, path string, maxRows int) ([]byte, error) {
	an := NewCSVAnalyzer(path)
	if maxRows > 0 {
		an.MaxRows = maxRows
	}
	res, err := an.Analyze()
	if err != nil {
		return nil, err
	}
	return GenerateAVROFromAnalysis(projectName, res)
}

// Potential future: write profiling report to docs directory.
func WriteAnalysisReport(projectPath string, res *AnalysisResult) error {
	rel := filepath.Join(projectPath, "analysis.md")
	content := "# CSV Analysis\n\n" + ExportAnalysisAsMarkdown(res) + "\n"
	return os.WriteFile(rel, []byte(content), 0644)
}

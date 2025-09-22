package generator

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCSVAnalyzerBasic(t *testing.T) {
	path := filepath.Join("testdata", "simple.csv")
	an := NewCSVAnalyzer(path)
	res, err := an.Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if len(res.Columns) != 5 {
		t.Fatalf("expected 5 columns, got %d", len(res.Columns))
	}
	cases := map[string]string{
		"id":         "int",    // all small ints
		"name":       "string", // text
		"active":     "boolean",
		"score":      "double",    // float + empty
		"created_at": "timestamp", // mixed timestamp formats
	}
	for _, c := range res.Columns {
		if exp, ok := cases[c.Name]; ok {
			if c.Type != exp {
				t.Errorf("col %s expected type %s got %s", c.Name, exp, c.Type)
			}
		}
	}
	md := ExportAnalysisAsMarkdown(res)
	if md == "" {
		p := md
		_ = p
		t.Errorf("expected markdown output")
	}
}

func TestCSVAnalyzerNumericWiden(t *testing.T) {
	path := filepath.Join("testdata", "numeric_widen.csv")
	an := NewCSVAnalyzer(path)
	res, err := an.Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	if len(res.Columns) != 1 {
		t.Fatalf("expected 1 column got %d", len(res.Columns))
	}
	c := res.Columns[0]
	// 3,000,000,000 exceeds int32 range -> should widen to long
	if c.Type != "long" && c.Type != "string" { // allow string if escalation went further
		t.Fatalf("expected long or string, got %s", c.Type)
	}
}

func TestGenerateAVROFromAnalysis(t *testing.T) {
	path := filepath.Join("testdata", "simple.csv")
	an := NewCSVAnalyzer(path)
	res, err := an.Analyze()
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}
	bytes, err := GenerateAVROFromAnalysis("demo", res)
	if err != nil {
		t.Fatalf("GenerateAVROFromAnalysis failed: %v", err)
	}
	if len(bytes) == 0 {
		t.Fatalf("expected schema bytes")
	}
	// Basic sanity checks
	content := string(bytes)
	if !containsAll(content, []string{"\"type\": \"record\"", "\"fields\""}) {
		t.Errorf("schema content missing expected keys: %s", content)
	}
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}

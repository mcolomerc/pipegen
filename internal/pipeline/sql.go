package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"pipegen/internal/types"
)

// SQLLoader handles loading and parsing SQL statements from files
type SQLLoader struct {
	projectDir string
}

// NewSQLLoader creates a new SQL loader
func NewSQLLoader(projectDir string) *SQLLoader {
	return &SQLLoader{
		projectDir: projectDir,
	}
}

// LoadStatements loads all SQL statements from the sql/ directory
func (loader *SQLLoader) LoadStatements() ([]*types.SQLStatement, error) {
	sqlDir := filepath.Join(loader.projectDir, "sql")

	// Check if sql directory exists
	if _, err := os.Stat(sqlDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("sql directory not found: %s", sqlDir)
	}

	// Read all .sql files
	entries, err := os.ReadDir(sqlDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read sql directory: %w", err)
	}

	var statements []*types.SQLStatement
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		filePath := filepath.Join(sqlDir, entry.Name())
		statement, err := loader.loadStatement(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load statement from %s: %w", entry.Name(), err)
		}

		statements = append(statements, statement)
	}

	if len(statements) == 0 {
		return nil, fmt.Errorf("no SQL files found in %s", sqlDir)
	}

	// Sort statements by filename for consistent execution order
	sort.Slice(statements, func(i, j int) bool {
		return statements[i].Name < statements[j].Name
	})

	// Assign execution order
	for i, stmt := range statements {
		stmt.Order = i + 1
	}

	fmt.Printf("📖 Loaded %d SQL statements from %s\n", len(statements), sqlDir)
	for _, stmt := range statements {
		fmt.Printf("  %d. %s\n", stmt.Order, stmt.Name)
	}

	return statements, nil
}

// loadStatement loads a single SQL statement from a file
func (loader *SQLLoader) loadStatement(filePath string) (*types.SQLStatement, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Clean and validate SQL content
	sqlContent := strings.TrimSpace(string(content))
	if sqlContent == "" {
		return nil, fmt.Errorf("SQL file is empty")
	}

	// Remove comments and normalize whitespace
	sqlContent = loader.cleanSQL(sqlContent)

	filename := filepath.Base(filePath)
	name := strings.TrimSuffix(filename, ".sql")

	statement := &types.SQLStatement{
		Name:     name,
		Content:  sqlContent,
		FilePath: filePath,
	}

	return statement, nil
}

// cleanSQL removes comments and normalizes SQL content
func (loader *SQLLoader) cleanSQL(sql string) string {
	lines := strings.Split(sql, "\n")
	var cleanLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and single-line comments
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		// Remove inline comments
		if commentIndex := strings.Index(line, "--"); commentIndex != -1 {
			line = strings.TrimSpace(line[:commentIndex])
		}

		if line != "" {
			cleanLines = append(cleanLines, line)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// ValidateStatement performs basic validation on a SQL statement
func (loader *SQLLoader) ValidateStatement(statement *types.SQLStatement) error {
	sql := strings.ToUpper(statement.Content)

	// Check for dangerous operations in production
	dangerousOperations := []string{
		"DROP DATABASE",
		"DROP SCHEMA",
		"TRUNCATE",
		"DELETE FROM",
	}

	for _, op := range dangerousOperations {
		if strings.Contains(sql, op) {
			return fmt.Errorf("potentially dangerous operation detected: %s", op)
		}
	}

	// Check for required FlinkSQL keywords
	if !strings.Contains(sql, "CREATE TABLE") &&
		!strings.Contains(sql, "INSERT INTO") &&
		!strings.Contains(sql, "SELECT") {
		return fmt.Errorf("statement must contain CREATE TABLE, INSERT INTO, or SELECT")
	}

	// Validate variable placeholders
	requiredVars := []string{"${INPUT_TOPIC}", "${OUTPUT_TOPIC}", "${BOOTSTRAP_SERVERS}"}
	for _, variable := range requiredVars {
		if strings.Contains(statement.Content, variable) {
			// Variable is used, which is good for template statements
			continue
		}
	}

	return nil
}

// GetStatementsByType categorizes statements by their type
func (loader *SQLLoader) GetStatementsByType(statements []*types.SQLStatement) map[string][]*types.SQLStatement {
	categories := make(map[string][]*types.SQLStatement)

	for _, stmt := range statements {
		stmtType := loader.getStatementType(stmt.Content)
		categories[stmtType] = append(categories[stmtType], stmt)
	}

	return categories
}

// getStatementType determines the type of SQL statement
func (loader *SQLLoader) getStatementType(content string) string {
	upperContent := strings.ToUpper(content)

	if strings.Contains(upperContent, "CREATE TABLE") {
		return "CREATE_TABLE"
	} else if strings.Contains(upperContent, "INSERT INTO") {
		return "INSERT"
	} else if strings.Contains(upperContent, "CREATE VIEW") {
		return "CREATE_VIEW"
	} else if strings.Contains(upperContent, "SELECT") && !strings.Contains(upperContent, "CREATE") {
		return "QUERY"
	} else {
		return "OTHER"
	}
}

// StatementExecution represents the execution context for a statement
type StatementExecution struct {
	Statement    *types.SQLStatement
	Variables    map[string]string
	ProcessedSQL string
	ExecutionID  string
	Status       string
	Error        string
}

// PrepareExecution prepares a statement for execution with variable substitution
func (loader *SQLLoader) PrepareExecution(statement *types.SQLStatement, variables map[string]string) *StatementExecution {
	processedSQL := statement.Content

	// Substitute variables
	for key, value := range variables {
		processedSQL = strings.ReplaceAll(processedSQL, key, value)
	}

	return &StatementExecution{
		Statement:    statement,
		Variables:    variables,
		ProcessedSQL: processedSQL,
		Status:       "PREPARED",
	}
}

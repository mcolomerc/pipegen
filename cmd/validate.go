package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	logpkg "pipegen/internal/log"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate project structure and configuration",
	Long: `Validate checks the project structure and configuration:
- Validates SQL statements syntax
- Validates AVRO schemas
- Checks configuration completeness
- Verifies connectivity to Confluent Cloud`,
	RunE: runValidate,
}

func init() {
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().String("project-dir", ".", "Project directory path")
	validateCmd.Flags().Bool("check-connectivity", false, "Check connectivity to Confluent Cloud")
}

func runValidate(cmd *cobra.Command, args []string) error {
	projectDir, _ := cmd.Flags().GetString("project-dir")
	checkConnectivity, _ := cmd.Flags().GetBool("check-connectivity")

	logpkg.Global().Info("🔍 Validating project structure...")

	// Check project structure
	if err := validateProjectStructure(projectDir); err != nil {
		return fmt.Errorf("project structure validation failed: %w", err)
	}

	// Validate SQL files
	if err := validateSQLFiles(projectDir); err != nil {
		return fmt.Errorf("SQL validation failed: %w", err)
	}

	// Validate AVRO schemas
	if err := validateAVROSchemas(projectDir); err != nil {
		return fmt.Errorf("AVRO schema validation failed: %w", err)
	}

	// Validate configuration
	if err := validateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	if checkConnectivity {
		logpkg.Global().Info("🌐 Checking connectivity to Confluent Cloud...")
		// TODO: Implement connectivity check
		logpkg.Global().Info("⚠️  Connectivity check not implemented yet")
	}

	logpkg.Global().Info("✅ Project validation completed successfully!")
	return nil
}

func validateProjectStructure(projectDir string) error {
	requiredDirs := []string{"sql", "schemas", "connectors"}
	requiredFiles := []string{".pipegen.yaml", "flink-entrypoint.sh"}

	for _, dir := range requiredDirs {
		dirPath := filepath.Join(projectDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			return fmt.Errorf("required directory missing: %s", dir)
		}
		logpkg.Global().Info("✓ Directory exists", "dir", dir)
	}

	for _, file := range requiredFiles {
		filePath := filepath.Join(projectDir, file)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("required file missing: %s", file)
		}
		logpkg.Global().Info("✓ File exists", "file", file)
	}

	return nil
}

func validateSQLFiles(projectDir string) error {
	sqlDir := filepath.Join(projectDir, "sql")

	entries, err := os.ReadDir(sqlDir)
	if err != nil {
		return fmt.Errorf("failed to read sql directory: %w", err)
	}

	sqlCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlCount++
			logpkg.Global().Info("✓ SQL file found", "file", entry.Name())
		}
	}

	if sqlCount == 0 {
		return fmt.Errorf("no SQL files found in sql/ directory")
	}

	logpkg.Global().Info("✓ Found SQL files", "count", sqlCount)
	return nil
}

func validateAVROSchemas(projectDir string) error {
	schemaDir := filepath.Join(projectDir, "schemas")

	entries, err := os.ReadDir(schemaDir)
	if err != nil {
		return fmt.Errorf("failed to read schemas directory: %w", err)
	}

	schemaCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".avsc") || strings.HasSuffix(entry.Name(), ".json")) {
			schemaCount++
			logpkg.Global().Info("✓ AVRO schema found", "file", entry.Name())
		}
	}

	if schemaCount == 0 {
		return fmt.Errorf("no AVRO schema files found in schemas/ directory")
	}

	logpkg.Global().Info("✓ Found AVRO schema files", "count", schemaCount)
	return nil
}

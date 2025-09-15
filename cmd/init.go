package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"pipegen/internal/generator"
	"pipegen/internal/llm"
)

var initCmd = &cobra.Command{
	Use:   "init [project-name]",
	Short: "Initialize a new streaming pipeline project",
	Long: `Initialize creates a new streaming pipeline project with the following structure:
- sql/ directory with sample SQL statements
- schemas/ directory for AVRO schemas
- config/ directory for pipeline configuration
- Example producer and consumer configurations

You can provide your own input AVRO schema using the --input-schema flag.
The tool will copy your schema and generate the project structure accordingly.

For AI-powered generation, use --describe to let LLM generate optimized pipeline components:
  pipegen init my-pipeline --describe "Process user events and calculate hourly metrics"`,
	Args: cobra.ExactArgs(1),
	RunE: runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("force", false, "Overwrite existing project directory")
	initCmd.Flags().String("input-schema", "", "Path to existing AVRO schema file to use as input schema")
	initCmd.Flags().String("describe", "", "Natural language description of your streaming pipeline (requires PIPEGEN_OLLAMA_MODEL or PIPEGEN_OPENAI_API_KEY)")
	initCmd.Flags().String("domain", "", "Business domain for better AI context (e.g., ecommerce, fintech, iot)")
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	force, _ := cmd.Flags().GetBool("force")
	inputSchemaPath, _ := cmd.Flags().GetString("input-schema")
	description, _ := cmd.Flags().GetString("describe")
	domain, _ := cmd.Flags().GetString("domain")

	projectPath := filepath.Join(".", projectName)

	// Check if directory exists
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) && !force {
		return fmt.Errorf("directory %s already exists. Use --force to overwrite", projectPath)
	}

	// Allow combining --input-schema with --describe (Issue #8)
	// If both provided, we'll ground AI with the provided schema content

	// Validate input schema file if provided
	if inputSchemaPath != "" {
		if _, err := os.Stat(inputSchemaPath); os.IsNotExist(err) {
			return fmt.Errorf("input schema file not found: %s", inputSchemaPath)
		}
	}

	fmt.Printf("Initializing streaming pipeline project: %s\n", projectName)

	// Check if local mode is enabled from config (defaults to true)
	localMode := viper.GetBool("local_mode")

	// Initialize LLM service for AI-powered generation
	llmService := llm.NewLLMService()

	// Handle AI-powered generation
	if description != "" {
		if !llmService.IsEnabled() {
			return fmt.Errorf("LLM service not available. Set PIPEGEN_OPENAI_API_KEY or PIPEGEN_OLLAMA_MODEL environment variable to use --describe feature")
		}

		fmt.Printf("🤖 Generating pipeline with AI assistance (%s)...\n", llmService.GetProvider())
		fmt.Printf("📝 Description: %s\n", description)
		if domain != "" {
			fmt.Printf("🏢 Domain: %s\n", domain)
		}

		// Generate content with LLM (optionally grounded with input schema)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var generatedContent *llm.GeneratedContent
		var err error

		// If user provided an input schema, read it and ground the AI prompt with it
		if inputSchemaPath != "" {
			schemaBytes, readErr := os.ReadFile(inputSchemaPath)
			if readErr != nil {
				return fmt.Errorf("failed to read input schema file: %w", readErr)
			}
			generatedContent, err = llmService.GeneratePipelineWithSchema(ctx, string(schemaBytes), description, domain)
		} else {
			generatedContent, err = llmService.GeneratePipeline(ctx, description, domain)
		}
		if err != nil {
			return fmt.Errorf("AI generation failed: %w", err)
		}

		fmt.Println("✨ AI generation completed!")
		fmt.Printf("📊 Generated: %s\n", generatedContent.Description)

		// Create generator with LLM content
		// Prefer the user-provided schema as canonical input schema if available
		finalInputSchema := generatedContent.InputSchema
		if inputSchemaPath != "" {
			if b, e := os.ReadFile(inputSchemaPath); e == nil {
				finalInputSchema = string(b)
			}
		}

		llmContent := &generator.LLMContent{
			InputSchema:   finalInputSchema,
			OutputSchema:  generatedContent.OutputSchema,
			SQLStatements: generatedContent.SQLStatements,
			Description:   generatedContent.Description,
			Optimizations: generatedContent.Optimizations,
		}
		llmGen, err := generator.NewProjectGeneratorWithLLM(projectName, projectPath, localMode, llmContent)
		if err != nil {
			return fmt.Errorf("failed to create LLM generator: %w", err)
		}

		// Print optimizations
		if len(generatedContent.Optimizations) > 0 {
			fmt.Println("\n💡 AI Optimization Suggestions:")
			for _, opt := range generatedContent.Optimizations {
				fmt.Printf("  • %s\n", opt)
			}
		}

		// Generate the project using LLM generator
		if err := llmGen.Generate(); err != nil {
			return fmt.Errorf("failed to generate project: %w", err)
		}
	} else {
		// Standard generation
		if inputSchemaPath != "" {
			fmt.Printf("📋 Using provided input schema: %s\n", inputSchemaPath)
		} else {
			fmt.Println("📋 Generating default input schema (use --input-schema to provide your own)")
		}

		gen, err := generator.NewProjectGenerator(projectName, projectPath, localMode)
		if err != nil {
			return fmt.Errorf("failed to create generator: %w", err)
		}
		if inputSchemaPath != "" {
			gen.SetInputSchemaPath(inputSchemaPath)
		}

		// Generate the project using standard generator
		if err := gen.Generate(); err != nil {
			return fmt.Errorf("failed to generate project: %w", err)
		}
	}

	fmt.Printf("✅ Project %s initialized successfully!\n", projectName)
	fmt.Printf("📁 Project structure created at: %s\n", projectPath)

	// Print next steps
	printNextSteps(projectName, localMode, description != "")

	return nil
}

func printNextSteps(projectName string, localMode bool, isAIGenerated bool) {
	fmt.Println("\nNext steps:")
	fmt.Printf("1. cd %s\n", projectName)

	if localMode {
		fmt.Println("2. Review and customize the generated files:")
		fmt.Println("   • docker-compose.yml - Docker stack configuration")
		fmt.Println("   • flink-conf.yaml - Flink configuration")
		if isAIGenerated {
			fmt.Println("   • sql/ - AI-generated SQL statements")
			fmt.Println("   • schemas/ - AI-generated AVRO schemas")
			fmt.Println("3. Deploy local stack: pipegen deploy")
			fmt.Println("4. Run pipeline: pipegen run")
		} else {
			fmt.Println("   • sql/ - SQL statements for processing")
			fmt.Println("   • schemas/ - AVRO schemas for events")
			fmt.Println("3. Deploy local stack: pipegen deploy")
			fmt.Println("4. Run pipeline: pipegen run")
		}

		if !isAIGenerated {
			fmt.Println("\n💡 Try AI-powered generation:")
			fmt.Println("   • Ollama (local): export PIPEGEN_OLLAMA_MODEL=llama3.1")
			fmt.Println("   • OpenAI (cloud): export PIPEGEN_OPENAI_API_KEY=your-key")
		}
	}
}

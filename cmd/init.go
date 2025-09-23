package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pipegen/internal/generator"
	"pipegen/internal/llm"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	initCmd.Flags().String("input-csv", "", "Path to CSV file to use as source dataset (generates filesystem CSV source table)")
	initCmd.Flags().String("describe", "", "Natural language description of your streaming pipeline (requires PIPEGEN_OLLAMA_MODEL or PIPEGEN_OPENAI_API_KEY)")
	initCmd.Flags().String("domain", "", "Business domain for better AI context (e.g., ecommerce, fintech, iot)")
}

func runInit(cmd *cobra.Command, args []string) error {
	projectName := args[0]
	force, _ := cmd.Flags().GetBool("force")
	inputSchemaPath, _ := cmd.Flags().GetString("input-schema")
	description, _ := cmd.Flags().GetString("describe")
	domain, _ := cmd.Flags().GetString("domain")
	csvPath, _ := cmd.Flags().GetString("input-csv")
	projectPath := filepath.Join(".", projectName)

	if err := validateProjectDir(projectPath, force); err != nil {
		return err
	}
	if err := validateInputFile(inputSchemaPath, "input schema"); err != nil {
		return err
	}
	if err := validateInputFile(csvPath, "input CSV"); err != nil {
		return err
	}

	fmt.Printf("Initializing streaming pipeline project: %s\n", projectName)
	if csvPath != "" {
		fmt.Printf("📄 Using provided input CSV: %s\n", csvPath)
	}

	localMode := viper.GetBool("local_mode")
	llmService := llm.NewLLMService()

	var err error
	if description != "" {
		err = handleAIGeneration(projectName, projectPath, localMode, inputSchemaPath, csvPath, description, domain, llmService)
	} else {
		err = handleStandardGeneration(projectName, projectPath, localMode, inputSchemaPath, csvPath)
	}
	if err != nil {
		return err
	}

	fmt.Printf("✅ Project %s initialized successfully!\n", projectName)
	fmt.Printf("📁 Project structure created at: %s\n", projectPath)
	printNextSteps(projectName, localMode, description != "")
	return nil
}

func validateProjectDir(projectPath string, force bool) error {
	if _, err := os.Stat(projectPath); !os.IsNotExist(err) && !force {
		return fmt.Errorf("directory %s already exists. Use --force to overwrite", projectPath)
	}
	return nil
}

func validateInputFile(path, label string) error {
	if path == "" {
		return nil
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("%s file not found: %s", label, path)
	}
	return nil
}

func handleAIGeneration(projectName, projectPath string, localMode bool, inputSchemaPath, csvPath, description, domain string, llmService *llm.LLMService) error {
	if !llmService.IsEnabled() {
		fmt.Println("🤖 LLM service not available. Falling back to minimal generation.")
		if inputSchemaPath != "" {
			fmt.Println("📋 Using provided input schema for schema-based generation")
		} else {
			fmt.Println("📋 No input schema provided; generating with default schema and templates")
		}
		return handleStandardGeneration(projectName, projectPath, localMode, inputSchemaPath, csvPath)
	}

	fmt.Printf("🤖 Generating pipeline with AI assistance (%s)...\n", llmService.GetProvider())
	fmt.Printf("📝 Description: %s\n", description)
	if domain != "" {
		fmt.Printf("🏢 Domain: %s\n", domain)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var generatedContent *llm.GeneratedContent
	var err error
	switch {
	case inputSchemaPath != "":
		schemaBytes, readErr := os.ReadFile(inputSchemaPath)
		if readErr != nil {
			return fmt.Errorf("failed to read input schema file: %w", readErr)
		}
		generatedContent, err = llmService.GeneratePipelineWithSchema(ctx, string(schemaBytes), description, domain)
	case csvPath != "":
		an := generator.NewCSVAnalyzer(csvPath)
		res, aErr := an.Analyze()
		if aErr != nil {
			return fmt.Errorf("CSV analysis failed: %w", aErr)
		}
		analysisSummary := generator.ExportAnalysisForPrompt(res, 25)
		avroBytes, avErr := generator.GenerateAVROFromAnalysis(projectName, res)
		if avErr != nil {
			return fmt.Errorf("failed to generate AVRO from analysis: %w", avErr)
		}
		generatedContent, err = llmService.GeneratePipelineWithCSVAnalysis(ctx, description, domain, analysisSummary, string(avroBytes))
	default:
		generatedContent, err = llmService.GeneratePipeline(ctx, description, domain)
	}
	if err != nil {
		return fmt.Errorf("AI generation failed: %w", err)
	}

	fmt.Println("✨ AI generation completed!")
	fmt.Printf("📊 Generated: %s\n", generatedContent.Description)

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
	if inputSchemaPath != "" {
		if b, e := os.ReadFile(inputSchemaPath); e == nil {
			llmGen.SetInputSchemaContent(string(b))
		}
	}
	if csvPath != "" {
		llmGen.SetInputCSVPath(csvPath)
	}
	if len(generatedContent.Optimizations) > 0 {
		fmt.Println("\n💡 AI Optimization Suggestions:")
		for _, opt := range generatedContent.Optimizations {
			fmt.Printf("  • %s\n", opt)
		}
	}
	if csvPath != "" && inputSchemaPath == "" && strings.TrimSpace(llmGen.InputSchemaContent) == "" {
		llmGen.SetInputCSVPath(csvPath)
	}
	if err := llmGen.Generate(); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}
	return nil
}

func handleStandardGeneration(projectName, projectPath string, localMode bool, inputSchemaPath, csvPath string) error {
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
	if csvPath != "" {
		gen.SetInputCSVPath(csvPath)
	}
	if err := gen.Generate(); err != nil {
		return fmt.Errorf("failed to generate project: %w", err)
	}
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

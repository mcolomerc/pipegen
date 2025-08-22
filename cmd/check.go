package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"pipegen/internal/llm"
)

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Check AI provider configuration and connectivity",
	Long: `Check validates your AI setup and reports the status of configured providers.

This command helps troubleshoot AI-powered features by:
- Detecting configured AI providers (Ollama/OpenAI)
- Testing connectivity to AI services
- Verifying model availability
- Providing setup instructions`,
	RunE: runCheck,
}

func init() {
	rootCmd.AddCommand(checkCmd)
}

func runCheck(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Checking AI provider configuration...")

	llmService := llm.NewLLMService()

	if !llmService.IsEnabled() {
		fmt.Println("❌ No AI provider configured")
		fmt.Println("\n💡 To enable AI features, choose one:")
		fmt.Println("   • Ollama (local, free):")
		fmt.Println("     1. Install: curl -fsSL https://ollama.com/install.sh | sh")
		fmt.Println("     2. Pull model: ollama pull llama3.1")
		fmt.Println("     3. Set: export PIPEGEN_OLLAMA_MODEL=llama3.1")
		fmt.Println("   • OpenAI (cloud, requires API key):")
		fmt.Println("     1. Get API key from https://platform.openai.com/")
		fmt.Println("     2. Set: export PIPEGEN_OPENAI_API_KEY=your-key")
		return nil
	}

	fmt.Printf("✅ AI provider detected: %s\n", llmService.GetProviderInfo())

	// Test connectivity
	fmt.Println("🔗 Testing connectivity...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := llmService.CheckOllamaConnection(ctx); err != nil {
		fmt.Printf("❌ Connection failed: %v\n", err)
		return nil
	}

	fmt.Println("✅ AI provider is ready!")
	fmt.Println("\n💡 Try generating a pipeline:")
	fmt.Println("   pipegen init my-pipeline --describe \"your pipeline description\"")

	return nil
}

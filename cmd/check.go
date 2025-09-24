package cmd

import (
	"context"
	"time"

	"pipegen/internal/llm"
	logpkg "pipegen/internal/log"

	"github.com/spf13/cobra"
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
	logpkg.Global().Info("🔍 Checking AI provider configuration...")

	llmService := llm.NewLLMService()

	if !llmService.IsEnabled() {
		logpkg.Global().Info("❌ No AI provider configured")
		logpkg.Global().Info("\n💡 To enable AI features, choose one:")
		logpkg.Global().Info("   • Ollama (local, free):")
		logpkg.Global().Info("     1. Install: curl -fsSL https://ollama.com/install.sh | sh")
		logpkg.Global().Info("     2. Pull model: ollama pull llama3.1")
		logpkg.Global().Info("     3. Set: export PIPEGEN_OLLAMA_MODEL=llama3.1")
		logpkg.Global().Info("   • OpenAI (cloud, requires API key):")
		logpkg.Global().Info("     1. Get API key from https://platform.openai.com/")
		logpkg.Global().Info("     2. Set: export PIPEGEN_OPENAI_API_KEY=your-key")
		return nil
	}

	logpkg.Global().Info("✅ AI provider detected", "provider", llmService.GetProviderInfo())

	// Test connectivity
	logpkg.Global().Info("🔗 Testing connectivity...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := llmService.CheckOllamaConnection(ctx); err != nil {
		logpkg.Global().Warn("❌ Connection failed", "error", err)
		return nil
	}

	logpkg.Global().Info("✅ AI provider is ready!")
	logpkg.Global().Info("\n💡 Try generating a pipeline:")
	logpkg.Global().Info("   pipegen init my-pipeline --describe \"your pipeline description\"")

	return nil
}

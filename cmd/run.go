package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"pipegen/internal/dashboard"
	"pipegen/internal/pipeline"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the streaming pipeline",
	Long: `Run executes the complete streaming pipeline:
1. Creates dynamic Kafka topics
2. Deploys FlinkSQL statements  
3. Starts producer with configured message rate
4. Starts consumer for output validation
5. Generates detailed HTML execution reports (default)
6. Cleans up resources on completion

HTML execution reports include:
• Execution metrics and performance charts
• Parameter tracking and configuration details  
• Interactive visualizations using Chart.js
• Professional theme matching the dashboard

Reports are saved with timestamps to prevent overwrites.
Use --generate-report=false to disable report generation.`,
	RunE: runPipeline,
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags().String("project-dir", ".", "Project directory path")
	runCmd.Flags().Int("message-rate", 100, "Messages per second for producer")
	runCmd.Flags().Duration("duration", 5*time.Minute, "Pipeline execution duration")
	runCmd.Flags().Bool("cleanup", true, "Clean up resources after execution")
	runCmd.Flags().Bool("dry-run", false, "Show what would be executed without running")
	runCmd.Flags().Bool("dashboard", false, "Start live dashboard during pipeline execution")
	runCmd.Flags().Int("dashboard-port", 3000, "Dashboard server port")
	runCmd.Flags().Bool("generate-report", true, "Generate HTML execution report")
	runCmd.Flags().String("reports-dir", "", "Directory to save execution reports (default: project-dir/reports)")
}

func runPipeline(cmd *cobra.Command, args []string) error {
	projectDir, _ := cmd.Flags().GetString("project-dir")
	messageRate, _ := cmd.Flags().GetInt("message-rate")
	duration, _ := cmd.Flags().GetDuration("duration")
	cleanup, _ := cmd.Flags().GetBool("cleanup")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	useDashboard, _ := cmd.Flags().GetBool("dashboard")
	dashboardPort, _ := cmd.Flags().GetInt("dashboard-port")
	generateReport, _ := cmd.Flags().GetBool("generate-report")
	reportsDir, _ := cmd.Flags().GetString("reports-dir")

	// Validate configuration
	if err := validateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	config := &pipeline.Config{
		ProjectDir:        projectDir,
		MessageRate:       messageRate,
		Duration:          duration,
		Cleanup:           cleanup,
		DryRun:            dryRun,
		BootstrapServers:  viper.GetString("bootstrap_servers"),
		FlinkURL:          viper.GetString("flink_url"),
		SchemaRegistryURL: viper.GetString("schema_registry_url"),
		LocalMode:         viper.GetBool("local_mode"),
		GenerateReport:    generateReport,
		ReportsDir:        reportsDir,
	}

	if dryRun {
		fmt.Println("🔍 Dry run mode - showing execution plan:")
		return showExecutionPlan(config)
	}

	if useDashboard {
		fmt.Printf("🚀 Starting pipeline with live dashboard on port %d...\n", dashboardPort)
		return runWithDashboard(config, dashboardPort)
	}

	// Create pipeline runner
	runner, err := pipeline.NewRunner(config)
	if err != nil {
		return fmt.Errorf("failed to create pipeline runner: %w", err)
	}

	// Set up report generation if enabled
	if config.GenerateReport {
		reportGenerator := dashboard.NewExecutionReportGenerator(getReportsDir(config))
		runner.SetReportGenerator(reportGenerator)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n🛑 Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Run pipeline
	fmt.Println("🚀 Starting streaming pipeline...")
	if err := runner.Run(ctx); err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	fmt.Println("✅ Pipeline completed successfully!")
	return nil
}

func validateConfig() error {
	required := []string{
		"bootstrap_servers",
		"flink_url",
	}

	// Only require schema registry URL if not in local mode
	if !viper.GetBool("local_mode") {
		required = append(required, "schema_registry_url")
	}

	for _, key := range required {
		if viper.GetString(key) == "" {
			return fmt.Errorf("missing required configuration: %s", key)
		}
	}
	return nil
}

func showExecutionPlan(config *pipeline.Config) error {
	fmt.Println("📋 Execution Plan:")
	fmt.Printf("  Project Directory: %s\n", config.ProjectDir)
	fmt.Printf("  Message Rate: %d msg/sec\n", config.MessageRate)
	fmt.Printf("  Duration: %v\n", config.Duration)
	fmt.Printf("  Bootstrap Servers: %s\n", config.BootstrapServers)
	fmt.Printf("  Schema Registry: %s\n", config.SchemaRegistryURL)
	fmt.Printf("  FlinkSQL URL: %s\n", config.FlinkURL)
	fmt.Printf("  Local Mode: %t\n", config.LocalMode)
	fmt.Printf("  Cleanup Resources: %t\n", config.Cleanup)
	fmt.Println("\n📝 Steps that would be executed:")
	fmt.Println("  1. Load SQL statements from sql/ directory")
	fmt.Println("  2. Load AVRO schemas from schemas/ directory")
	fmt.Println("  3. Generate dynamic topic names")
	fmt.Println("  4. Create Kafka topics")
	fmt.Println("  5. Register AVRO schemas")
	fmt.Println("  6. Deploy FlinkSQL statements")
	fmt.Println("  7. Start Kafka producer")
	fmt.Println("  8. Start Kafka consumer")
	fmt.Println("  9. Monitor pipeline execution")
	if config.Cleanup {
		fmt.Println("  10. Clean up resources")
	}
	return nil
}

// runWithDashboard runs the pipeline with integrated dashboard
func runWithDashboard(config *pipeline.Config, dashboardPort int) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create dashboard server
	dashboardServer := dashboard.NewDashboardServer(dashboardPort)

	// Configure metrics collector
	kafkaAddrs := []string{config.BootstrapServers}
	dashboardServer.GetMetricsCollector().Configure(kafkaAddrs, config.FlinkURL, config.SchemaRegistryURL)

	// Start dashboard server
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- dashboardServer.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Open browser
	url := fmt.Sprintf("http://localhost:%d", dashboardPort)
	fmt.Printf("🌐 Opening dashboard in browser: %s\n", url)
	if err := openBrowser(url); err != nil {
		fmt.Printf("⚠️  Failed to open browser automatically: %v\n", err)
		fmt.Printf("💡 Please open %s manually in your browser\n", url)
	}

	// Create pipeline runner
	runner, err := pipeline.NewRunner(config)
	if err != nil {
		return fmt.Errorf("failed to create pipeline runner: %w", err)
	}

	// Set dashboard server for SQL statement tracking
	runner.SetDashboardServer(dashboardServer)

	// Set up report generation if enabled
	if config.GenerateReport {
		reportGenerator := dashboard.NewExecutionReportGenerator(getReportsDir(config))
		runner.SetReportGenerator(reportGenerator)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start pipeline with dashboard integration
	pipelineDone := make(chan error, 1)
	go func() {
		// Initialize dashboard status
		status := &dashboard.PipelineStatus{
			StartTime:    time.Now(),
			Status:       "RUNNING",
			Duration:     0,
			KafkaMetrics: &dashboard.KafkaMetrics{Topics: make(map[string]*dashboard.TopicMetrics)},
			FlinkMetrics: &dashboard.FlinkMetrics{
				Jobs:          make(map[string]*dashboard.FlinkJob),
				SQLStatements: make(map[string]*dashboard.FlinkStatement),
			},
			ProducerMetrics: &dashboard.ProducerMetrics{Status: "STARTING"},
			ConsumerMetrics: &dashboard.ConsumerMetrics{Status: "STARTING"},
			ExecutionSummary: &dashboard.ExecutionSummary{
				DataQuality: &dashboard.DataQualityMetrics{},
				Performance: &dashboard.PerformanceMetrics{},
			},
			Errors:      []dashboard.PipelineError{},
			LastUpdated: time.Now(),
		}
		dashboardServer.UpdatePipelineStatus(status)

		// Run the pipeline
		pipelineDone <- runner.Run(ctx)
	}()

	// Wait for completion or shutdown
	select {
	case <-sigChan:
		fmt.Println("\n🛑 Received interrupt signal, shutting down...")
		cancel()
	case err := <-pipelineDone:
		if err != nil {
			fmt.Printf("❌ Pipeline execution failed: %v\n", err)
		} else {
			fmt.Println("✅ Pipeline completed successfully!")
		}
	case err := <-serverDone:
		if err != nil {
			fmt.Printf("❌ Dashboard server error: %v\n", err)
		}
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := dashboardServer.Stop(shutdownCtx); err != nil {
		fmt.Printf("⚠️  Error during dashboard shutdown: %v\n", err)
	}

	return nil
}

// getReportsDir returns the directory where reports should be saved
func getReportsDir(config *pipeline.Config) string {
	if config.ReportsDir != "" {
		return config.ReportsDir
	}
	return filepath.Join(config.ProjectDir, "reports")
}

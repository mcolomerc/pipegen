package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"pipegen/internal/dashboard"
	logpkg "pipegen/internal/log"
	"pipegen/internal/pipeline"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	runCmd.Flags().Int("message-rate", 100, "Messages per second to produce")
	runCmd.Flags().Duration("duration", 30*time.Second, "Producer execution duration")
	runCmd.Flags().Duration("pipeline-timeout", 5*time.Minute, "Overall pipeline timeout (independent of producer duration)")
	runCmd.Flags().Int64("expected-messages", 0, "Expected number of messages to consume before stopping (0 = auto-calculate from producer)")
	runCmd.Flags().Bool("cleanup", true, "Clean up created topics and schemas after execution")
	runCmd.Flags().Bool("dry-run", false, "Show what would be executed without running")
	runCmd.Flags().Bool("dashboard", false, "Start live dashboard during pipeline execution")
	runCmd.Flags().Int("dashboard-port", 3000, "Dashboard server port")
	runCmd.Flags().Bool("generate-report", true, "Generate HTML execution report")
	runCmd.Flags().String("reports-dir", "", "Directory to save execution reports (default: project-dir/reports)")
	runCmd.Flags().String("traffic-pattern", "", "Define traffic peaks: 'start-end:rate%,start-end:rate%' (e.g., '30s-60s:300%,90s-120s:200%')")
	runCmd.Flags().Bool("global-tables", false, "Use global table creation mode (reuse session across pipeline runs)")
}

func runPipeline(cmd *cobra.Command, args []string) error {
	projectDir, _ := cmd.Flags().GetString("project-dir")
	messageRate, _ := cmd.Flags().GetInt("message-rate")
	duration, _ := cmd.Flags().GetDuration("duration")
	pipelineTimeout, _ := cmd.Flags().GetDuration("pipeline-timeout")
	expectedMessages, _ := cmd.Flags().GetInt64("expected-messages")
	cleanup, _ := cmd.Flags().GetBool("cleanup")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	useDashboard, _ := cmd.Flags().GetBool("dashboard")
	dashboardPort, _ := cmd.Flags().GetInt("dashboard-port")
	generateReport, _ := cmd.Flags().GetBool("generate-report")
	reportsDir, _ := cmd.Flags().GetString("reports-dir")
	trafficPatternStr, _ := cmd.Flags().GetString("traffic-pattern")
	globalTables, _ := cmd.Flags().GetBool("global-tables")

	// Validate configuration
	if err := validateConfig(); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Parse traffic patterns if provided
	var trafficPatterns *pipeline.TrafficPatterns
	var err error
	if trafficPatternStr != "" {
		trafficPatterns, err = pipeline.ParseTrafficPattern(trafficPatternStr, messageRate)
		if err != nil {
			return fmt.Errorf("invalid traffic pattern: %w", err)
		}

		// Validate that patterns fit within the execution duration
		if err := validateTrafficPatternDuration(trafficPatterns, duration); err != nil {
			return fmt.Errorf("traffic pattern validation failed: %w", err)
		}
	}

	config := &pipeline.Config{
		ProjectDir:        projectDir,
		MessageRate:       messageRate,
		Duration:          duration,
		PipelineTimeout:   pipelineTimeout,
		ExpectedMessages:  expectedMessages,
		Cleanup:           cleanup,
		DryRun:            dryRun,
		BootstrapServers:  viper.GetString("bootstrap_servers"),
		FlinkURL:          viper.GetString("flink_url"),
		SchemaRegistryURL: viper.GetString("schema_registry_url"),
		LocalMode:         viper.GetBool("local_mode"),
		GenerateReport:    generateReport,
		ReportsDir:        reportsDir,
		TrafficPatterns:   trafficPatterns,
		GlobalTables:      globalTables,
		KafkaConfig: pipeline.KafkaConfig{
			Partitions:        viper.GetInt("kafka_config.partitions"),
			ReplicationFactor: viper.GetInt("kafka_config.replication_factor"),
			RetentionMs:       viper.GetInt64("kafka_config.retention_ms"),
		},
	}

	// Detect CSV mode (filesystem connector CSV) by inspecting 01_create_source_table.sql if present
	createSourcePath := filepath.Join(projectDir, "sql", "01_create_source_table.sql")
	if b, err := os.ReadFile(createSourcePath); err == nil {
		contentUpper := strings.ToUpper(string(b))
		if strings.Contains(contentUpper, "'CONNECTOR' = 'FILESYSTEM'") && strings.Contains(contentUpper, "'FORMAT' = 'CSV'") {
			config.CSVMode = true
			logpkg.Global().Info("🧪 Detected filesystem CSV source table: enabling CSV mode (skip producer & consumer)")
		}
	}

	// --- NEW LOGIC: Check if stack is running, deploy if needed ---
	if !isDockerStackRunning(projectDir) {
		logpkg.Global().Info("🧹 Docker stack not running. Deploying stack...")
		deployCmd, _, _ := cmd.Root().Find([]string{"deploy"})
		if deployCmd == nil {
			return fmt.Errorf("deploy command not found")
		}
		deployArgs := []string{"--project-dir", projectDir}
		if err := deployCmd.RunE(deployCmd, deployArgs); err != nil {
			return fmt.Errorf("failed to deploy stack: %w", err)
		}
	} else {
		logpkg.Global().Info("✅ Docker stack is already running. Skipping deploy.")
	}
	// --- END NEW LOGIC ---

	if dryRun {
		logpkg.Global().Info("🔍 Dry run mode - showing execution plan:")
		return showExecutionPlan(config)
	}

	if useDashboard {
		logpkg.Global().Info("🚀 Starting pipeline with live dashboard", "port", dashboardPort)
		return runWithDashboard(config, dashboardPort)
	}

	// Create pipeline runner
	runner, err := pipeline.NewRunner(config)
	if err != nil {
		return fmt.Errorf("failed to create pipeline runner: %w", err)
	}

	// Set up report generation if enabled
	if config.GenerateReport {
		reportsPath := getReportsDir(config)
		logpkg.Global().Info("📊 Report generation enabled", "reports_dir", reportsPath)
		// Create reports directory if it does not exist
		if _, err := os.Stat(reportsPath); os.IsNotExist(err) {
			if err := os.MkdirAll(reportsPath, 0755); err != nil {
				return fmt.Errorf("failed to create reports directory: %w", err)
			}
			logpkg.Global().Info("[Runner] Created reports directory", "reports_dir", reportsPath)
		}
		// reportGenerator, err := dashboard.NewExecutionReportGenerator(reportsPath)
		// if err != nil {
		// 	return fmt.Errorf("failed to create report generator: %w", err)
		// }
		// runner.SetReportGenerator(reportGenerator) // Temporarily disabled
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logpkg.Global().Info("\n🛑 Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Run pipeline
	logpkg.Global().Info("🚀 Starting streaming pipeline...")
	if err := runner.Run(ctx); err != nil {
		return fmt.Errorf("pipeline execution failed: %w", err)
	}

	logpkg.Global().Info("✅ Pipeline completed successfully!")
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
	logpkg.Global().Info("📋 Execution Plan:")
	logpkg.Global().Info("  Project Directory:", "project_dir", config.ProjectDir)

	// Show traffic pattern information
	if config.TrafficPatterns != nil && config.TrafficPatterns.HasPatterns() {
		logpkg.Global().Info("  Traffic Pattern:")
		for _, line := range strings.Split(config.TrafficPatterns.GetPatternSummary(), "\n") {
			logpkg.Global().Info("    " + line)
		}
	} else {
		logpkg.Global().Info("  Message Rate", "rate_msg_per_sec", config.MessageRate)
	}
	logpkg.Global().Info("  Producer Duration", "duration", config.Duration)
	logpkg.Global().Info("  Pipeline Timeout", "timeout", config.PipelineTimeout)
	logpkg.Global().Info("  Bootstrap Servers", "bootstrap_servers", config.BootstrapServers)
	logpkg.Global().Info("  Schema Registry", "schema_registry", config.SchemaRegistryURL)
	logpkg.Global().Info("  FlinkSQL URL", "flink_url", config.FlinkURL)
	logpkg.Global().Info("  Local Mode", "local_mode", config.LocalMode)
	logpkg.Global().Info("  Cleanup Resources", "cleanup", config.Cleanup)
	logpkg.Global().Info("\n📝 Steps that would be executed:")
	logpkg.Global().Info("  1. Load SQL statements from sql/ directory")
	logpkg.Global().Info("  2. Load AVRO schemas from schemas/ directory")
	logpkg.Global().Info("  3. Generate dynamic topic names")
	logpkg.Global().Info("  4. Create Kafka topics")
	logpkg.Global().Info("  5. Register AVRO schemas")
	logpkg.Global().Info("  6. Deploy FlinkSQL statements")

	if config.TrafficPatterns != nil && config.TrafficPatterns.HasPatterns() {
		logpkg.Global().Info("  7. Start Kafka producer with dynamic traffic patterns")
	} else {
		logpkg.Global().Info("  7. Start Kafka producer with constant rate")
	}

	logpkg.Global().Info("  8. Start Kafka consumer")
	logpkg.Global().Info("  9. Monitor pipeline execution")
	if config.Cleanup {
		logpkg.Global().Info("  10. Clean up resources")
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
	logpkg.Global().Info("🌐 Opening dashboard in browser", "url", url)
	if err := openBrowser(url); err != nil {
		logpkg.Global().Warn("⚠️  Failed to open browser automatically", "error", err)
		logpkg.Global().Info("💡 Please open the dashboard manually", "url", url)
	}

	// Create pipeline runner
	runner, err := pipeline.NewRunner(config)
	if err != nil {
		return fmt.Errorf("failed to create pipeline runner: %w", err)
	}

	// Set dashboard server for SQL statement tracking
	// runner.SetDashboardServer(dashboardServer) // Temporarily disabled

	// Set up report generation if enabled
	if config.GenerateReport {
		// reportGenerator, err := dashboard.NewExecutionReportGenerator(getReportsDir(config))
		// if err != nil {
		// 	return fmt.Errorf("failed to create report generator: %w", err)
		// }
		// runner.SetReportGenerator(reportGenerator) // Temporarily disabled
		fmt.Println("📊 Report generation is currently disabled")
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
		logpkg.Global().Info("\n🛑 Received interrupt signal, shutting down...")
		cancel()
	case err := <-pipelineDone:
		if err != nil {
			logpkg.Global().Warn("❌ Pipeline execution failed", "error", err)
		} else {
			logpkg.Global().Info("✅ Pipeline completed successfully!")
		}
	case err := <-serverDone:
		if err != nil {
			logpkg.Global().Warn("❌ Dashboard server error", "error", err)
		}
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := dashboardServer.Stop(shutdownCtx); err != nil {
		logpkg.Global().Warn("⚠️  Error during dashboard shutdown", "error", err)
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

// validateTrafficPatternDuration ensures all traffic patterns fit within the execution duration
func validateTrafficPatternDuration(patterns *pipeline.TrafficPatterns, duration time.Duration) error {
	if patterns == nil || !patterns.HasPatterns() {
		return nil
	}

	for i, pattern := range patterns.Patterns {
		if pattern.EndTime > duration {
			return fmt.Errorf("pattern %d ends at %v, which exceeds execution duration of %v",
				i+1, pattern.EndTime, duration)
		}
		if pattern.StartTime >= duration {
			return fmt.Errorf("pattern %d starts at %v, which is at or beyond execution duration of %v",
				i+1, pattern.StartTime, duration)
		}
	}

	return nil
}

// isDockerStackRunning checks if the main containers are up
func isDockerStackRunning(projectDir string) bool {
	// This checks for running containers with docker compose ps
	cmd := exec.Command("docker", "compose", "ps", "--format", "json")
	cmd.Dir = projectDir
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), "running")
}

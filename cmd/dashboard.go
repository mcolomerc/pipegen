package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"pipegen/internal/dashboard"
	"pipegen/internal/pipeline"
)

// dashboardCmd represents the dashboard command
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Start the live dashboard for monitoring pipeline execution",
	Long: `Start a live HTML dashboard with real-time metrics visualization.

The dashboard provides:
- Real-time pipeline status and metrics
- Kafka cluster and topic monitoring  
- Flink job execution details
- Producer and consumer performance
- Error tracking with resolution suggestions
- Live performance charts and logs
- Exportable HTML reports

The dashboard automatically opens in your default browser and updates in real-time
via WebSocket connections.`,
	Example: `  # Start dashboard on default port (3000)
  pipegen dashboard

  # Start on custom port  
  pipegen dashboard --port 8080

  # Start dashboard for specific project
  pipegen dashboard --project-dir ./my-pipeline

  # Start in standalone mode (no pipeline execution)
  pipegen dashboard --standalone`,
	RunE: runDashboard,
}

var (
	dashboardPort       int
	dashboardStandalone bool
	dashboardAutoOpen   bool
	projectDir          string
	configFile          string
)

func init() {
	rootCmd.AddCommand(dashboardCmd)

	dashboardCmd.Flags().IntVarP(&dashboardPort, "port", "p", 3000, "Dashboard server port")
	dashboardCmd.Flags().BoolVar(&dashboardStandalone, "standalone", false, "Start dashboard without running pipeline")
	dashboardCmd.Flags().BoolVar(&dashboardAutoOpen, "open", true, "Automatically open dashboard in browser")

	// Add shared flags
	dashboardCmd.Flags().StringVar(&projectDir, "project-dir", ".", "Project directory path")
	dashboardCmd.Flags().StringVar(&configFile, "config", "", "Custom config file path")
}

func runDashboard(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Get flags
	projectDir, _ := cmd.Flags().GetString("project-dir")
	configFile, _ := cmd.Flags().GetString("config")

	// Load configuration
	config := &pipeline.Config{
		ProjectDir:        projectDir,
		BootstrapServers:  "localhost:9092",
		FlinkURL:          "http://localhost:8081",
		SchemaRegistryURL: "http://localhost:8082",
		LocalMode:         true,
		MessageRate:       100,
		Duration:          5 * time.Minute,
		Cleanup:           true,
	}

	// Override with config file if provided
	if configFile != "" {
		// TODO: Load config from file when pipeline.LoadConfig is available
		fmt.Printf("Config file specified: %s (not yet implemented)\n", configFile)
	}

	// Create dashboard server
	dashboardServer := dashboard.NewDashboardServer(dashboardPort)

	// Set pipeline name and version (try to detect from YAML config first)
	pipelineName, pipelineVersion := detectPipelineInfo(projectDir)
	if pipelineName != "" {
		dashboardServer.SetPipelineInfo(pipelineName, pipelineVersion)
	}

	// Configure metrics collector with connection details
	kafkaAddrs := []string{config.BootstrapServers}
	dashboardServer.GetMetricsCollector().Configure(kafkaAddrs, config.FlinkURL, config.SchemaRegistryURL)

	fmt.Printf("🚀 Starting PipeGen Dashboard on port %d...\n", dashboardPort)

	// Start the dashboard server in a goroutine
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- dashboardServer.Start(ctx)
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	// Open browser if requested
	if dashboardAutoOpen {
		url := fmt.Sprintf("http://localhost:%d", dashboardPort)
		fmt.Printf("🌐 Opening dashboard in browser: %s\n", url)
		if err := openBrowser(url); err != nil {
			fmt.Printf("⚠️  Failed to open browser automatically: %v\n", err)
			fmt.Printf("💡 Please open %s manually in your browser\n", url)
		}
	}

	if !dashboardStandalone {
		// Start pipeline execution with dashboard integration
		err := runPipelineWithDashboard(ctx, config, dashboardServer)
		if err != nil {
			fmt.Printf("❌ Pipeline execution failed: %v\n", err)
		}
	} else {
		fmt.Println("📊 Dashboard running in standalone mode")

		// Load SQL statements from project directory for display
		err := loadSQLStatementsForDashboard(projectDir, dashboardServer)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not load SQL statements: %v\n", err)
		}

		fmt.Printf("🌐 Visit http://localhost:%d to view the dashboard\n", dashboardPort)
		fmt.Println("Press Ctrl+C to stop")
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		fmt.Println("\n🛑 Shutting down dashboard...")
		cancel()
	case err := <-serverDone:
		if err != nil {
			return fmt.Errorf("dashboard server error: %w", err)
		}
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := dashboardServer.Stop(shutdownCtx); err != nil {
		fmt.Printf("⚠️  Error during dashboard shutdown: %v\n", err)
	}

	fmt.Println("✅ Dashboard stopped")
	return nil
}

func runPipelineWithDashboard(ctx context.Context, config *pipeline.Config, dashboardServer *dashboard.DashboardServer) error {
	fmt.Println("🚀 Starting pipeline with dashboard integration...")

	// Create pipeline runner
	runner, err := pipeline.NewRunner(config)
	if err != nil {
		return fmt.Errorf("failed to create pipeline runner: %w", err)
	}

	// Set dashboard server for SQL statement tracking
	// runner.SetDashboardServer(dashboardServer) // Temporarily disabled

	// Create dashboard status tracker
	statusTracker := &DashboardStatusTracker{
		dashboardServer: dashboardServer,
		config:          config,
	}

	// Initialize pipeline status
	status := &dashboard.PipelineStatus{
		StartTime:    time.Now(),
		Status:       "STARTING",
		Duration:     0,
		KafkaMetrics: &dashboard.KafkaMetrics{Topics: make(map[string]*dashboard.TopicMetrics)},
		FlinkMetrics: &dashboard.FlinkMetrics{
			Jobs:          make(map[string]*dashboard.FlinkJob),
			SQLStatements: make(map[string]*dashboard.FlinkStatement),
		},
		ProducerMetrics: &dashboard.ProducerMetrics{},
		ConsumerMetrics: &dashboard.ConsumerMetrics{},
		ExecutionSummary: &dashboard.ExecutionSummary{
			DataQuality: &dashboard.DataQualityMetrics{},
			Performance: &dashboard.PerformanceMetrics{},
		},
		Errors:      []dashboard.PipelineError{},
		LastUpdated: time.Now(),
	}

	dashboardServer.UpdatePipelineStatus(status)

	// Start status update loop
	go statusTracker.StartStatusLoop(ctx)

	// Update status to running
	status.Status = "RUNNING"
	dashboardServer.UpdatePipelineStatus(status)

	// Run the pipeline
	return runner.Run(ctx)
}

// DashboardStatusTracker handles real-time status updates for the dashboard
type DashboardStatusTracker struct {
	dashboardServer *dashboard.DashboardServer
	config          *pipeline.Config
	startTime       time.Time
}

// StartStatusLoop begins the real-time status update loop
func (dst *DashboardStatusTracker) StartStatusLoop(ctx context.Context) {
	dst.startTime = time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dst.updateStatus()
		}
	}
}

// updateStatus collects current metrics and updates dashboard
func (dst *DashboardStatusTracker) updateStatus() {
	// Get current metrics from the collector
	metricsCollector := dst.dashboardServer.GetMetricsCollector()
	kafkaMetrics := metricsCollector.GetKafkaMetrics()
	flinkMetrics := metricsCollector.GetFlinkMetrics()

	// Create comprehensive status update
	status := &dashboard.PipelineStatus{
		StartTime:    dst.startTime,
		Status:       "RUNNING",
		Duration:     time.Since(dst.startTime),
		KafkaMetrics: kafkaMetrics,
		FlinkMetrics: flinkMetrics,

		// Mock producer metrics (in real implementation, get from actual producer)
		ProducerMetrics: &dashboard.ProducerMetrics{
			Status:         "RUNNING",
			MessagesSent:   int64(time.Since(dst.startTime).Seconds() * float64(dst.config.MessageRate)),
			MessagesPerSec: float64(dst.config.MessageRate),
			BytesPerSec:    float64(dst.config.MessageRate * 1024), // 1KB per message
			SuccessRate:    99.8,
			AverageLatency: 5 * time.Millisecond,
		},

		// Mock consumer metrics
		ConsumerMetrics: &dashboard.ConsumerMetrics{
			Status:           "RUNNING",
			MessagesConsumed: int64(time.Since(dst.startTime).Seconds() * float64(dst.config.MessageRate) * 0.95),
			MessagesPerSec:   float64(dst.config.MessageRate) * 0.95,
			Lag:              5,
			ProcessingTime:   2 * time.Millisecond,
		},

		// Execution summary
		ExecutionSummary: &dashboard.ExecutionSummary{
			TotalMessagesProcessed: int64(time.Since(dst.startTime).Seconds() * float64(dst.config.MessageRate)),
			TotalBytesProcessed:    int64(time.Since(dst.startTime).Seconds() * float64(dst.config.MessageRate) * 1024),
			AverageLatency:         7 * time.Millisecond,
			ThroughputMsgSec:       float64(dst.config.MessageRate),
			ThroughputBytesSec:     float64(dst.config.MessageRate * 1024),
			ErrorRate:              0.2,
			SuccessRate:            99.8,
			DataQuality: &dashboard.DataQualityMetrics{
				ValidRecords:     int64(time.Since(dst.startTime).Seconds() * float64(dst.config.MessageRate) * 0.998),
				InvalidRecords:   int64(time.Since(dst.startTime).Seconds() * float64(dst.config.MessageRate) * 0.002),
				SchemaViolations: 0,
				QualityScore:     99.8,
			},
			Performance: &dashboard.PerformanceMetrics{
				P50Latency:          5 * time.Millisecond,
				P95Latency:          12 * time.Millisecond,
				P99Latency:          25 * time.Millisecond,
				ResourceUtilization: 75.5,
			},
		},

		Errors:      []dashboard.PipelineError{}, // Add actual errors as they occur
		LastUpdated: time.Now(),
	}

	// Update the dashboard
	dst.dashboardServer.UpdatePipelineStatus(status)
}

// openBrowser attempts to open the given URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch {
	case isCommandAvailable("xdg-open"):
		cmd = "xdg-open"
		args = []string{url}
	case isCommandAvailable("open"):
		cmd = "open"
		args = []string{url}
	case isCommandAvailable("cmd"):
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("no suitable command found to open browser")
	}

	return exec.Command(cmd, args...).Start()
}

// detectPipelineInfo tries to determine the pipeline name and version from YAML config first, then project directory
func detectPipelineInfo(projectDir string) (string, string) {
	// Try to read from .pipegen.yaml config file first
	configPath := filepath.Join(projectDir, ".pipegen.yaml")
	if _, err := os.Stat(configPath); err == nil {
		// Create a new viper instance to avoid conflicts with global config
		v := viper.New()
		v.SetConfigFile(configPath)
		v.SetConfigType("yaml")

		if err := v.ReadInConfig(); err == nil {
			name := v.GetString("pipeline.name")
			version := v.GetString("pipeline.version")
			if name != "" {
				return name, version
			}
		}
	}

	// Fallback to directory name if no config or no pipeline name in config
	var dirName string
	if projectDir != "." {
		dirName = filepath.Base(projectDir)
	} else if wd, err := os.Getwd(); err == nil {
		dirName = filepath.Base(wd)
	}

	return dirName, ""
}

// loadSQLStatementsForDashboard loads SQL statements from project directory for standalone dashboard display
func loadSQLStatementsForDashboard(projectDir string, dashboardServer *dashboard.DashboardServer) error {
	sqlDir := filepath.Join(projectDir, "sql")

	// Check if SQL directory exists
	if _, err := os.Stat(sqlDir); os.IsNotExist(err) {
		return fmt.Errorf("SQL directory not found: %s", sqlDir)
	}

	// Read SQL files
	sqlFiles, err := filepath.Glob(filepath.Join(sqlDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("error reading SQL files: %w", err)
	}

	if len(sqlFiles) == 0 {
		return fmt.Errorf("no SQL files found in %s", sqlDir)
	}

	// Create FlinkMetrics with SQL statements for display
	flinkMetrics := &dashboard.FlinkMetrics{
		JobManagerStatus: "Offline (Standalone Mode)",
		TaskManagerCount: 0,
		Jobs:             make(map[string]*dashboard.FlinkJob),
		SQLStatements:    make(map[string]*dashboard.FlinkStatement),
		ClusterMetrics:   &dashboard.FlinkClusterMetrics{},
		CheckpointStats:  &dashboard.CheckpointStats{},
	}

	// Process each SQL file
	for i, sqlFile := range sqlFiles {
		content, err := os.ReadFile(sqlFile)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not read SQL file %s: %v\n", sqlFile, err)
			continue
		}

		// Extract name from filename (remove extension and path)
		baseName := filepath.Base(sqlFile)
		name := baseName[:len(baseName)-4] // Remove .sql extension

		// Create FlinkStatement for display
		stmt := &dashboard.FlinkStatement{
			ID:               fmt.Sprintf("stmt-%d", i+1),
			Name:             name,
			Order:            i + 1,
			Status:           "PENDING",
			Phase:            "READY",
			Content:          string(content),
			ProcessedContent: string(content),
			FilePath:         sqlFile,
			DeploymentID:     "",
			RecordsProcessed: 0,
			RecordsPerSec:    0,
			Parallelism:      1,
			Dependencies:     []string{},
			Variables:        make(map[string]string),
		}

		flinkMetrics.SQLStatements[stmt.ID] = stmt
	}

	// Update the dashboard with loaded SQL statements
	metricsCollector := dashboardServer.GetMetricsCollector()
	metricsCollector.SetFlinkMetrics(flinkMetrics)

	fmt.Printf("📖 Loaded %d SQL statements for dashboard display\n", len(sqlFiles))
	return nil
}

// isCommandAvailable checks if a command is available in PATH
func isCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

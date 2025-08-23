package pipeline

import (
	"fmt"
	"path/filepath"
	"time"
	
	"pipegen/internal/dashboard"
)

// ExecutionReportIntegration handles the integration between pipeline runner and report generation
type ExecutionReportIntegration struct {
	runner      *Runner
	collector   *dashboard.ExecutionDataCollector
	generator   *dashboard.ExecutionReportGenerator
	executionID string
	startTime   time.Time
}

// NewExecutionReportIntegration creates a new integration instance
func NewExecutionReportIntegration(runner *Runner, reportsDir string) *ExecutionReportIntegration {
	generator := dashboard.NewExecutionReportGenerator(reportsDir)
	return &ExecutionReportIntegration{
		runner:    runner,
		generator: generator,
	}
}

// StartExecution initializes the execution tracking
func (e *ExecutionReportIntegration) StartExecution(config *Config) error {
	e.executionID = e.generateExecutionID()
	e.startTime = time.Now()
	
	// Create execution parameters
	params := dashboard.ExecutionParameters{
		MessageRate:       config.MessageRate,
		Duration:          config.Duration,
		BootstrapServers:  config.BootstrapServers,
		FlinkURL:          config.FlinkURL,
		SchemaRegistryURL: config.SchemaRegistryURL,
		LocalMode:         config.LocalMode,
		ProjectDir:        config.ProjectDir,
		Cleanup:           config.Cleanup,
	}
	
	// Initialize data collector
	e.collector = dashboard.NewExecutionDataCollector(e.executionID, params)
	
	fmt.Printf("📊 Started execution tracking with ID: %s\n", e.executionID)
	return nil
}

// UpdateMetrics updates the execution metrics
func (e *ExecutionReportIntegration) UpdateMetrics(producer *Producer, consumer *Consumer) {
	if e.collector == nil {
		return
	}
	
	var producerStats *dashboard.ProducerStats
	var consumerStats *dashboard.ConsumerStats
	
	if producer != nil && producer.GetStats() != nil {
		stats := producer.GetStats()
		producerStats = &dashboard.ProducerStats{
			MessagesSent:     stats.MessagesSent,
			BytesSent:        stats.BytesSent,
			MessagesPerSec:   stats.MessagesPerSec,
			ErrorCount:       stats.ErrorCount,
			LastMessageTime:  stats.LastMessageTime,
		}
	}
	
	if consumer != nil && consumer.GetStats() != nil {
		stats := consumer.GetStats()
		consumerStats = &dashboard.ConsumerStats{
			MessagesConsumed: stats.MessagesConsumed,
			BytesConsumed:    stats.BytesConsumed,
			MessagesPerSec:   stats.MessagesPerSec,
			ErrorCount:       stats.ErrorCount,
			LastMessageTime:  stats.LastMessageTime,
		}
	}
	
	e.collector.UpdateMetrics(producerStats, consumerStats)
}

// FinishExecution generates and saves the final report
func (e *ExecutionReportIntegration) FinishExecution(status string, pipelineName, pipelineVersion string) (string, error) {
	if e.collector == nil {
		return "", fmt.Errorf("execution not started")
	}
	
	// Set final status
	e.collector.SetStatus(status)
	
	// Generate report
	report := e.collector.GetCurrentReport(pipelineName, pipelineVersion)
	report.Duration = time.Since(e.startTime)
	
	// Save report to file
	reportPath, err := e.generator.GenerateReport(report)
	if err != nil {
		return "", fmt.Errorf("failed to generate report: %w", err)
	}
	
	fmt.Printf("✅ Execution report saved: %s\n", reportPath)
	return reportPath, nil
}

// generateExecutionID creates a unique execution ID
func (e *ExecutionReportIntegration) generateExecutionID() string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("exec-%s", timestamp)
}

// GetExecutionID returns the current execution ID
func (e *ExecutionReportIntegration) GetExecutionID() string {
	return e.executionID
}

// StartPeriodicMetricsCollection starts collecting metrics at regular intervals
func (e *ExecutionReportIntegration) StartPeriodicMetricsCollection(producer *Producer, consumer *Consumer) chan struct{} {
	if e.collector == nil {
		return nil
	}
	
	return e.collector.StartPeriodicCollection(producer, consumer, 5*time.Second)
}

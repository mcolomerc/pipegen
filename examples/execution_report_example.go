package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	
	"pipegen/internal/dashboard"
)

// Example demonstrating how to use the execution report feature
func main() {
	// Example execution parameters
	params := dashboard.ExecutionParameters{
		MessageRate:       1000,
		Duration:          5 * time.Minute,
		BootstrapServers:  "localhost:9092",
		FlinkURL:          "http://localhost:8081",
		SchemaRegistryURL: "http://localhost:8081",
		LocalMode:         false,
		ProjectDir:        "/tmp/my-pipeline",
		Cleanup:           true,
	}

	// Create execution data collector
	executionID := fmt.Sprintf("example-%d", time.Now().Unix())
	collector := dashboard.NewExecutionDataCollector(executionID, params)

	// Simulate execution metrics updates
	fmt.Println("🔄 Starting simulated pipeline execution...")
	
	// Update metrics over time (simulating real execution)
	for i := 0; i < 10; i++ {
		// Simulate producer metrics
		producerStats := &dashboard.ProducerStats{
			MessagesSent:    int64((i + 1) * 100),
			BytesSent:       int64((i + 1) * 100 * 512), // 512 bytes per message
			MessagesPerSec:  100.0,
			ErrorCount:      int64(i / 5), // Some errors
			LastMessageTime: time.Now(),
		}

		// Simulate consumer metrics
		consumerStats := &dashboard.ConsumerStats{
			MessagesConsumed: int64((i + 1) * 95), // Slightly less than produced
			BytesConsumed:    int64((i + 1) * 95 * 512),
			MessagesPerSec:   95.0,
			ErrorCount:       int64(i / 7),
			LastMessageTime:  time.Now(),
		}

		// Update collector
		collector.UpdateMetrics(producerStats, consumerStats)
		
		// Add some latency measurements
		collector.AddLatencyPoint(time.Duration(10+i*2) * time.Millisecond)
		
		fmt.Printf("⏱️  Step %d: %d messages processed\n", i+1, (i+1)*100)
		time.Sleep(500 * time.Millisecond) // Simulate time passing
	}

	// Mark as completed
	collector.SetStatus("completed")

	// Generate final report
	report := collector.GetCurrentReport("Example Pipeline", "v1.0.0")
	
	// Create reports directory
	reportsDir := "./example-reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		fmt.Printf("❌ Failed to create reports directory: %v\n", err)
		return
	}

	// Generate HTML report
	generator := dashboard.NewExecutionReportGenerator(reportsDir)
	reportPath, err := generator.GenerateReport(report)
	if err != nil {
		fmt.Printf("❌ Failed to generate report: %v\n", err)
		return
	}

	fmt.Printf("✅ Execution report generated successfully!\n")
	fmt.Printf("📄 Report saved to: %s\n", reportPath)
	fmt.Printf("🌐 Open this file in your browser to view the report\n")
	
	// Show some final statistics
	fmt.Println("\n📊 Execution Summary:")
	fmt.Printf("   • Execution ID: %s\n", report.ExecutionID)
	fmt.Printf("   • Status: %s\n", report.Status)
	fmt.Printf("   • Duration: %v\n", report.Duration)
	fmt.Printf("   • Total Messages: %d\n", report.Metrics.TotalMessages)
	fmt.Printf("   • Messages/Second: %.1f\n", report.Metrics.MessagesPerSecond)
	fmt.Printf("   • Success Rate: %.1f%%\n", report.Metrics.SuccessRate)
	fmt.Printf("   • Errors: %d\n", report.Metrics.ErrorCount)
}

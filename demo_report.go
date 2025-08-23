package main

import (
	"fmt"
	"os"
	"time"
	
	"pipegen/internal/dashboard"
)

func main() {
	fmt.Println("🚀 Creating execution report example...")
	
	// Create sample execution parameters
	params := dashboard.ExecutionParameters{
		MessageRate:       1000,
		Duration:          5 * time.Minute,
		BootstrapServers:  "localhost:9092",
		FlinkURL:          "http://localhost:8081",
		SchemaRegistryURL: "http://localhost:8082",
		LocalMode:         true,
		ProjectDir:        "/tmp/example-pipeline",
		Cleanup:           true,
	}
	
	// Create a sample execution report
	report := dashboard.ExecutionReport{
		Timestamp:   time.Now(),
		ExecutionID: fmt.Sprintf("demo-%d", time.Now().Unix()),
		Parameters:  params,
		Summary: dashboard.ExecutionSummary{
			TotalMessagesProcessed: 50000,
			TotalBytesProcessed:   50000 * 1024, // 50MB
			ThroughputMsgSec:      1666.7,       // 50k messages over 30s
			ThroughputBytesSec:    1666.7 * 1024, // bytes per second
			ErrorRate:             0.02,         // 0.02% error rate
			SuccessRate:           99.98,        // 99.98% success rate
			AverageLatency:        50 * time.Millisecond,
		},
		Metrics: dashboard.ExecutionMetrics{
			TotalMessages:     50000,
			MessagesPerSecond: 1666.7,
			BytesProcessed:    50000 * 1024,
			ErrorCount:        10, // 10 errors out of 50k messages
			SuccessRate:       99.98,
			AvgLatency:        50 * time.Millisecond,
		},
		Charts: dashboard.ChartData{
			ThroughputOverTime: []dashboard.TimeSeriesPoint{
				{Timestamp: time.Now().Add(-30 * time.Second), Value: 0},
				{Timestamp: time.Now().Add(-25 * time.Second), Value: 500},
				{Timestamp: time.Now().Add(-20 * time.Second), Value: 1200},
				{Timestamp: time.Now().Add(-15 * time.Second), Value: 1800},
				{Timestamp: time.Now().Add(-10 * time.Second), Value: 2000},
				{Timestamp: time.Now().Add(-5 * time.Second), Value: 1900},
				{Timestamp: time.Now(), Value: 1666},
			},
			LatencyOverTime: []dashboard.TimeSeriesPoint{
				{Timestamp: time.Now().Add(-30 * time.Second), Value: 0},
				{Timestamp: time.Now().Add(-25 * time.Second), Value: 45},
				{Timestamp: time.Now().Add(-20 * time.Second), Value: 52},
				{Timestamp: time.Now().Add(-15 * time.Second), Value: 48},
				{Timestamp: time.Now().Add(-10 * time.Second), Value: 55},
				{Timestamp: time.Now().Add(-5 * time.Second), Value: 49},
				{Timestamp: time.Now(), Value: 50},
			},
		},
		Status:   "completed",
		Duration: 30 * time.Second,
	}
	
	fmt.Println("📊 Sample execution report created with:")
	fmt.Printf("  • %d messages processed\n", report.Summary.TotalMessagesProcessed)
	fmt.Printf("  • %.1f messages/second average throughput\n", report.Summary.ThroughputMsgSec)
	fmt.Printf("  • %.2f%% success rate\n", report.Summary.SuccessRate)
	fmt.Printf("  • %v average latency\n", report.Summary.AverageLatency)
	
	// Generate the HTML report
	reportsDir := "/tmp/example-reports"
	
	// Ensure reports directory exists
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		fmt.Printf("❌ Failed to create reports directory: %v\n", err)
		return
	}
	
	generator := dashboard.NewExecutionReportGenerator(reportsDir)
	reportPath, err := generator.GenerateReport(&report)
	if err != nil {
		fmt.Printf("❌ Failed to generate report: %v\n", err)
		return
	}
	
	fmt.Printf("📄 Execution report generated: %s\n", reportPath)
	fmt.Printf("🌐 You can view the report by opening: file://%s\n", reportPath)
	
	// Also show file size
	if info, err := os.Stat(reportPath); err == nil {
		fmt.Printf("📋 Report file size: %d bytes\n", info.Size())
	}
}

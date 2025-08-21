package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"pipegen/internal/pipeline"
)

// MetricsCollector collects metrics from various pipeline components
type MetricsCollector struct {
	kafkaMetrics    *KafkaMetrics
	flinkMetrics    *FlinkMetrics
	metricsLock     sync.RWMutex
	
	// Configuration
	kafkaAddrs      []string
	flinkURL        string
	schemaRegistryURL string
	
	// Collection intervals
	kafkaInterval   time.Duration
	flinkInterval   time.Duration
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		kafkaMetrics:    &KafkaMetrics{Topics: make(map[string]*TopicMetrics)},
		flinkMetrics:    &FlinkMetrics{
			Jobs:          make(map[string]*FlinkJob),
			SQLStatements: make(map[string]*FlinkStatement),
		},
		kafkaInterval:   2 * time.Second,
		flinkInterval:   3 * time.Second,
	}
}

// Configure sets up the metrics collector with connection details
func (mc *MetricsCollector) Configure(kafkaAddrs []string, flinkURL, schemaRegistryURL string) {
	mc.kafkaAddrs = kafkaAddrs
	mc.flinkURL = flinkURL
	mc.schemaRegistryURL = schemaRegistryURL
}

// Start begins metrics collection
func (mc *MetricsCollector) Start(ctx context.Context) {
	// Start Kafka metrics collection
	go mc.collectKafkaMetrics(ctx)
	
	// Start Flink metrics collection
	go mc.collectFlinkMetrics(ctx)
}

// collectKafkaMetrics periodically collects Kafka cluster and topic metrics
func (mc *MetricsCollector) collectKafkaMetrics(ctx context.Context) {
	ticker := time.NewTicker(mc.kafkaInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := mc.updateKafkaMetrics(); err != nil {
				fmt.Printf("⚠️  Failed to collect Kafka metrics: %v\n", err)
			}
		}
	}
}

// collectFlinkMetrics periodically collects Flink job and cluster metrics
func (mc *MetricsCollector) collectFlinkMetrics(ctx context.Context) {
	ticker := time.NewTicker(mc.flinkInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := mc.updateFlinkMetrics(); err != nil {
				fmt.Printf("⚠️  Failed to collect Flink metrics: %v\n", err)
			}
		}
	}
}

// updateKafkaMetrics queries Kafka for current metrics
func (mc *MetricsCollector) updateKafkaMetrics() error {
	if len(mc.kafkaAddrs) == 0 {
		mc.kafkaAddrs = []string{"localhost:9092"}
	}
	
	// Create Kafka client
	conn, err := kafka.Dial("tcp", mc.kafkaAddrs[0])
	if err != nil {
		return fmt.Errorf("failed to connect to Kafka: %w", err)
	}
	defer conn.Close()
	
	// Get cluster metadata
	partitions, err := conn.ReadPartitions()
	if err != nil {
		return fmt.Errorf("failed to read partitions: %w", err)
	}
	
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()
	
	// Update broker count
	brokers := make(map[string]bool)
	for _, partition := range partitions {
		for _, replica := range partition.Replicas {
			brokers[fmt.Sprintf("%s:%d", replica.Host, replica.Port)] = true
		}
	}
	mc.kafkaMetrics.BrokerCount = len(brokers)
	
	// Group partitions by topic
	topicPartitions := make(map[string][]kafka.Partition)
	for _, partition := range partitions {
		topicPartitions[partition.Topic] = append(topicPartitions[partition.Topic], partition)
	}
	
	mc.kafkaMetrics.TopicCount = len(topicPartitions)
	
	// Update topic metrics
	var totalMessages, totalBytes int64
	var totalProduceRate, totalConsumeRate float64
	
	for topic, parts := range topicPartitions {
		if mc.kafkaMetrics.Topics[topic] == nil {
			mc.kafkaMetrics.Topics[topic] = &TopicMetrics{Name: topic}
		}
		
		topicMetric := mc.kafkaMetrics.Topics[topic]
		topicMetric.Partitions = len(parts)
		if len(parts) > 0 {
			topicMetric.ReplicationFactor = len(parts[0].Replicas)
		}
		
		// Get topic stats (simplified - in real implementation, you'd query Kafka JMX)
		topicMetric.MessageCount += int64(len(parts)) * 100 // Placeholder
		topicMetric.Size += int64(len(parts)) * 1024 * 100    // Placeholder
		topicMetric.ProduceRate = float64(len(parts)) * 10    // Placeholder
		topicMetric.ConsumeRate = float64(len(parts)) * 9     // Placeholder
		
		totalMessages += topicMetric.MessageCount
		totalBytes += topicMetric.Size
		totalProduceRate += topicMetric.ProduceRate
		totalConsumeRate += topicMetric.ConsumeRate
	}
	
	mc.kafkaMetrics.TotalMessages = totalMessages
	mc.kafkaMetrics.TotalBytes = totalBytes
	mc.kafkaMetrics.MessagesPerSec = totalProduceRate
	mc.kafkaMetrics.BytesPerSec = totalProduceRate * 1024 // Assume 1KB avg message
	mc.kafkaMetrics.ClusterHealth = "HEALTHY" // Simplified
	
	return nil
}

// updateFlinkMetrics queries Flink REST API for current metrics
func (mc *MetricsCollector) updateFlinkMetrics() error {
	if mc.flinkURL == "" {
		mc.flinkURL = "http://localhost:8081"
	}
	
	// Get JobManager overview
	overviewResp, err := http.Get(mc.flinkURL + "/overview")
	if err != nil {
		return fmt.Errorf("failed to get Flink overview: %w", err)
	}
	defer overviewResp.Body.Close()
	
	if overviewResp.StatusCode != http.StatusOK {
		return fmt.Errorf("Flink API returned status %d", overviewResp.StatusCode)
	}
	
	overviewBody, err := ioutil.ReadAll(overviewResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read overview response: %w", err)
	}
	
	var overview struct {
		Flink               string `json:"flink-version"`
		TaskManagers        int    `json:"taskmanagers"`
		SlotsTotal          int    `json:"slots-total"`
		SlotsAvailable      int    `json:"slots-available"`
		JobsRunning         int    `json:"jobs-running"`
		JobsFinished        int    `json:"jobs-finished"`
		JobsCancelled       int    `json:"jobs-cancelled"`
		JobsFailed          int    `json:"jobs-failed"`
	}
	
	if err := json.Unmarshal(overviewBody, &overview); err != nil {
		return fmt.Errorf("failed to parse overview JSON: %w", err)
	}
	
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()
	
	mc.flinkMetrics.JobManagerStatus = "RUNNING"
	mc.flinkMetrics.TaskManagerCount = overview.TaskManagers
	
	// Get jobs list
	jobsResp, err := http.Get(mc.flinkURL + "/jobs")
	if err != nil {
		fmt.Printf("⚠️  Failed to get Flink jobs: %v\n", err)
		return nil // Don't fail completely
	}
	defer jobsResp.Body.Close()
	
	if jobsResp.StatusCode == http.StatusOK {
		jobsBody, err := ioutil.ReadAll(jobsResp.Body)
		if err == nil {
			var jobsResponse struct {
				Jobs []struct {
					ID       string `json:"id"`
					Status   string `json:"status"`
					Name     string `json:"name"`
					StartTime int64 `json:"start-time"`
				} `json:"jobs"`
			}
			
			if json.Unmarshal(jobsBody, &jobsResponse) == nil {
				for _, job := range jobsResponse.Jobs {
					if mc.flinkMetrics.Jobs[job.ID] == nil {
						mc.flinkMetrics.Jobs[job.ID] = &FlinkJob{}
					}
					
					flinkJob := mc.flinkMetrics.Jobs[job.ID]
					flinkJob.ID = job.ID
					flinkJob.Name = job.Name
					flinkJob.Status = job.Status
					flinkJob.StartTime = time.Unix(job.StartTime/1000, 0)
					flinkJob.Duration = time.Since(flinkJob.StartTime)
					
					// Get detailed job metrics
					mc.updateFlinkJobMetrics(job.ID, flinkJob)
				}
			}
		}
	}
	
	// Update cluster metrics (simplified)
	mc.flinkMetrics.ClusterMetrics = &FlinkClusterMetrics{
		CPUUsage:     float64(overview.SlotsTotal-overview.SlotsAvailable) / float64(overview.SlotsTotal) * 100,
		MemoryUsed:   int64(overview.TaskManagers * 512 * 1024 * 1024), // 512MB per TaskManager
		MemoryTotal:  int64(overview.TaskManagers * 1024 * 1024 * 1024), // 1GB per TaskManager
		NetworkIn:    0, // Would need JMX for real metrics
		NetworkOut:   0,
		GCTime:       0,
	}
	
	return nil
}

// updateFlinkJobMetrics gets detailed metrics for a specific job
func (mc *MetricsCollector) updateFlinkJobMetrics(jobID string, job *FlinkJob) {
	// Get job details
	detailsResp, err := http.Get(fmt.Sprintf("%s/jobs/%s", mc.flinkURL, jobID))
	if err != nil {
		return
	}
	defer detailsResp.Body.Close()
	
	if detailsResp.StatusCode != http.StatusOK {
		return
	}
	
	detailsBody, err := ioutil.ReadAll(detailsResp.Body)
	if err != nil {
		return
	}
	
	var jobDetails struct {
		Vertices []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Parallelism int    `json:"parallelism"`
			Status      string `json:"status"`
			Metrics     struct {
				ReadBytes     int64   `json:"read-bytes"`
				WriteBytes    int64   `json:"write-bytes"`
				ReadRecords   int64   `json:"read-records"`
				WriteRecords  int64   `json:"write-records"`
			} `json:"metrics"`
		} `json:"vertices"`
	}
	
	if json.Unmarshal(detailsBody, &jobDetails) == nil {
		var totalRecordsIn, totalRecordsOut int64
		var totalParallelism int
		
		for _, vertex := range jobDetails.Vertices {
			totalRecordsIn += vertex.Metrics.ReadRecords
			totalRecordsOut += vertex.Metrics.WriteRecords
			totalParallelism += vertex.Parallelism
		}
		
		job.RecordsIn = totalRecordsIn
		job.RecordsOut = totalRecordsOut
		job.Parallelism = totalParallelism
		
		if job.Duration.Seconds() > 0 {
			job.RecordsPerSec = float64(totalRecordsOut) / job.Duration.Seconds()
		}
		
		// Simplified metrics
		job.Watermark = time.Now().UnixMilli()
		job.BackPressure = "OK"
	}
}

// GetKafkaMetrics returns current Kafka metrics
func (mc *MetricsCollector) GetKafkaMetrics() *KafkaMetrics {
	mc.metricsLock.RLock()
	defer mc.metricsLock.RUnlock()
	
	// Deep copy to avoid race conditions
	metrics := *mc.kafkaMetrics
	metrics.Topics = make(map[string]*TopicMetrics)
	for k, v := range mc.kafkaMetrics.Topics {
		topicCopy := *v
		metrics.Topics[k] = &topicCopy
	}
	
	return &metrics
}

// GetFlinkMetrics returns current Flink metrics
func (mc *MetricsCollector) GetFlinkMetrics() *FlinkMetrics {
	mc.metricsLock.RLock()
	defer mc.metricsLock.RUnlock()
	
	// Deep copy to avoid race conditions
	metrics := *mc.flinkMetrics
	metrics.Jobs = make(map[string]*FlinkJob)
	for k, v := range mc.flinkMetrics.Jobs {
		jobCopy := *v
		metrics.Jobs[k] = &jobCopy
	}
	
	if mc.flinkMetrics.ClusterMetrics != nil {
		clusterCopy := *mc.flinkMetrics.ClusterMetrics
		metrics.ClusterMetrics = &clusterCopy
	}
	
	return &metrics
}

// GetAllMetrics returns all collected metrics
func (mc *MetricsCollector) GetAllMetrics() map[string]interface{} {
	return map[string]interface{}{
		"kafka": mc.GetKafkaMetrics(),
		"flink": mc.GetFlinkMetrics(),
		"timestamp": time.Now(),
	}
}

// AddError adds an error to the pipeline status
func (mc *MetricsCollector) AddError(component, severity, message, details string, context map[string]interface{}) *PipelineError {
	error := &PipelineError{
		Timestamp: time.Now(),
		Severity:  severity,
		Component: component,
		Message:   message,
		Details:   details,
		Context:   context,
	}
	
	// Add resolution suggestions based on common error patterns
	error.Resolution = mc.suggestResolution(component, message)
	
	return error
}

// suggestResolution provides resolution suggestions for common errors
func (mc *MetricsCollector) suggestResolution(component, message string) string {
	message = strings.ToLower(message)
	
	switch component {
	case "KAFKA":
		if strings.Contains(message, "connection refused") {
			return "Check if Kafka broker is running and accessible. Verify bootstrap servers configuration."
		}
		if strings.Contains(message, "topic does not exist") {
			return "Ensure topics are created before starting the pipeline. Check topic creation permissions."
		}
		if strings.Contains(message, "authentication") {
			return "Verify Kafka authentication credentials (API key/secret) in configuration."
		}
		
	case "FLINK":
		if strings.Contains(message, "connection refused") {
			return "Check if Flink JobManager is running. Verify Flink URL configuration."
		}
		if strings.Contains(message, "job failed") {
			return "Check Flink job logs for detailed error information. Verify SQL syntax and resource availability."
		}
		if strings.Contains(message, "checkpoint") {
			return "Check Flink checkpoint configuration and storage availability."
		}
		
	case "SCHEMA_REGISTRY":
		if strings.Contains(message, "schema not found") {
			return "Ensure AVRO schemas are registered before starting data processing."
		}
		if strings.Contains(message, "compatibility") {
			return "Check schema evolution compatibility settings in Schema Registry."
		}
		
	case "PRODUCER":
		if strings.Contains(message, "serialization") {
			return "Verify AVRO schema format and data structure compatibility."
		}
		if strings.Contains(message, "timeout") {
			return "Increase producer timeout settings or check network connectivity."
		}
		
	case "CONSUMER":
		if strings.Contains(message, "deserialization") {
			return "Verify AVRO schema registration and data format compatibility."
		}
		if strings.Contains(message, "offset") {
			return "Check consumer group settings and offset management configuration."
		}
	}
	
	return "Check component logs and configuration settings for more details."
}

// InitializeSQLStatements initializes SQL statement tracking from loaded statements
func (mc *MetricsCollector) InitializeSQLStatements(statements []*pipeline.SQLStatement, variables map[string]string) {
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()
	
	for _, stmt := range statements {
		flinkStmt := &FlinkStatement{
			ID:               generateStatementID(stmt.Name),
			Name:             stmt.Name,
			Order:            stmt.Order,
			Status:           "PENDING",
			Phase:            "PREPARING",
			Content:          stmt.Content,
			ProcessedContent: "", // Will be set when processed
			FilePath:         stmt.FilePath,
			Variables:        make(map[string]string),
		}
		
		// Copy variables
		for k, v := range variables {
			flinkStmt.Variables[k] = v
		}
		
		mc.flinkMetrics.SQLStatements[stmt.Name] = flinkStmt
	}
}

// UpdateStatementStatus updates the status of a FlinkSQL statement
func (mc *MetricsCollector) UpdateStatementStatus(statementName, status, phase string, deploymentID string, errorMsg string) {
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()
	
	stmt, exists := mc.flinkMetrics.SQLStatements[statementName]
	if !exists {
		return
	}
	
	oldStatus := stmt.Status
	stmt.Status = status
	stmt.Phase = phase
	stmt.DeploymentID = deploymentID
	stmt.ErrorMessage = errorMsg
	
	now := time.Now()
	
	// Handle status transitions
	switch status {
	case "RUNNING":
		if oldStatus == "PENDING" {
			stmt.StartTime = &now
		}
	case "COMPLETED":
		if stmt.StartTime != nil {
			duration := now.Sub(*stmt.StartTime)
			stmt.Duration = &duration
		}
		stmt.CompletionTime = &now
	case "FAILED":
		if stmt.StartTime != nil {
			duration := now.Sub(*stmt.StartTime)
			stmt.Duration = &duration
		}
		stmt.CompletionTime = &now
	}
}

// UpdateStatementMetrics updates runtime metrics for a FlinkSQL statement
func (mc *MetricsCollector) UpdateStatementMetrics(statementName string, recordsProcessed int64, recordsPerSec float64, parallelism int) {
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()
	
	stmt, exists := mc.flinkMetrics.SQLStatements[statementName]
	if !exists {
		return
	}
	
	stmt.RecordsProcessed = recordsProcessed
	stmt.RecordsPerSec = recordsPerSec
	stmt.Parallelism = parallelism
}

// GetSQLStatements returns a copy of current SQL statement metrics
func (mc *MetricsCollector) GetSQLStatements() map[string]*FlinkStatement {
	mc.metricsLock.RLock()
	defer mc.metricsLock.RUnlock()
	
	statements := make(map[string]*FlinkStatement)
	for k, v := range mc.flinkMetrics.SQLStatements {
		// Deep copy to avoid race conditions
		stmt := *v
		statements[k] = &stmt
	}
	
	return statements
}

// SetStatementDependencies sets the dependency chain for SQL statements
func (mc *MetricsCollector) SetStatementDependencies(dependencies map[string][]string) {
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()
	
	for stmtName, deps := range dependencies {
		if stmt, exists := mc.flinkMetrics.SQLStatements[stmtName]; exists {
			stmt.Dependencies = deps
		}
	}
}

// SetFlinkMetrics sets the entire FlinkMetrics object (used for standalone mode)
func (mc *MetricsCollector) SetFlinkMetrics(flinkMetrics *FlinkMetrics) {
	mc.metricsLock.Lock()
	defer mc.metricsLock.Unlock()
	
	mc.flinkMetrics = flinkMetrics
}

// generateStatementID generates a unique ID for a SQL statement
func generateStatementID(name string) string {
	return fmt.Sprintf("stmt-%s-%d", strings.ReplaceAll(strings.ToLower(name), " ", "-"), time.Now().UnixNano()%1000000)
}

package dashboard

import (
	"sync"
	"time"
	
	"pipegen/internal/pipeline"
)

// ExecutionDataCollector gathers metrics during pipeline execution
type ExecutionDataCollector struct {
	executionID     string
	startTime       time.Time
	parameters      ExecutionParameters
	metrics         ExecutionMetrics
	chartData       ChartData
	status          string
	mutex           sync.RWMutex
	dataPoints      []TimeSeriesPoint
	throughputData  []TimeSeriesPoint
	latencyData     []TimeSeriesPoint
	errorData       []TimeSeriesPoint
	lastUpdate      time.Time
}

// NewExecutionDataCollector creates a new data collector for an execution
func NewExecutionDataCollector(executionID string, params ExecutionParameters) *ExecutionDataCollector {
	return &ExecutionDataCollector{
		executionID:    executionID,
		startTime:      time.Now(),
		parameters:     params,
		status:         "running",
		dataPoints:     make([]TimeSeriesPoint, 0),
		throughputData: make([]TimeSeriesPoint, 0),
		latencyData:    make([]TimeSeriesPoint, 0),
		errorData:      make([]TimeSeriesPoint, 0),
		lastUpdate:     time.Now(),
	}
}

// UpdateMetrics updates the current metrics
func (c *ExecutionDataCollector) UpdateMetrics(producerMetrics *pipeline.ProducerStats, consumerMetrics *pipeline.ConsumerStats) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()
	
	// Update basic metrics
	if producerMetrics != nil {
		c.metrics.ProducerMetrics = producerMetrics
		c.metrics.TotalMessages += producerMetrics.MessagesSent
		c.metrics.BytesProcessed += producerMetrics.BytesSent
	}
	
	if consumerMetrics != nil {
		c.metrics.ConsumerMetrics = consumerMetrics
		c.metrics.ErrorCount += consumerMetrics.ErrorCount
	}

	// Calculate messages per second
	duration := now.Sub(c.startTime).Seconds()
	if duration > 0 {
		c.metrics.MessagesPerSecond = float64(c.metrics.TotalMessages) / duration
	}

	// Calculate success rate
	if c.metrics.TotalMessages > 0 {
		c.metrics.SuccessRate = float64(c.metrics.TotalMessages-c.metrics.ErrorCount) / float64(c.metrics.TotalMessages) * 100
	} else {
		c.metrics.SuccessRate = 100.0
	}

	// Add data points for charts
	c.addDataPoint(now, float64(c.metrics.TotalMessages))
	c.addThroughputPoint(now, c.metrics.MessagesPerSecond)
	
	c.lastUpdate = now
}

// UpdateFlinkMetrics updates Flink-specific metrics
func (c *ExecutionDataCollector) UpdateFlinkMetrics(flinkMetrics *FlinkMetrics) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.metrics.FlinkMetrics = flinkMetrics
}

// AddLatencyPoint adds a latency measurement
func (c *ExecutionDataCollector) AddLatencyPoint(latency time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now()
	c.latencyData = append(c.latencyData, TimeSeriesPoint{
		Timestamp: now,
		Value:     float64(latency.Milliseconds()),
	})
	
	// Update average latency (simple moving average)
	if len(c.latencyData) > 0 {
		var sum float64
		for _, point := range c.latencyData {
			sum += point.Value
		}
		c.metrics.AvgLatency = time.Duration(sum/float64(len(c.latencyData))) * time.Millisecond
	}
}

// AddErrorPoint adds an error event
func (c *ExecutionDataCollector) AddErrorPoint(errorType string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now()
	c.metrics.ErrorCount++
	c.errorData = append(c.errorData, TimeSeriesPoint{
		Timestamp: now,
		Value:     1.0,
	})
	
	// Recalculate success rate
	if c.metrics.TotalMessages > 0 {
		c.metrics.SuccessRate = float64(c.metrics.TotalMessages-c.metrics.ErrorCount) / float64(c.metrics.TotalMessages) * 100
	}
}

// SetStatus updates the execution status
func (c *ExecutionDataCollector) SetStatus(status string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.status = status
}

// GetCurrentReport generates a report with current data
func (c *ExecutionDataCollector) GetCurrentReport(pipelineName, pipelineVersion string) *ExecutionReport {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return &ExecutionReport{
		Timestamp:       c.startTime,
		ExecutionID:     c.executionID,
		Parameters:      c.parameters,
		Metrics:         c.metrics,
		Summary:         c.buildSummary(),
		Charts:          c.buildChartData(),
		Status:          c.status,
		Duration:        time.Since(c.startTime),
		PipelineName:    pipelineName,
		PipelineVersion: pipelineVersion,
	}
}

// addDataPoint adds a data point to the messages chart
func (c *ExecutionDataCollector) addDataPoint(timestamp time.Time, value float64) {
	c.dataPoints = append(c.dataPoints, TimeSeriesPoint{
		Timestamp: timestamp,
		Value:     value,
	})
	
	// Keep only last 100 points to avoid memory issues
	if len(c.dataPoints) > 100 {
		c.dataPoints = c.dataPoints[1:]
	}
}

// addThroughputPoint adds a throughput data point
func (c *ExecutionDataCollector) addThroughputPoint(timestamp time.Time, value float64) {
	c.throughputData = append(c.throughputData, TimeSeriesPoint{
		Timestamp: timestamp,
		Value:     value,
	})
	
	// Keep only last 100 points
	if len(c.throughputData) > 100 {
		c.throughputData = c.throughputData[1:]
	}
}

// buildSummary creates execution summary
func (c *ExecutionDataCollector) buildSummary() ExecutionSummary {
	return ExecutionSummary{
		TotalMessagesProcessed: c.metrics.TotalMessages,
		TotalBytesProcessed:   c.metrics.BytesProcessed,
		ThroughputMsgSec:      c.metrics.MessagesPerSecond,
		ErrorRate:             float64(c.metrics.ErrorCount) / float64(c.metrics.TotalMessages) * 100.0,
		SuccessRate:           c.metrics.SuccessRate,
		AverageLatency:        c.metrics.AvgLatency,
	}
}

// buildChartData creates chart data structure
func (c *ExecutionDataCollector) buildChartData() ChartData {
	return ChartData{
		MessagesOverTime:   c.dataPoints,
		ThroughputOverTime: c.throughputData,
		LatencyOverTime:    c.latencyData,
		ErrorRateOverTime:  c.errorData,
	}
}

// StartPeriodicCollection starts collecting metrics at regular intervals
func (c *ExecutionDataCollector) StartPeriodicCollection(producer *pipeline.Producer, consumer *pipeline.Consumer, interval time.Duration) chan struct{} {
	stopChan := make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				var producerStats *pipeline.ProducerStats
				var consumerStats *pipeline.ConsumerStats
				
				if producer != nil {
					producerStats = producer.GetStats()
				}
				
				if consumer != nil {
					consumerStats = consumer.GetStats()
				}
				
				c.UpdateMetrics(producerStats, consumerStats)
				
			case <-stopChan:
				return
			}
		}
	}()
	
	return stopChan
}

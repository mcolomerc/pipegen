package dashboard

import "time"

// TimeSeriesPoint represents a point in time series data for charts
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// ExecutionFlinkMetrics holds Flink-specific metrics for reports
type ExecutionFlinkMetrics struct {
	JobID              string  `json:"job_id"`
	JobStatus          string  `json:"job_status"`
	TaskManagers       int     `json:"task_managers"`
	TaskSlots          int     `json:"task_slots"`
	ProcessedRecords   int64   `json:"processed_records"`
	ProcessedBytes     int64   `json:"processed_bytes"`
	BackPressureStatus string  `json:"back_pressure_status"`
	Checkpoints        int     `json:"checkpoints"`
	RestartCount       int     `json:"restart_count"`
}

// ExecutionReportSummary provides a high-level overview for reports
type ExecutionReportSummary struct {
	Status            string        `json:"status"`
	StartTime         time.Time     `json:"start_time"`
	EndTime           time.Time     `json:"end_time"`
	Duration          time.Duration `json:"duration"`
	TotalMessages     int64         `json:"total_messages"`
	AverageLatency    time.Duration `json:"average_latency"`
	PeakThroughput    float64       `json:"peak_throughput"`
	ErrorRate         float64       `json:"error_rate"`
}

// ExecutionReport represents a complete execution report with all data
type ExecutionReport struct {
	ExecutionID     string            `json:"execution_id"`
	Timestamp       time.Time         `json:"timestamp"`
	Status          string            `json:"status"`
	Duration        time.Duration     `json:"duration"`
	PipelineName    string            `json:"pipeline_name"`
	PipelineVersion string            `json:"pipeline_version"`
	Parameters      map[string]interface{} `json:"parameters"`
	Metrics         map[string]interface{} `json:"metrics"`
	Summary         ExecutionReportSummary  `json:"summary"`
	Charts          map[string][]TimeSeriesPoint `json:"charts"`
}

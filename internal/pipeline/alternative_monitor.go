package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	logpkg "pipegen/internal/log"
)

// Alternative monitoring approaches when Flink REST API metrics are unreliable
type AlternativeMonitor struct {
	config        *Config
	kafkaBrokers  string
	containerName string
	logger        logpkg.Logger
}

func NewAlternativeMonitor(config *Config) *AlternativeMonitor {
	return &AlternativeMonitor{
		config:        config,
		kafkaBrokers:  "broker:29092", // Default internal broker address
		containerName: "broker",       // Default Kafka container name
		logger:        logpkg.Global(),
	}
}

// MonitoringResult contains results from different monitoring approaches
type MonitoringResult struct {
	ConsumerGroupLag   int64
	OutputTopicSize    int64
	OutputTopicRecords int64
	FlinkJobsRunning   bool
	ProcessingDetected bool
	MonitoringMethod   string
	Details            string
}

// WaitForFlinkProcessingAlternative uses multiple monitoring approaches
func (am *AlternativeMonitor) WaitForFlinkProcessingAlternative(ctx context.Context, consumerGroups []string) error {
	if am.logger != nil {
		am.logger.Info("using enhanced monitoring")
	}

	maxAttempts := 15
	checkInterval := 3 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if am.logger != nil {
			am.logger.Info("monitoring check", "attempt", attempt, "max", maxAttempts)
		}

		result, err := am.checkProcessingActivity(consumerGroups)
		if err != nil {
			if am.logger != nil {
				am.logger.Warn("monitoring error", "err", err)
			}
		} else if result.ProcessingDetected {
			if am.logger != nil {
				am.logger.Info("processing detected", "details", result.Details)
			}
			return nil
		} else {
			if am.logger != nil {
				am.logger.Debug("no processing yet", "details", result.Details)
			}
		}

		// Wait before next attempt
		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(checkInterval):
				// Continue to next attempt
			}
		}
	}

	if am.logger != nil {
		am.logger.Warn("no processing detected after attempts", "attempts", maxAttempts)
	}
	return nil // Don't fail - processing might still be working
}

// checkProcessingActivity tries multiple monitoring approaches
func (am *AlternativeMonitor) checkProcessingActivity(consumerGroups []string) (*MonitoringResult, error) {
	result := &MonitoringResult{}

	// Approach 1: Check Kafka consumer group lag
	if len(consumerGroups) > 0 {
		lag, err := am.checkConsumerGroupLag(consumerGroups[0])
		if err == nil {
			result.ConsumerGroupLag = lag
			if lag == 0 { // No lag means records have been consumed
				result.ProcessingDetected = true
				result.MonitoringMethod = "Consumer Group Lag"
				result.Details = fmt.Sprintf("Flink has processed all input data (consumer group lag: %d)", lag)
				return result, nil
			}
		}
	}

	// Approach 2: Check output topic growth
	outputSize, outputRecords, err := am.checkOutputTopicGrowth()
	if err == nil {
		result.OutputTopicSize = outputSize
		result.OutputTopicRecords = outputRecords
		if outputSize > 0 || outputRecords > 0 {
			result.ProcessingDetected = true
			result.MonitoringMethod = "Output Topic Growth"
			result.Details = fmt.Sprintf("Output topic has %d bytes, ~%d records", outputSize, outputRecords)
			return result, nil
		}
	}

	// Approach 3: Check if Flink jobs are running
	running, err := am.checkFlinkJobsRunning()
	if err == nil {
		result.FlinkJobsRunning = running
		if running {
			result.Details = fmt.Sprintf("Flink jobs running, consumer group lag: %d, output size: %d bytes",
				result.ConsumerGroupLag, result.OutputTopicSize)
		} else {
			result.Details = "No Flink jobs running"
		}
	}

	return result, nil
}

// checkConsumerGroupLag checks Kafka consumer group lag
func (am *AlternativeMonitor) checkConsumerGroupLag(consumerGroup string) (int64, error) {
	cmd := exec.Command("docker", "exec", am.containerName,
		"/opt/kafka/bin/kafka-consumer-groups.sh",
		"--bootstrap-server", am.kafkaBrokers,
		"--describe", "--group", consumerGroup)

	output, err := cmd.Output()
	if err != nil {
		return -1, fmt.Errorf("failed to check consumer group: %w", err)
	}

	// Parse the output to extract lag
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, consumerGroup) && strings.Contains(line, "transactions") {
			fields := strings.Fields(line)
			if len(fields) >= 6 {
				lag, err := strconv.ParseInt(fields[5], 10, 64)
				if err == nil {
					return lag, nil
				}
			}
		}
	}

	return -1, fmt.Errorf("could not parse consumer group lag")
}

// checkOutputTopicGrowth checks if the output topic is growing
func (am *AlternativeMonitor) checkOutputTopicGrowth() (int64, int64, error) {
	cmd := exec.Command("docker", "exec", am.containerName,
		"/opt/kafka/bin/kafka-log-dirs.sh",
		"--bootstrap-server", am.kafkaBrokers,
		"--topic-list", "output-results",
		"--describe")

	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to check output topic: %w", err)
	}

	// Parse JSON output to get partition size
	var logDirs struct {
		Brokers []struct {
			LogDirs []struct {
				Partitions []struct {
					Partition string `json:"partition"`
					Size      int64  `json:"size"`
				} `json:"partitions"`
			} `json:"logDirs"`
		} `json:"brokers"`
	}

	if err := json.Unmarshal(output, &logDirs); err != nil {
		return 0, 0, fmt.Errorf("failed to parse log dirs output: %w", err)
	}

	for _, broker := range logDirs.Brokers {
		for _, logDir := range broker.LogDirs {
			for _, partition := range logDir.Partitions {
				if partition.Partition == "output-results-0" {
					// Estimate records (assuming ~100 bytes per record)
					estimatedRecords := partition.Size / 100
					return partition.Size, estimatedRecords, nil
				}
			}
		}
	}

	return 0, 0, fmt.Errorf("output topic partition not found")
}

// checkFlinkJobsRunning checks if any Flink jobs are currently running
func (am *AlternativeMonitor) checkFlinkJobsRunning() (bool, error) {
	resp, err := http.Get(fmt.Sprintf("%s/jobs", am.config.FlinkURL))
	if err != nil {
		return false, fmt.Errorf("failed to fetch Flink jobs: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			if am.logger != nil {
				am.logger.Warn("failed close response body", "err", err)
			}
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read Flink jobs response: %w", err)
	}

	var jobsResp struct {
		Jobs []struct {
			Status string `json:"status"`
		} `json:"jobs"`
	}

	if err := json.Unmarshal(body, &jobsResp); err != nil {
		return false, fmt.Errorf("failed to parse Flink jobs response: %w", err)
	}

	for _, job := range jobsResp.Jobs {
		if job.Status == "RUNNING" {
			return true, nil
		}
	}

	return false, nil
}

// GetConsumerGroupFromTableName extracts consumer group name from Flink table
func (am *AlternativeMonitor) GetConsumerGroupFromTableName(tableName string) string {
	// Pattern: table name -> consumer group (e.g., transactions_v4 -> flink_table_transactions_v4)
	return fmt.Sprintf("flink_table_%s", tableName)
}

// CheckKafkaTopicHasRecords verifies that a topic has data
func (am *AlternativeMonitor) CheckKafkaTopicHasRecords(topicName string) (bool, int64, error) {
	cmd := exec.Command("docker", "exec", am.containerName,
		"/opt/kafka/bin/kafka-log-dirs.sh",
		"--bootstrap-server", am.kafkaBrokers,
		"--topic-list", topicName,
		"--describe")

	output, err := cmd.Output()
	if err != nil {
		return false, 0, fmt.Errorf("failed to check topic: %w", err)
	}

	// Use regex to extract size from the JSON output
	sizeRegex := regexp.MustCompile(`"size":(\d+)`)
	matches := sizeRegex.FindStringSubmatch(string(output))
	if len(matches) >= 2 {
		size, err := strconv.ParseInt(matches[1], 10, 64)
		if err == nil && size > 0 {
			return true, size, nil
		}
	}

	return false, 0, nil
}

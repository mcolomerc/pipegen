package pipeline

import (
	"testing"
)

func TestAlternativeMonitor_GetConsumerGroupFromTableName(t *testing.T) {
	// Create a mock config
	config := &Config{}

	monitor := NewAlternativeMonitor(config)

	// Test consumer group extraction
	consumerGroup := monitor.GetConsumerGroupFromTableName("transactions_v4")
	expected := "flink_table_transactions_v4"
	if consumerGroup != expected {
		t.Errorf("Expected consumer group %s, got %s", expected, consumerGroup)
	}
}

func TestAlternativeMonitor_MonitoringResult(t *testing.T) {
	// Test monitoring result structure
	result := &MonitoringResult{
		ConsumerGroupLag:   0,
		OutputTopicSize:    12345,
		ProcessingDetected: true,
		MonitoringMethod:   "Output Topic Growth",
		Details:            "Test details",
	}

	if !result.ProcessingDetected {
		t.Error("Expected processing to be detected")
	}

	if result.MonitoringMethod != "Output Topic Growth" {
		t.Errorf("Expected monitoring method 'Output Topic Growth', got %s", result.MonitoringMethod)
	}

	if result.OutputTopicSize != 12345 {
		t.Errorf("Expected output topic size 12345, got %d", result.OutputTopicSize)
	}
}

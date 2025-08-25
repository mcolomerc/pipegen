package pipeline

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// TrafficPattern defines a traffic spike during pipeline execution
type TrafficPattern struct {
	StartTime      time.Duration
	EndTime        time.Duration
	RateMultiplier float64 // Percentage as multiplier (300% = 3.0)
}

// TrafficPatterns holds all traffic patterns for an execution
type TrafficPatterns struct {
	BaseRate int // Base messages per second
	Patterns []TrafficPattern
}

// ParseTrafficPattern parses a traffic pattern string into TrafficPatterns
// Format: "start-end:rate%,start-end:rate%"
// Example: "30s-60s:300%,90s-120s:200%"
func ParseTrafficPattern(patternStr string, baseRate int) (*TrafficPatterns, error) {
	if patternStr == "" {
		return &TrafficPatterns{
			BaseRate: baseRate,
			Patterns: []TrafficPattern{},
		}, nil
	}

	patterns := []TrafficPattern{}
	parts := strings.Split(patternStr, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Split by colon: "30s-60s:300%"
		colonParts := strings.Split(part, ":")
		if len(colonParts) != 2 {
			return nil, fmt.Errorf("invalid pattern format '%s': expected 'start-end:rate%%'", part)
		}

		timeRange := strings.TrimSpace(colonParts[0])
		rateStr := strings.TrimSpace(colonParts[1])

		// Parse time range: "30s-60s"
		dashParts := strings.Split(timeRange, "-")
		if len(dashParts) != 2 {
			return nil, fmt.Errorf("invalid time range '%s': expected 'start-end'", timeRange)
		}

		startTime, err := time.ParseDuration(strings.TrimSpace(dashParts[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid start time '%s': %w", dashParts[0], err)
		}

		endTime, err := time.ParseDuration(strings.TrimSpace(dashParts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid end time '%s': %w", dashParts[1], err)
		}

		if endTime <= startTime {
			return nil, fmt.Errorf("end time '%s' must be after start time '%s'", dashParts[1], dashParts[0])
		}

		// Parse rate: "300%"
		if !strings.HasSuffix(rateStr, "%") {
			return nil, fmt.Errorf("invalid rate format '%s': expected percentage (e.g., '300%%')", rateStr)
		}

		rateValue, err := strconv.ParseFloat(strings.TrimSuffix(rateStr, "%"), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid rate value '%s': %w", rateStr, err)
		}

		if rateValue <= 0 {
			return nil, fmt.Errorf("rate value must be positive, got '%s'", rateStr)
		}

		patterns = append(patterns, TrafficPattern{
			StartTime:      startTime,
			EndTime:        endTime,
			RateMultiplier: rateValue / 100.0, // Convert percentage to multiplier
		})
	}

	// Validate patterns don't overlap and are in chronological order
	if err := validatePatterns(patterns); err != nil {
		return nil, err
	}

	return &TrafficPatterns{
		BaseRate: baseRate,
		Patterns: patterns,
	}, nil
}

// validatePatterns checks for overlapping patterns and ensures chronological order
func validatePatterns(patterns []TrafficPattern) error {
	for i := 0; i < len(patterns)-1; i++ {
		current := patterns[i]
		next := patterns[i+1]

		// Check if patterns overlap
		if current.EndTime > next.StartTime {
			return fmt.Errorf("traffic patterns overlap: pattern ending at %v conflicts with pattern starting at %v",
				current.EndTime, next.StartTime)
		}
	}
	return nil
}

// GetRateAt returns the message rate that should be used at a specific time
func (tp *TrafficPatterns) GetRateAt(elapsed time.Duration) int {
	// Check if we're in any pattern period
	for _, pattern := range tp.Patterns {
		if elapsed >= pattern.StartTime && elapsed < pattern.EndTime {
			return int(float64(tp.BaseRate) * pattern.RateMultiplier)
		}
	}

	// Not in any pattern, use base rate
	return tp.BaseRate
}

// GetPatternSummary returns a human-readable summary of the traffic patterns
func (tp *TrafficPatterns) GetPatternSummary() string {
	if len(tp.Patterns) == 0 {
		return fmt.Sprintf("Constant rate: %d msg/sec", tp.BaseRate)
	}

	summary := fmt.Sprintf("Base rate: %d msg/sec", tp.BaseRate)
	for i, pattern := range tp.Patterns {
		peakRate := int(float64(tp.BaseRate) * pattern.RateMultiplier)
		summary += fmt.Sprintf("\n  Peak %d: %v-%v at %d msg/sec (%.0f%%)",
			i+1, pattern.StartTime, pattern.EndTime, peakRate, pattern.RateMultiplier*100)
	}
	return summary
}

// HasPatterns returns true if any traffic patterns are defined
func (tp *TrafficPatterns) HasPatterns() bool {
	return len(tp.Patterns) > 0
}

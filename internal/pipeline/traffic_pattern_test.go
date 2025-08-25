package pipeline

import (
	"testing"
	"time"
)

func TestParseTrafficPattern(t *testing.T) {
	tests := []struct {
		name         string
		pattern      string
		baseRate     int
		wantErr      bool
		wantPatterns int
	}{
		{
			name:         "empty pattern",
			pattern:      "",
			baseRate:     100,
			wantErr:      false,
			wantPatterns: 0,
		},
		{
			name:         "single pattern",
			pattern:      "30s-60s:300%",
			baseRate:     100,
			wantErr:      false,
			wantPatterns: 1,
		},
		{
			name:         "multiple patterns",
			pattern:      "30s-60s:300%,90s-120s:200%",
			baseRate:     100,
			wantErr:      false,
			wantPatterns: 2,
		},
		{
			name:     "invalid format - missing percentage",
			pattern:  "30s-60s:300",
			baseRate: 100,
			wantErr:  true,
		},
		{
			name:     "invalid format - missing colon",
			pattern:  "30s-60s",
			baseRate: 100,
			wantErr:  true,
		},
		{
			name:     "invalid time format",
			pattern:  "invalid-60s:300%",
			baseRate: 100,
			wantErr:  true,
		},
		{
			name:     "overlapping patterns",
			pattern:  "30s-60s:300%,45s-90s:200%",
			baseRate: 100,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns, err := ParseTrafficPattern(tt.pattern, tt.baseRate)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if len(patterns.Patterns) != tt.wantPatterns {
				t.Errorf("expected %d patterns, got %d", tt.wantPatterns, len(patterns.Patterns))
			}
		})
	}
}

func TestTrafficPatterns_GetRateAt(t *testing.T) {
	patterns, err := ParseTrafficPattern("30s-60s:300%,90s-120s:200%", 100)
	if err != nil {
		t.Fatalf("failed to parse pattern: %v", err)
	}

	tests := []struct {
		name     string
		elapsed  time.Duration
		wantRate int
	}{
		{
			name:     "before first pattern",
			elapsed:  15 * time.Second,
			wantRate: 100, // base rate
		},
		{
			name:     "during first pattern",
			elapsed:  45 * time.Second,
			wantRate: 300, // 3x base rate
		},
		{
			name:     "between patterns",
			elapsed:  75 * time.Second,
			wantRate: 100, // base rate
		},
		{
			name:     "during second pattern",
			elapsed:  105 * time.Second,
			wantRate: 200, // 2x base rate
		},
		{
			name:     "after all patterns",
			elapsed:  150 * time.Second,
			wantRate: 100, // base rate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := patterns.GetRateAt(tt.elapsed)
			if got != tt.wantRate {
				t.Errorf("GetRateAt(%v) = %d, want %d", tt.elapsed, got, tt.wantRate)
			}
		})
	}
}

func TestTrafficPatterns_HasPatterns(t *testing.T) {
	// Test with patterns
	patterns, err := ParseTrafficPattern("30s-60s:300%", 100)
	if err != nil {
		t.Fatalf("failed to parse pattern: %v", err)
	}
	if !patterns.HasPatterns() {
		t.Error("expected HasPatterns() to return true")
	}

	// Test without patterns
	emptyPatterns, err := ParseTrafficPattern("", 100)
	if err != nil {
		t.Fatalf("failed to parse empty pattern: %v", err)
	}
	if emptyPatterns.HasPatterns() {
		t.Error("expected HasPatterns() to return false for empty patterns")
	}
}

func TestTrafficPatterns_GetPatternSummary(t *testing.T) {
	// Test with patterns
	patterns, err := ParseTrafficPattern("30s-60s:300%,90s-120s:200%", 100)
	if err != nil {
		t.Fatalf("failed to parse pattern: %v", err)
	}

	summary := patterns.GetPatternSummary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}

	// Test without patterns
	emptyPatterns, err := ParseTrafficPattern("", 100)
	if err != nil {
		t.Fatalf("failed to parse empty pattern: %v", err)
	}

	emptySummary := emptyPatterns.GetPatternSummary()
	if emptySummary == "" {
		t.Error("expected non-empty summary even for constant rate")
	}
}

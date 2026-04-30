package api

import (
	"testing"
	"time"
)

func TestParseCustomDuration(t *testing.T) {
	scenarios := []struct {
		name        string
		input       string
		expected    time.Duration
		expectError bool
	}{
		{name: "1h", input: "1h", expected: 1 * time.Hour},
		{name: "24h", input: "24h", expected: 24 * time.Hour},
		{name: "1d", input: "1d", expected: 24 * time.Hour},
		{name: "7d", input: "7d", expected: 7 * 24 * time.Hour},
		{name: "14d", input: "14d", expected: 14 * 24 * time.Hour},
		{name: "30d", input: "30d", expected: 30 * 24 * time.Hour},
		{name: "60d", input: "60d", expected: 60 * 24 * time.Hour},
		{name: "90d", input: "90d", expected: 90 * 24 * time.Hour},
		{name: "2h", input: "2h", expected: 2 * time.Hour},
		{name: "48h", input: "48h", expected: 48 * time.Hour},
		// Error cases
		{name: "empty-string", input: "", expectError: true},
		{name: "no-unit", input: "30", expectError: true},
		{name: "invalid-unit", input: "30m", expectError: true},
		{name: "invalid-unit-s", input: "30s", expectError: true},
		{name: "zero-value", input: "0d", expectError: true},
		{name: "negative", input: "-1d", expectError: true},
		{name: "letters", input: "abc", expectError: true},
		{name: "mixed", input: "1d2h", expectError: true},
		{name: "over-90d", input: "91d", expectError: true},
		{name: "over-90d-in-hours", input: "2161h", expectError: true},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			result, err := ParseCustomDuration(scenario.input)
			if scenario.expectError {
				if err == nil {
					t.Errorf("expected error for input %s, but got none", scenario.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for input %s: %v", scenario.input, err)
				return
			}
			if result != scenario.expected {
				t.Errorf("expected %v for input %s, got %v", scenario.expected, scenario.input, result)
			}
		})
	}
}

func TestCalculateLabelWidth(t *testing.T) {
	scenarios := []struct {
		name      string
		duration  string
		baseLabel string
		minWidth  int
	}{
		{name: "uptime-1h", duration: "1h", baseLabel: "uptime", minWidth: 65},
		{name: "uptime-30d", duration: "30d", baseLabel: "uptime", minWidth: 65},
		{name: "uptime-90d", duration: "90d", baseLabel: "uptime", minWidth: 65},
		{name: "response-time-1h", duration: "1h", baseLabel: "response time", minWidth: 65},
		{name: "response-time-30d", duration: "30d", baseLabel: "response time", minWidth: 65},
		{name: "response-time-90d", duration: "90d", baseLabel: "response time", minWidth: 65},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			result := CalculateLabelWidth(scenario.duration, scenario.baseLabel)
			if result < scenario.minWidth {
				t.Errorf("expected label width >= %d for %s/%s, got %d", scenario.minWidth, scenario.baseLabel, scenario.duration, result)
			}
			if result <= 0 {
				t.Errorf("label width must be positive for %s/%s, got %d", scenario.baseLabel, scenario.duration, result)
			}
		})
	}
}

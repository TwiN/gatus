package endpoint

import (
	"testing"
	"time"
)

func TestNewEndpointStatus(t *testing.T) {
	ep := &Endpoint{Name: "name", Group: "group"}
	status := NewStatus(ep.Group, ep.Name)
	if status.Name != ep.Name {
		t.Errorf("expected %s, got %s", ep.Name, status.Name)
	}
	if status.Group != ep.Group {
		t.Errorf("expected %s, got %s", ep.Group, status.Group)
	}
	if status.Key != "group_name" {
		t.Errorf("expected %s, got %s", "group_name", status.Key)
	}
}

func TestStatusSetPeriod(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{name: "zero", duration: 0, expected: ""},
		{name: "1h", duration: 1 * time.Hour, expected: "1h"},
		{name: "2h", duration: 2 * time.Hour, expected: "2h"},
		{name: "24h", duration: 24 * time.Hour, expected: "1d"},
		{name: "48h", duration: 48 * time.Hour, expected: "2d"},
		{name: "7d", duration: 7 * 24 * time.Hour, expected: "7d"},
		{name: "14d", duration: 14 * 24 * time.Hour, expected: "14d"},
		{name: "30d", duration: 30 * 24 * time.Hour, expected: "30d"},
		{name: "60d", duration: 60 * 24 * time.Hour, expected: "60d"},
		{name: "90d", duration: 90 * 24 * time.Hour, expected: "90d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := NewStatus("group", "name")
			status.SetPeriod(tt.duration)
			if status.Period != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, status.Period)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		value    int
		unit     string
		expected string
	}{
		{value: 0, unit: "h", expected: "0h"},
		{value: 1, unit: "h", expected: "1h"},
		{value: 24, unit: "h", expected: "24h"},
		{value: 1, unit: "d", expected: "1d"},
		{value: 7, unit: "d", expected: "7d"},
		{value: 30, unit: "d", expected: "30d"},
		{value: 90, unit: "d", expected: "90d"},
		{value: 100, unit: "d", expected: "100d"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDurationString(tt.value, tt.unit)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestNormalizeUptime(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{name: "zero", input: 0, expected: 0},
		{name: "one", input: 1, expected: 1},
		{name: "0.995", input: 0.995, expected: 0.995},
		{name: "0.9999", input: 0.9999, expected: 0.9999},
		{name: "0.12345-rounds-down", input: 0.12345, expected: 0.1235},
		{name: "0.12344-rounds-down", input: 0.12344, expected: 0.1234},
		{name: "clamp-above-1", input: 1.5, expected: 1},
		{name: "clamp-below-0", input: -0.1, expected: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeUptime(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeUptime(%f) = %f, want %f", tt.input, result, tt.expected)
			}
		})
	}
}

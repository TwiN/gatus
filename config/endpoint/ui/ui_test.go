package ui

import (
	"errors"
	"testing"
	"time"
)

func TestValidateAndSetDefaults(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name: "with-valid-config",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
			},
			wantErr: nil,
		},
		{
			name: "with-invalid-threshold-length",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500}},
				},
			},
			wantErr: ErrInvalidBadgeResponseTimeConfig,
		},
		{
			name: "with-invalid-thresholds-order",
			config: &Config{
				Badge: &Badge{ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 500, 300, 750}}},
			},
			wantErr: ErrInvalidBadgeResponseTimeConfig,
		},
		{
			name:    "with-no-badge-configured", // should give default badge cfg
			config:  &Config{},
			wantErr: nil,
		},
		{
			name: "with-valid-period-1h",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
				Period: CustomDuration(1 * time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "with-valid-period-30d",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
				Period: CustomDuration(30 * 24 * time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "with-valid-period-90d",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
				Period: CustomDuration(90 * 24 * time.Hour),
			},
			wantErr: nil,
		},
		{
			name: "with-invalid-period-too-small",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
				Period: CustomDuration(30 * time.Minute),
			},
			wantErr: ErrInvalidPeriod,
		},
		{
			name: "with-invalid-period-too-large",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
				Period: CustomDuration(91 * 24 * time.Hour),
			},
			wantErr: ErrInvalidPeriod,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.ValidateAndSetDefaults(); !errors.Is(err, tt.wantErr) {
				t.Errorf("Expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestPeriodDurationString(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name:     "no-period",
			config:   &Config{},
			expected: "",
		},
		{
			name:     "1-hour",
			config:   &Config{Period: CustomDuration(1 * time.Hour)},
			expected: "1h",
		},
		{
			name:     "24-hours",
			config:   &Config{Period: CustomDuration(24 * time.Hour)},
			expected: "1d",
		},
		{
			name:     "7-days",
			config:   &Config{Period: CustomDuration(7 * 24 * time.Hour)},
			expected: "7d",
		},
		{
			name:     "30-days",
			config:   &Config{Period: CustomDuration(30 * 24 * time.Hour)},
			expected: "30d",
		},
		{
			name:     "90-days",
			config:   &Config{Period: CustomDuration(90 * 24 * time.Hour)},
			expected: "90d",
		},
		{
			name:     "48-hours",
			config:   &Config{Period: CustomDuration(48 * time.Hour)},
			expected: "2d",
		},
		{
			name:     "2-hours",
			config:   &Config{Period: CustomDuration(2 * time.Hour)},
			expected: "2h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.PeriodDurationString()
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseHumanDuration(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    time.Duration
		expectError bool
	}{
		{name: "1d", input: "1d", expected: 24 * time.Hour},
		{name: "7d", input: "7d", expected: 7 * 24 * time.Hour},
		{name: "30d", input: "30d", expected: 30 * 24 * time.Hour},
		{name: "90d", input: "90d", expected: 90 * 24 * time.Hour},
		{name: "1h", input: "1h", expected: 1 * time.Hour},
		{name: "24h", input: "24h", expected: 24 * time.Hour},
		{name: "1h30m", input: "1h30m", expected: 90 * time.Minute},
		{name: "5m", input: "5m", expected: 5 * time.Minute},
		{name: "empty", input: "", expectError: true},
		{name: "invalid", input: "abc", expectError: true},
		{name: "zero-d", input: "0d", expectError: true},
		{name: "negative-d", input: "-1d", expectError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseHumanDuration(tt.input)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for input %q, but got none", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("expected %v for input %q, got %v", tt.expected, tt.input, result)
			}
		})
	}
}

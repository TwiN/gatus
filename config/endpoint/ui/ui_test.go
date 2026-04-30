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
				Period: 1 * time.Hour,
			},
			wantErr: nil,
		},
		{
			name: "with-valid-period-30d",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
				Period: 30 * 24 * time.Hour,
			},
			wantErr: nil,
		},
		{
			name: "with-valid-period-90d",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
				Period: 90 * 24 * time.Hour,
			},
			wantErr: nil,
		},
		{
			name: "with-invalid-period-too-small",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
				Period: 30 * time.Minute,
			},
			wantErr: ErrInvalidPeriod,
		},
		{
			name: "with-invalid-period-too-large",
			config: &Config{
				Badge: &Badge{
					ResponseTime: &ResponseTime{Thresholds: []int{50, 200, 300, 500, 750}},
				},
				Period: 91 * 24 * time.Hour,
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
			config:   &Config{Period: 1 * time.Hour},
			expected: "1h",
		},
		{
			name:     "24-hours",
			config:   &Config{Period: 24 * time.Hour},
			expected: "1d",
		},
		{
			name:     "7-days",
			config:   &Config{Period: 7 * 24 * time.Hour},
			expected: "7d",
		},
		{
			name:     "30-days",
			config:   &Config{Period: 30 * 24 * time.Hour},
			expected: "30d",
		},
		{
			name:     "90-days",
			config:   &Config{Period: 90 * 24 * time.Hour},
			expected: "90d",
		},
		{
			name:     "48-hours",
			config:   &Config{Period: 48 * time.Hour},
			expected: "2d",
		},
		{
			name:     "2-hours",
			config:   &Config{Period: 2 * time.Hour},
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

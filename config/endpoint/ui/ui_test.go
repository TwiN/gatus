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
			name:    "with-valid-dashboard-history",
			config:  &Config{DashboardHistory: 24 * time.Hour},
			wantErr: nil,
		},
		{
			name:    "with-dashboard-history-too-low",
			config:  &Config{DashboardHistory: 30 * time.Second},
			wantErr: ErrInvalidDashboardHistory,
		},
		{
			name:    "with-negative-dashboard-history",
			config:  &Config{DashboardHistory: -time.Hour},
			wantErr: ErrInvalidDashboardHistory,
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

package ui

import (
	"errors"
	"testing"
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
			name: "with-valid-links",
			config: &Config{
				Links: []Link{
					{Name: "Repository", Value: "https://github.com/example/example"},
					{Name: "Staging", Value: "https://staging.example.org"},
				},
			},
			wantErr: nil,
		},
		{
			name: "with-link-missing-name",
			config: &Config{
				Links: []Link{{Value: "https://example.org"}},
			},
			wantErr: ErrInvalidLinkConfig,
		},
		{
			name: "with-link-missing-value",
			config: &Config{
				Links: []Link{{Name: "Example"}},
			},
			wantErr: ErrInvalidLinkConfig,
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

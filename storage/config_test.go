package storage

import (
	"testing"
	"time"
)

func TestConfig_ValidateAndSetDefaults_UptimeRetention(t *testing.T) {
	scenarios := []struct {
		name              string
		input             time.Duration
		expectedRetention time.Duration
	}{
		{
			name:              "unset-defaults-to-30d",
			input:             0,
			expectedRetention: DefaultUptimeRetention,
		},
		{
			name:              "negative-defaults-to-30d",
			input:             -1,
			expectedRetention: DefaultUptimeRetention,
		},
		{
			name:              "custom-365d-is-kept",
			input:             365 * 24 * time.Hour,
			expectedRetention: 365 * 24 * time.Hour,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			cfg := &Config{Type: TypeMemory, UptimeRetention: scenario.input}
			if err := cfg.ValidateAndSetDefaults(); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if cfg.UptimeRetention != scenario.expectedRetention {
				t.Errorf("expected UptimeRetention to be %s, got %s", scenario.expectedRetention, cfg.UptimeRetention)
			}
		})
	}
}

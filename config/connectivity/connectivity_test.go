package connectivity

import (
	"fmt"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	scenarios := []struct {
		name             string
		cfg              *Config
		expectedErr      error
		expectedInterval time.Duration
	}{
		{
			name:             "good-config",
			cfg:              &Config{Checker: &Checker{Target: "1.1.1.1:53", Interval: 10 * time.Second}},
			expectedInterval: 10 * time.Second,
		},
		{
			name:             "good-config-with-default-interval",
			cfg:              &Config{Checker: &Checker{Target: "8.8.8.8:53", Interval: 0}},
			expectedInterval: 60 * time.Second,
		},
		{
			name:        "config-with-interval-too-low",
			cfg:         &Config{Checker: &Checker{Target: "1.1.1.1:53", Interval: 4 * time.Second}},
			expectedErr: ErrInvalidInterval,
		},
		{
			name:        "config-with-invalid-target-due-to-missing-port",
			cfg:         &Config{Checker: &Checker{Target: "1.1.1.1", Interval: 15 * time.Second}},
			expectedErr: ErrInvalidDNSTarget,
		},
		{
			name:        "config-with-invalid-target-due-to-invalid-dns-port",
			cfg:         &Config{Checker: &Checker{Target: "1.1.1.1:52", Interval: 15 * time.Second}},
			expectedErr: ErrInvalidDNSTarget,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.cfg.ValidateAndSetDefaults()
			if fmt.Sprintf("%s", err) != fmt.Sprintf("%s", scenario.expectedErr) {
				t.Errorf("expected error %v, got %v", scenario.expectedErr, err)
			}
			if err == nil && scenario.expectedErr == nil {
				if scenario.cfg.Checker.Interval != scenario.expectedInterval {
					t.Errorf("expected interval %v, got %v", scenario.expectedInterval, scenario.cfg.Checker.Interval)
				}
			}
		})
	}
}

func TestChecker_IsConnected(t *testing.T) {
	checker := &Checker{Target: "1.1.1.1:53", Interval: 10 * time.Second}
	if !checker.IsConnected() {
		t.Error("expected checker.IsConnected() to be true")
	}
}

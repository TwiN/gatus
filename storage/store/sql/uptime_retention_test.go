package sql

import (
	"testing"
	"time"
)

func TestSetUptimeRetention(t *testing.T) {
	// Restore the default after the test so other tests are unaffected
	defer SetUptimeRetention(defaultUptimeRetention)

	SetUptimeRetention(365 * 24 * time.Hour)
	if uptimeRetention != 365*24*time.Hour {
		t.Errorf("expected uptimeRetention to be 365d, got %s", uptimeRetention)
	}
	if uptimeAgeCleanUpThreshold != 365*24*time.Hour+uptimeCleanUpBuffer {
		t.Errorf("expected uptimeAgeCleanUpThreshold to be retention+buffer, got %s", uptimeAgeCleanUpThreshold)
	}

	// A value <= 0 must reset to the default
	SetUptimeRetention(0)
	if uptimeRetention != defaultUptimeRetention {
		t.Errorf("expected uptimeRetention to reset to default, got %s", uptimeRetention)
	}
}

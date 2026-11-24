package security

import "testing"

func TestConfig_ProtectStatusesEnabled(t *testing.T) {
	t.Parallel()
	if (&Config{}).ProtectStatusesEnabled() != true {
		t.Error("expected default protect statuses enabled")
	}
	disabled := false
	if (&Config{ProtectStatuses: &disabled}).ProtectStatusesEnabled() {
		t.Error("expected protect statuses disabled")
	}
	enabled := true
	if !(&Config{ProtectStatuses: &enabled}).ProtectStatusesEnabled() {
		t.Error("expected protect statuses enabled")
	}
}
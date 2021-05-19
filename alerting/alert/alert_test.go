package alert

import "testing"

func TestAlert_IsEnabled(t *testing.T) {
	if (Alert{Enabled: nil}).IsEnabled() {
		t.Error("alert.IsEnabled() should've returned false, because Enabled was set to nil")
	}
	if value := false; (Alert{Enabled: &value}).IsEnabled() {
		t.Error("alert.IsEnabled() should've returned false, because Enabled was set to false")
	}
	if value := true; !(Alert{Enabled: &value}).IsEnabled() {
		t.Error("alert.IsEnabled() should've returned true, because Enabled was set to true")
	}
}

func TestAlert_GetDescription(t *testing.T) {
	if (Alert{Description: nil}).GetDescription() != "" {
		t.Error("alert.GetDescription() should've returned an empty string, because Description was set to nil")
	}
	if value := "description"; (Alert{Description: &value}).GetDescription() != value {
		t.Error("alert.GetDescription() should've returned false, because Description was set to 'description'")
	}
}

func TestAlert_IsSendingOnResolved(t *testing.T) {
	if (Alert{SendOnResolved: nil}).IsSendingOnResolved() {
		t.Error("alert.IsSendingOnResolved() should've returned false, because SendOnResolved was set to nil")
	}
	if value := false; (Alert{SendOnResolved: &value}).IsSendingOnResolved() {
		t.Error("alert.IsSendingOnResolved() should've returned false, because SendOnResolved was set to false")
	}
	if value := true; !(Alert{SendOnResolved: &value}).IsSendingOnResolved() {
		t.Error("alert.IsSendingOnResolved() should've returned true, because SendOnResolved was set to true")
	}
}

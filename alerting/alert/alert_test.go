package alert

import (
	"testing"
)

func TestAlert_ValidateAndSetDefaults(t *testing.T) {
	invalidDescription := "\""
	scenarios := []struct {
		name                     string
		alert                    Alert
		expectedError            error
		expectedSuccessThreshold int
		expectedFailureThreshold int
	}{
		{
			name: "valid-empty",
			alert: Alert{
				Description:      nil,
				FailureThreshold: 0,
				SuccessThreshold: 0,
			},
			expectedError:            nil,
			expectedFailureThreshold: 3,
			expectedSuccessThreshold: 2,
		},
		{
			name: "invalid-description",
			alert: Alert{
				Description:      &invalidDescription,
				FailureThreshold: 10,
				SuccessThreshold: 5,
			},
			expectedError:            ErrAlertWithInvalidDescription,
			expectedFailureThreshold: 10,
			expectedSuccessThreshold: 5,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if err := scenario.alert.ValidateAndSetDefaults(); err != scenario.expectedError {
				t.Errorf("expected error %v, got %v", scenario.expectedError, err)
			}
			if scenario.alert.SuccessThreshold != scenario.expectedSuccessThreshold {
				t.Errorf("expected success threshold %v, got %v", scenario.expectedSuccessThreshold, scenario.alert.SuccessThreshold)
			}
			if scenario.alert.FailureThreshold != scenario.expectedFailureThreshold {
				t.Errorf("expected failure threshold %v, got %v", scenario.expectedFailureThreshold, scenario.alert.FailureThreshold)
			}
		})
	}
}

func TestAlert_IsEnabled(t *testing.T) {
	if !(Alert{Enabled: nil}).IsEnabled() {
		t.Error("alert.IsEnabled() should've returned true, because Enabled was set to nil")
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

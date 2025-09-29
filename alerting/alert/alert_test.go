package alert

import (
	"errors"
	"testing"
	"time"
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
		{
			name: "valid-minimum-reminder-interval-0",
			alert: Alert{
				MinimumReminderInterval: 0,
				FailureThreshold:        10,
				SuccessThreshold:        5,
			},
			expectedError:            nil,
			expectedFailureThreshold: 10,
			expectedSuccessThreshold: 5,
		},
		{
			name: "valid-minimum-reminder-interval-5m",
			alert: Alert{
				MinimumReminderInterval: 5 * time.Minute,
				FailureThreshold:        10,
				SuccessThreshold:        5,
			},
			expectedError:            nil,
			expectedFailureThreshold: 10,
			expectedSuccessThreshold: 5,
		},
		{
			name: "valid-minimum-reminder-interval-10m",
			alert: Alert{
				MinimumReminderInterval: 10 * time.Minute,
				FailureThreshold:        10,
				SuccessThreshold:        5,
			},
			expectedError:            nil,
			expectedFailureThreshold: 10,
			expectedSuccessThreshold: 5,
		},
		{
			name: "invalid-minimum-reminder-interval-1m",
			alert: Alert{
				MinimumReminderInterval: 1 * time.Minute,
				FailureThreshold:        10,
				SuccessThreshold:        5,
			},
			expectedError:            ErrAlertWithInvalidMinimumReminderInterval,
			expectedFailureThreshold: 10,
			expectedSuccessThreshold: 5,
		},
		{
			name: "invalid-minimum-reminder-interval-1s",
			alert: Alert{
				MinimumReminderInterval: 1 * time.Second,
				FailureThreshold:        10,
				SuccessThreshold:        5,
			},
			expectedError:            ErrAlertWithInvalidMinimumReminderInterval,
			expectedFailureThreshold: 10,
			expectedSuccessThreshold: 5,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if err := scenario.alert.ValidateAndSetDefaults(); !errors.Is(err, scenario.expectedError) {
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
	if !(&Alert{Enabled: nil}).IsEnabled() {
		t.Error("alert.IsEnabled() should've returned true, because Enabled was set to nil")
	}
	if value := false; (&Alert{Enabled: &value}).IsEnabled() {
		t.Error("alert.IsEnabled() should've returned false, because Enabled was set to false")
	}
	if value := true; !(&Alert{Enabled: &value}).IsEnabled() {
		t.Error("alert.IsEnabled() should've returned true, because Enabled was set to true")
	}
}

func TestAlert_GetDescription(t *testing.T) {
	if (&Alert{Description: nil}).GetDescription() != "" {
		t.Error("alert.GetDescription() should've returned an empty string, because Description was set to nil")
	}
	if value := "description"; (&Alert{Description: &value}).GetDescription() != value {
		t.Error("alert.GetDescription() should've returned false, because Description was set to 'description'")
	}
}

func TestAlert_IsSendingOnResolved(t *testing.T) {
	if (&Alert{SendOnResolved: nil}).IsSendingOnResolved() {
		t.Error("alert.IsSendingOnResolved() should've returned false, because SendOnResolved was set to nil")
	}
	if value := false; (&Alert{SendOnResolved: &value}).IsSendingOnResolved() {
		t.Error("alert.IsSendingOnResolved() should've returned false, because SendOnResolved was set to false")
	}
	if value := true; !(&Alert{SendOnResolved: &value}).IsSendingOnResolved() {
		t.Error("alert.IsSendingOnResolved() should've returned true, because SendOnResolved was set to true")
	}
}

func TestAlert_Checksum(t *testing.T) {
	description1, description2 := "a", "b"
	yes, no := true, false
	scenarios := []struct {
		name     string
		alert    Alert
		expected string
	}{
		{
			name: "barebone",
			alert: Alert{
				Type: TypeDiscord,
			},
			expected: "fed0580e44ed5701dbba73afa1f14b2c53ca5a7b8067a860441c212916057fe3",
		},
		{
			name: "with-description-1",
			alert: Alert{
				Type:        TypeDiscord,
				Description: &description1,
			},
			expected: "005f407ebe506e74a4aeb46f74c28b376debead7011e1b085da3840f72ba9707",
		},
		{
			name: "with-description-2",
			alert: Alert{
				Type:        TypeDiscord,
				Description: &description2,
			},
			expected: "3c2c4a9570cdc614006993c21f79a860a7f5afea10cf70d1a79d3c49342ef2c8",
		},
		{
			name: "with-description-2-and-enabled-false",
			alert: Alert{
				Type:        TypeDiscord,
				Enabled:     &no,
				Description: &description2,
			},
			expected: "837945c2b4cd5e961db3e63e10c348d4f1c3446ba68cf5a48e35a1ae22cf0c22",
		},
		{
			name: "with-description-2-and-enabled-true",
			alert: Alert{
				Type:        TypeDiscord,
				Enabled:     &yes, // it defaults to true if not set, but just to make sure
				Description: &description2,
			},
			expected: "3c2c4a9570cdc614006993c21f79a860a7f5afea10cf70d1a79d3c49342ef2c8",
		},
		{
			name: "with-description-2-and-enabled-true-and-send-on-resolved-true",
			alert: Alert{
				Type:           TypeDiscord,
				Enabled:        &yes,
				SendOnResolved: &yes,
				Description:    &description2,
			},
			expected: "bf1436995a880eb4a352c74c5dfee1f1b5ff6b9fc55aef9bf411b3631adfd80c",
		},
		{
			name: "with-description-2-and-failure-threshold-7",
			alert: Alert{
				Type:             TypeSlack,
				FailureThreshold: 7,
				Description:      &description2,
			},
			expected: "8bd479e18bda393d4c924f5a0d962e825002168dedaa88b445e435db7bacffd3",
		},
		{
			name: "with-description-2-and-failure-threshold-9",
			alert: Alert{
				Type:             TypeSlack,
				FailureThreshold: 9,
				Description:      &description2,
			},
			expected: "5abdfce5236e344996d264d526e769c07cb0d3d329a999769a1ff84b157ca6f1",
		},
		{
			name: "with-description-2-and-success-threshold-5",
			alert: Alert{
				Type:             TypeSlack,
				SuccessThreshold: 7,
				Description:      &description2,
			},
			expected: "c0000e73626b80e212cfc24830de7094568f648e37f3e16f9e68c7f8ef75c34c",
		},
		{
			name: "with-description-2-and-success-threshold-1",
			alert: Alert{
				Type:             TypeSlack,
				SuccessThreshold: 1,
				Description:      &description2,
			},
			expected: "5c28963b3a76104cfa4a0d79c89dd29ec596c8cfa4b1af210ec83d6d41587b5f",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			scenario.alert.ValidateAndSetDefaults()
			if checksum := scenario.alert.Checksum(); checksum != scenario.expected {
				t.Errorf("expected checksum %v, got %v", scenario.expected, checksum)
			}
		})
	}
}

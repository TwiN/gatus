package awsses

import (
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestAlertDefaultProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	invalidProviderWithOneKey := AlertProvider{From: "from@example.com", To: "to@example.com", AccessKeyID: "1"}
	if invalidProviderWithOneKey.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{From: "from@example.com", To: "to@example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
	validProviderWithKeys := AlertProvider{From: "from@example.com", To: "to@example.com", AccessKeyID: "1", SecretAccessKey: "1"}
	if !validProviderWithKeys.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_IsValidWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				To:    "to@example.com",
				Group: "",
			},
		},
	}
	if providerWithInvalidOverrideGroup.IsValid() {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				To:    "",
				Group: "group",
			},
		},
	}
	if providerWithInvalidOverrideTo.IsValid() {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		From: "from@example.com",
		To:   "to@example.com",
		Overrides: []Override{
			{
				To:    "to@example.com",
				Group: "group",
			},
		},
	}
	if !providerWithValidOverride.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name            string
		Provider        AlertProvider
		Alert           alert.Alert
		Resolved        bool
		ExpectedSubject string
		ExpectedBody    string
	}{
		{
			Name:            "triggered",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        false,
			ExpectedSubject: "[endpoint-name] Alert triggered",
			ExpectedBody:    "An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row\n\nAlert description: description-1\n\nCondition results:\n❌ [CONNECTED] == true\n❌ [STATUS] == 200\n",
		},
		{
			Name:            "resolved",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        true,
			ExpectedSubject: "[endpoint-name] Alert resolved",
			ExpectedBody:    "An alert for endpoint-name has been resolved after passing successfully 5 time(s) in a row\n\nAlert description: description-2\n\nCondition results:\n✅ [CONNECTED] == true\n✅ [STATUS] == 200\n",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			subject, body := scenario.Provider.buildMessageSubjectAndBody(
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			if subject != scenario.ExpectedSubject {
				t.Errorf("expected subject to be %s, got %s", scenario.ExpectedSubject, subject)
			}
			if body != scenario.ExpectedBody {
				t.Errorf("expected body to be %s, got %s", scenario.ExpectedBody, body)
			}
		})
	}
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	if (&AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

func TestAlertProvider_getToForGroup(t *testing.T) {
	tests := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		ExpectedOutput string
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				To:        "to@example.com",
				Overrides: nil,
			},
			InputGroup:     "",
			ExpectedOutput: "to@example.com",
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				To:        "to@example.com",
				Overrides: nil,
			},
			InputGroup:     "group",
			ExpectedOutput: "to@example.com",
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				To: "to@example.com",
				Overrides: []Override{
					{
						Group: "group",
						To:    "to01@example.com",
					},
				},
			},
			InputGroup:     "",
			ExpectedOutput: "to@example.com",
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				To: "to@example.com",
				Overrides: []Override{
					{
						Group: "group",
						To:    "to01@example.com",
					},
				},
			},
			InputGroup:     "group",
			ExpectedOutput: "to01@example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if got := tt.Provider.getToForGroup(tt.InputGroup); got != tt.ExpectedOutput {
				t.Errorf("AlertProvider.getToForGroup() = %v, want %v", got, tt.ExpectedOutput)
			}
		})
	}
}

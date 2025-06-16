package awsses

import (
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	invalidProviderWithOneKey := AlertProvider{DefaultConfig: Config{From: "from@example.com", To: "to@example.com", AccessKeyID: "1"}}
	if err := invalidProviderWithOneKey.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{From: "from@example.com", To: "to@example.com"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
	validProviderWithKeys := AlertProvider{DefaultConfig: Config{From: "from@example.com", To: "to@example.com", AccessKeyID: "1", SecretAccessKey: "1"}}
	if err := validProviderWithKeys.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{To: "to@example.com"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{To: ""},
				Group:  "group",
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{
			From: "from@example.com",
			To:   "to@example.com",
		},
		Overrides: []Override{
			{
				Config: Config{To: "to@example.com"},
				Group:  "group",
			},
		},
	}
	if err := providerWithValidOverride.Validate(); err != nil {
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

func TestAlertProvider_getConfigWithOverrides(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		InputAlert     alert.Alert
		ExpectedOutput Config
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{
					From: "from@example.com",
					To:   "to@example.com",
				},
				Overrides: nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{From: "from@example.com", To: "to@example.com"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{
					From: "from@example.com",
					To:   "to@example.com",
				},
				Overrides: nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{From: "from@example.com", To: "to@example.com"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{
					From: "from@example.com",
					To:   "to@example.com",
				},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "groupto@example.com"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{From: "from@example.com", To: "to@example.com"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					From: "from@example.com",
					To:   "to@example.com",
				},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "groupto@example.com", SecretAccessKey: "wow", AccessKeyID: "noway"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{From: "from@example.com", To: "groupto@example.com", SecretAccessKey: "wow", AccessKeyID: "noway"},
		},
		{
			Name: "provider-with-override-specify-group-but-alert-override-should-override-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					From: "from@example.com",
					To:   "to@example.com",
				},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{From: "from@example.com", To: "groupto@example.com", SecretAccessKey: "sekrit"},
					},
				},
			},
			InputGroup: "group",
			InputAlert: alert.Alert{
				ProviderOverride: map[string]any{
					"to":            "alertto@example.com",
					"access-key-id": 123,
				},
			},
			ExpectedOutput: Config{To: "alertto@example.com", From: "from@example.com", AccessKeyID: "123", SecretAccessKey: "sekrit"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.From != scenario.ExpectedOutput.From {
				t.Errorf("expected From to be %s, got %s", scenario.ExpectedOutput.From, got.From)
			}
			if got.To != scenario.ExpectedOutput.To {
				t.Errorf("expected To to be %s, got %s", scenario.ExpectedOutput.To, got.To)
			}
			if got.AccessKeyID != scenario.ExpectedOutput.AccessKeyID {
				t.Errorf("expected AccessKeyID to be %s, got %s", scenario.ExpectedOutput.AccessKeyID, got.AccessKeyID)
			}
			if got.SecretAccessKey != scenario.ExpectedOutput.SecretAccessKey {
				t.Errorf("expected SecretAccessKey to be %s, got %s", scenario.ExpectedOutput.SecretAccessKey, got.SecretAccessKey)
			}
			if got.Region != scenario.ExpectedOutput.Region {
				t.Errorf("expected Region to be %s, got %s", scenario.ExpectedOutput.Region, got.Region)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

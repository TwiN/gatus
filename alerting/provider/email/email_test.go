package email

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
	validProvider := AlertProvider{DefaultConfig: Config{From: "from@example.com", Password: "password", Host: "smtp.gmail.com", Port: 587, To: "to@example.com"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithNoCredentials(t *testing.T) {
	validProvider := AlertProvider{DefaultConfig: Config{From: "from@example.com", Host: "smtp-relay.gmail.com", Port: 587, To: "to@example.com"}}
	if err := validProvider.Validate(); err != nil {
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
			From:     "from@example.com",
			Password: "password",
			Host:     "smtp.gmail.com",
			Port:     587,
			To:       "to@example.com",
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
		Endpoint        *endpoint.Endpoint
		ExpectedSubject string
		ExpectedBody    string
	}{
		{
			Name:            "triggered",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        false,
			Endpoint:        &endpoint.Endpoint{Name: "endpoint-name"},
			ExpectedSubject: "[endpoint-name] Alert triggered",
			ExpectedBody:    "An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row\n\nAlert description: description-1\n\nCondition results:\n❌ [CONNECTED] == true\n❌ [STATUS] == 200\n",
		},
		{
			Name:            "resolved",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        true,
			Endpoint:        &endpoint.Endpoint{Name: "endpoint-name"},
			ExpectedSubject: "[endpoint-name] Alert resolved",
			ExpectedBody:    "An alert for endpoint-name has been resolved after passing successfully 5 time(s) in a row\n\nAlert description: description-2\n\nCondition results:\n✅ [CONNECTED] == true\n✅ [STATUS] == 200\n",
		},
		{
			Name:            "triggered-with-single-extra-label",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        false,
			Endpoint:        &endpoint.Endpoint{Name: "endpoint-name", ExtraLabels: map[string]string{"environment": "production"}},
			ExpectedSubject: "[endpoint-name] Alert triggered",
			ExpectedBody:    "An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row\n\nAlert description: description-1\n\nExtra labels:\n  environment: production\n\n\nCondition results:\n❌ [CONNECTED] == true\n❌ [STATUS] == 200\n",
		},
		{
			Name:            "resolved-with-single-extra-label",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        true,
			Endpoint:        &endpoint.Endpoint{Name: "endpoint-name", ExtraLabels: map[string]string{"service": "api"}},
			ExpectedSubject: "[endpoint-name] Alert resolved",
			ExpectedBody:    "An alert for endpoint-name has been resolved after passing successfully 5 time(s) in a row\n\nAlert description: description-2\n\nExtra labels:\n  service: api\n\n\nCondition results:\n✅ [CONNECTED] == true\n✅ [STATUS] == 200\n",
		},
		{
			Name:            "triggered-with-no-extra-labels",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        false,
			Endpoint:        &endpoint.Endpoint{Name: "endpoint-name", ExtraLabels: map[string]string{}},
			ExpectedSubject: "[endpoint-name] Alert triggered",
			ExpectedBody:    "An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row\n\nAlert description: description-1\n\nCondition results:\n❌ [CONNECTED] == true\n❌ [STATUS] == 200\n",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			subject, body := scenario.Provider.buildMessageSubjectAndBody(
				scenario.Endpoint,
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

func TestAlertProvider_GetConfig(t *testing.T) {
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
				DefaultConfig: Config{From: "from@example.com", To: "to@example.com", Host: "smtp.gmail.com", Port: 587, Password: "password"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{From: "from@example.com", To: "to@example.com", Host: "smtp.gmail.com", Port: 587, Password: "password"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{From: "from@example.com", To: "to@example.com", Host: "smtp.gmail.com", Port: 587, Password: "password"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{From: "from@example.com", To: "to@example.com", Host: "smtp.gmail.com", Port: 587, Password: "password"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{From: "from@example.com", To: "to@example.com", Host: "smtp.gmail.com", Port: 587, Password: "password"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "to01@example.com"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{From: "from@example.com", To: "to@example.com", Host: "smtp.gmail.com", Port: 587, Password: "password"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{From: "from@example.com", To: "to@example.com", Host: "smtp.gmail.com", Port: 587, Password: "password"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "group-to@example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{From: "from@example.com", To: "group-to@example.com", Host: "smtp.gmail.com", Port: 587, Password: "password"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{From: "from@example.com", To: "to@example.com", Host: "smtp.gmail.com", Port: 587, Password: "password"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "group-to@example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"to": "alert-to@example.com", "host": "smtp.example.com", "port": 588, "password": "hunter2"}},
			ExpectedOutput: Config{From: "from@example.com", To: "alert-to@example.com", Host: "smtp.example.com", Port: 588, Password: "hunter2"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.From != scenario.ExpectedOutput.From {
				t.Errorf("expected from to be %s, got %s", scenario.ExpectedOutput.From, got.From)
			}
			if got.To != scenario.ExpectedOutput.To {
				t.Errorf("expected to be %s, got %s", scenario.ExpectedOutput.To, got.To)
			}
			if got.Host != scenario.ExpectedOutput.Host {
				t.Errorf("expected host to be %s, got %s", scenario.ExpectedOutput.Host, got.Host)
			}
			if got.Port != scenario.ExpectedOutput.Port {
				t.Errorf("expected port to be %d, got %d", scenario.ExpectedOutput.Port, got.Port)
			}
			if got.Password != scenario.ExpectedOutput.Password {
				t.Errorf("expected password to be %s, got %s", scenario.ExpectedOutput.Password, got.Password)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

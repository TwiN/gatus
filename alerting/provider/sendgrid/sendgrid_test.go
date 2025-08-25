package sendgrid

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{DefaultConfig: Config{APIKey: "", From: "", To: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		DefaultConfig: Config{
			APIKey: "SG.test",
			From:   "from@example.com",
			To:     "to@example.com",
		},
		Overrides: []Override{
			{
				Config: Config{To: "to@example.com"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider with empty Group should not have been valid")
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err != ErrDuplicateGroupOverride {
		t.Error("provider with empty Group should return ErrDuplicateGroupOverride")
	}
	providerWithDuplicateOverrideGroups := AlertProvider{
		DefaultConfig: Config{
			APIKey: "SG.test",
			From:   "from@example.com",
			To:     "to@example.com",
		},
		Overrides: []Override{
			{
				Config: Config{To: "to1@example.com"},
				Group:  "group",
			},
			{
				Config: Config{To: "to2@example.com"},
				Group:  "group",
			},
		},
	}
	if err := providerWithDuplicateOverrideGroups.Validate(); err == nil {
		t.Error("provider with duplicate group overrides should not have been valid")
	}
	if err := providerWithDuplicateOverrideGroups.Validate(); err != ErrDuplicateGroupOverride {
		t.Error("provider with duplicate group overrides should return ErrDuplicateGroupOverride")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{
			APIKey: "SG.test",
			From:   "from@example.com",
			To:     "to@example.com",
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
	providerWithValidMultipleOverrides := AlertProvider{
		DefaultConfig: Config{
			APIKey: "SG.test",
			From:   "from@example.com",
			To:     "to@example.com",
		},
		Overrides: []Override{
			{
				Config: Config{To: "group1@example.com"},
				Group:  "group1",
			},
			{
				Config: Config{To: "group2@example.com"},
				Group:  "group2",
			},
		},
	}
	if err := providerWithValidMultipleOverrides.Validate(); err != nil {
		t.Error("provider with multiple valid overrides should've been valid")
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:     "triggered",
			Provider: AlertProvider{DefaultConfig: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusBadRequest, Body: io.NopCloser(strings.NewReader(`{"errors": [{"message": "Invalid API key"}]}`))}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			err := scenario.Provider.Send(
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
			if scenario.ExpectedError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.ExpectedError && err != nil {
				t.Error("expected no error, got", err.Error())
			}
		})
	}
}

func TestAlertProvider_buildSendGridPayload(t *testing.T) {
	provider := &AlertProvider{}
	cfg := &Config{
		From: "test@example.com",
		To:   "to1@example.com,to2@example.com",
	}
	subject := "Test Subject"
	body := "Test Body\nWith new line"
	payload := provider.buildSendGridPayload(cfg, subject, body)
	if payload.Subject != subject {
		t.Errorf("expected subject to be %s, got %s", subject, payload.Subject)
	}
	if payload.From.Email != cfg.From {
		t.Errorf("expected from email to be %s, got %s", cfg.From, payload.From.Email)
	}
	if len(payload.Personalizations) != 1 {
		t.Errorf("expected 1 personalization, got %d", len(payload.Personalizations))
	}
	if len(payload.Personalizations[0].To) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(payload.Personalizations[0].To))
	}
	if payload.Personalizations[0].To[0].Email != "to1@example.com" {
		t.Errorf("expected first recipient to be to1@example.com, got %s", payload.Personalizations[0].To[0].Email)
	}
	if payload.Personalizations[0].To[1].Email != "to2@example.com" {
		t.Errorf("expected second recipient to be to2@example.com, got %s", payload.Personalizations[0].To[1].Email)
	}
	if len(payload.Content) != 2 {
		t.Errorf("expected 2 content types, got %d", len(payload.Content))
	}
	if payload.Content[0].Type != "text/plain" {
		t.Errorf("expected first content type to be text/plain, got %s", payload.Content[0].Type)
	}
	if payload.Content[0].Value != body {
		t.Errorf("expected plain text content to be %s, got %s", body, payload.Content[0].Value)
	}
	if payload.Content[1].Type != "text/html" {
		t.Errorf("expected second content type to be text/html, got %s", payload.Content[1].Type)
	}
	expectedHTML := "Test Body<br>With new line"
	if payload.Content[1].Value != expectedHTML {
		t.Errorf("expected HTML content to be %s, got %s", expectedHTML, payload.Content[1].Value)
	}
}

func TestAlertProvider_buildMessageSubjectAndBody(t *testing.T) {
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
				DefaultConfig: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "to01@example.com"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "group-to@example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "SG.test", From: "from@example.com", To: "group-to@example.com"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "group-to@example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"api-key": "SG.override", "to": "alert-to@example.com", "from": "alert-from@example.com"}},
			ExpectedOutput: Config{APIKey: "SG.override", From: "alert-from@example.com", To: "alert-to@example.com"},
		},
		{
			Name: "provider-with-multiple-overrides-pick-correct-group",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "SG.default", From: "default@example.com", To: "default@example.com"},
				Overrides: []Override{
					{
						Group:  "group1",
						Config: Config{APIKey: "SG.group1", To: "group1@example.com"},
					},
					{
						Group:  "group2",
						Config: Config{APIKey: "SG.group2", From: "group2@example.com"},
					},
				},
			},
			InputGroup:     "group2",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "SG.group2", From: "group2@example.com", To: "default@example.com"},
		},
		{
			Name: "provider-partial-override-hierarchy",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "SG.default", From: "default@example.com", To: "default@example.com"},
				Overrides: []Override{
					{
						Group:  "test-group",
						Config: Config{From: "group@example.com"},
					},
				},
			},
			InputGroup:     "test-group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"to": "alert@example.com"}},
			ExpectedOutput: Config{APIKey: "SG.default", From: "group@example.com", To: "alert@example.com"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.APIKey != scenario.ExpectedOutput.APIKey {
				t.Errorf("expected APIKey to be %s, got %s", scenario.ExpectedOutput.APIKey, got.APIKey)
			}
			if got.From != scenario.ExpectedOutput.From {
				t.Errorf("expected From to be %s, got %s", scenario.ExpectedOutput.From, got.From)
			}
			if got.To != scenario.ExpectedOutput.To {
				t.Errorf("expected To to be %s, got %s", scenario.ExpectedOutput.To, got.To)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	scenarios := []struct {
		Name          string
		Config        Config
		ExpectedError error
	}{
		{
			Name:          "missing-api-key",
			Config:        Config{APIKey: "", From: "test@example.com", To: "to@example.com"},
			ExpectedError: ErrAPIKeyNotSet,
		},
		{
			Name:          "missing-from",
			Config:        Config{APIKey: "SG.test", From: "", To: "to@example.com"},
			ExpectedError: ErrFromNotSet,
		},
		{
			Name:          "missing-to",
			Config:        Config{APIKey: "SG.test", From: "test@example.com", To: ""},
			ExpectedError: ErrToNotSet,
		},
		{
			Name:          "valid-config",
			Config:        Config{APIKey: "SG.test", From: "test@example.com", To: "to@example.com"},
			ExpectedError: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := scenario.Config.Validate()
			if scenario.ExpectedError == nil && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if scenario.ExpectedError != nil && err == nil {
				t.Errorf("expected error %v, got none", scenario.ExpectedError)
			}
			if scenario.ExpectedError != nil && err != nil && err.Error() != scenario.ExpectedError.Error() {
				t.Errorf("expected error %v, got %v", scenario.ExpectedError, err)
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	config := Config{APIKey: "SG.original", From: "from@example.com", To: "to@example.com"}
	override := Config{APIKey: "SG.override", To: "override@example.com"}
	config.Merge(&override)
	if config.APIKey != "SG.override" {
		t.Errorf("expected APIKey to be SG.override, got %s", config.APIKey)
	}
	if config.From != "from@example.com" {
		t.Errorf("expected From to remain from@example.com, got %s", config.From)
	}
	if config.To != "override@example.com" {
		t.Errorf("expected To to be override@example.com, got %s", config.To)
	}
}

func TestConfig_MergeWithClientConfig(t *testing.T) {
	config := Config{APIKey: "SG.original", From: "from@example.com", To: "to@example.com"}
	override := Config{APIKey: "SG.override", ClientConfig: &client.Config{Timeout: 30000}}
	config.Merge(&override)
	if config.APIKey != "SG.override" {
		t.Errorf("expected APIKey to be SG.override, got %s", config.APIKey)
	}
	if config.ClientConfig == nil {
		t.Error("expected ClientConfig to be set")
	}
	if config.ClientConfig.Timeout != 30000 {
		t.Errorf("expected ClientConfig.Timeout to be 30000, got %d", config.ClientConfig.Timeout)
	}
	config2 := Config{APIKey: "SG.test", From: "from@example.com", To: "to@example.com", ClientConfig: &client.Config{Timeout: 10000}}
	override2 := Config{APIKey: "SG.override2"}
	config2.Merge(&override2)
	if config2.ClientConfig.Timeout != 10000 {
		t.Errorf("expected ClientConfig.Timeout to remain 10000, got %d", config2.ClientConfig.Timeout)
	}
}
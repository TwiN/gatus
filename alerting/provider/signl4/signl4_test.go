package signl4

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{DefaultConfig: Config{TeamSecret: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{TeamSecret: "team-secret-123"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{TeamSecret: "team-secret-123"},
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
				Config: Config{TeamSecret: ""},
				Group:  "group",
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider team secret shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{TeamSecret: "team-secret-123"},
		Overrides: []Override{
			{
				Config: Config{TeamSecret: "team-secret-override"},
				Group:  "group",
			},
		},
	}
	if err := providerWithValidOverride.Validate(); err != nil {
		t.Error("provider should've been valid")
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
			Provider: AlertProvider{DefaultConfig: Config{TeamSecret: "team-secret-123"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{TeamSecret: "team-secret-123"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{TeamSecret: "team-secret-123"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{TeamSecret: "team-secret-123"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
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

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Endpoint     endpoint.Endpoint
		Alert        alert.Alert
		NoConditions bool
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"Title\":\"TRIGGERED: name\",\"Message\":\"An alert for name has been triggered due to having failed 3 time(s) in a row\\n\\nDescription: description-1\\n\\nCondition results:\\n✗ [CONNECTED] == true\\n✗ [STATUS] == 200\\n\",\"X-S4-Service\":\"name\",\"X-S4-Status\":\"new\",\"X-S4-ExternalID\":\"gatus-_name\"}",
		},
		{
			Name:         "triggered-with-group",
			Provider:     AlertProvider{},
			Endpoint:     endpoint.Endpoint{Name: "name", Group: "group"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"Title\":\"TRIGGERED: group/name\",\"Message\":\"An alert for group/name has been triggered due to having failed 3 time(s) in a row\\n\\nDescription: description-1\\n\\nCondition results:\\n✗ [CONNECTED] == true\\n✗ [STATUS] == 200\\n\",\"X-S4-Service\":\"group/name\",\"X-S4-Status\":\"new\",\"X-S4-ExternalID\":\"gatus-group_name\"}",
		},
		{
			Name:         "triggered-with-no-conditions",
			NoConditions: true,
			Provider:     AlertProvider{},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"Title\":\"TRIGGERED: name\",\"Message\":\"An alert for name has been triggered due to having failed 3 time(s) in a row\\n\\nDescription: description-1\",\"X-S4-Service\":\"name\",\"X-S4-Status\":\"new\",\"X-S4-ExternalID\":\"gatus-_name\"}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"Title\":\"RESOLVED: name\",\"Message\":\"An alert for name has been resolved after passing successfully 5 time(s) in a row\\n\\nDescription: description-2\\n\\nCondition results:\\n✓ [CONNECTED] == true\\n✓ [STATUS] == 200\\n\",\"X-S4-Service\":\"name\",\"X-S4-Status\":\"resolved\",\"X-S4-ExternalID\":\"gatus-_name\"}",
		},
		{
			Name:         "resolved-with-group",
			Provider:     AlertProvider{},
			Endpoint:     endpoint.Endpoint{Name: "name", Group: "group"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"Title\":\"RESOLVED: group/name\",\"Message\":\"An alert for group/name has been resolved after passing successfully 5 time(s) in a row\\n\\nDescription: description-2\\n\\nCondition results:\\n✓ [CONNECTED] == true\\n✓ [STATUS] == 200\\n\",\"X-S4-Service\":\"group/name\",\"X-S4-Status\":\"resolved\",\"X-S4-ExternalID\":\"gatus-group_name\"}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var conditionResults []*endpoint.ConditionResult
			if !scenario.NoConditions {
				conditionResults = []*endpoint.ConditionResult{
					{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
					{Condition: "[STATUS] == 200", Success: scenario.Resolved},
				}
			}
			body, err := scenario.Provider.buildRequestBody(
				&scenario.Endpoint,
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: conditionResults,
				},
				scenario.Resolved,
			)
			if err != nil {
				t.Fatalf("buildRequestBody returned an error: %v", err)
			}
			if string(body) != scenario.ExpectedBody {
				t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
			}
			out := make(map[string]interface{})
			if err := json.Unmarshal(body, &out); err != nil {
				t.Error("expected body to be valid JSON, got error:", err.Error())
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
				DefaultConfig: Config{TeamSecret: "team-secret-123"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{TeamSecret: "team-secret-123"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{TeamSecret: "team-secret-123"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{TeamSecret: "team-secret-123"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{TeamSecret: "team-secret-123"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{TeamSecret: "team-secret-override"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{TeamSecret: "team-secret-123"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{TeamSecret: "team-secret-123"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{TeamSecret: "team-secret-override"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{TeamSecret: "team-secret-override"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{TeamSecret: "team-secret-123"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{TeamSecret: "team-secret-override"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"team-secret": "team-secret-alert"}},
			ExpectedOutput: Config{TeamSecret: "team-secret-alert"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.TeamSecret != scenario.ExpectedOutput.TeamSecret {
				t.Errorf("expected team secret to be %s, got %s", scenario.ExpectedOutput.TeamSecret, got.TeamSecret)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

func TestAlertProvider_GetConfigWithInvalidAlertOverride(t *testing.T) {
	// Test case 1: Empty override should be ignored, default config should be used
	provider := AlertProvider{
		DefaultConfig: Config{TeamSecret: "team-secret-123"},
	}
	alertWithEmptyOverride := alert.Alert{
		ProviderOverride: map[string]any{"team-secret": ""},
	}
	cfg, err := provider.GetConfig("", &alertWithEmptyOverride)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if cfg.TeamSecret != "team-secret-123" {
		t.Errorf("expected team secret to remain default 'team-secret-123', got %s", cfg.TeamSecret)
	}

	// Test case 2: Invalid default config with no valid override should fail
	providerWithInvalidDefault := AlertProvider{
		DefaultConfig: Config{TeamSecret: ""},
	}
	alertWithEmptyOverride2 := alert.Alert{
		ProviderOverride: map[string]any{"team-secret": ""},
	}
	_, err = providerWithInvalidDefault.GetConfig("", &alertWithEmptyOverride2)
	if err == nil {
		t.Error("expected error due to invalid default config, got none")
	}
	if err != ErrTeamSecretNotSet {
		t.Errorf("expected ErrTeamSecretNotSet, got %v", err)
	}
}

func TestAlertProvider_ValidateWithDuplicateGroupOverride(t *testing.T) {
	providerWithDuplicateOverride := AlertProvider{
		DefaultConfig: Config{TeamSecret: "team-secret-123"},
		Overrides: []Override{
			{
				Group:  "group1",
				Config: Config{TeamSecret: "team-secret-override-1"},
			},
			{
				Group:  "group1",
				Config: Config{TeamSecret: "team-secret-override-2"},
			},
		},
	}
	if err := providerWithDuplicateOverride.Validate(); err == nil {
		t.Error("provider should not have been valid due to duplicate group override")
	}
	if err := providerWithDuplicateOverride.Validate(); err != ErrDuplicateGroupOverride {
		t.Errorf("expected ErrDuplicateGroupOverride, got %v", providerWithDuplicateOverride.Validate())
	}
}

func TestAlertProvider_ValidateOverridesWithInvalidAlert(t *testing.T) {
	provider := AlertProvider{
		DefaultConfig: Config{TeamSecret: ""},
	}
	alertWithEmptyOverride := alert.Alert{
		ProviderOverride: map[string]any{"team-secret": ""},
	}
	err := provider.ValidateOverrides("", &alertWithEmptyOverride)
	if err == nil {
		t.Error("expected error due to invalid default config, got none")
	}
	if err != ErrTeamSecretNotSet {
		t.Errorf("expected ErrTeamSecretNotSet, got %v", err)
	}
}

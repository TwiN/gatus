package gotify

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestAlertProvider_Validate(t *testing.T) {
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected bool
	}{
		{
			name:     "valid",
			provider: AlertProvider{DefaultConfig: Config{ServerURL: "https://gotify.example.com", Token: "faketoken"}},
			expected: true,
		},
		{
			name:     "invalid-server-url",
			provider: AlertProvider{DefaultConfig: Config{ServerURL: "", Token: "faketoken"}},
			expected: false,
		},
		{
			name:     "invalid-app-token",
			provider: AlertProvider{DefaultConfig: Config{ServerURL: "https://gotify.example.com", Token: ""}},
			expected: false,
		},
		{
			name:     "no-priority-should-use-default-value",
			provider: AlertProvider{DefaultConfig: Config{ServerURL: "https://gotify.example.com", Token: "faketoken"}},
			expected: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if err := scenario.provider.Validate(); (err == nil) != scenario.expected {
				t.Errorf("expected: %t, got: %t", scenario.expected, err == nil)
			}
		})
	}
}

func TestAlertProvider_buildRequestBody(t *testing.T) {
	var (
		description = "custom-description"
		//title       = "custom-title"
		endpointName = "custom-endpoint"
	)
	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{DefaultConfig: Config{ServerURL: "https://gotify.example.com", Token: "faketoken"}},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: fmt.Sprintf("{\"message\":\"An alert for `%s` has been triggered due to having failed 3 time(s) in a row with the following description: %s\\n✕ - [CONNECTED] == true\\n✕ - [STATUS] == 200\",\"title\":\"Gatus: custom-endpoint\",\"priority\":0}", endpointName, description),
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{ServerURL: "https://gotify.example.com", Token: "faketoken"}},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: fmt.Sprintf("{\"message\":\"An alert for `%s` has been resolved after passing successfully 5 time(s) in a row with the following description: %s\\n✓ - [CONNECTED] == true\\n✓ - [STATUS] == 200\",\"title\":\"Gatus: custom-endpoint\",\"priority\":0}", endpointName, description),
		},
		{
			Name:         "custom-title",
			Provider:     AlertProvider{DefaultConfig: Config{ServerURL: "https://gotify.example.com", Token: "faketoken", Title: "custom-title"}},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: fmt.Sprintf("{\"message\":\"An alert for `%s` has been triggered due to having failed 3 time(s) in a row with the following description: %s\\n✕ - [CONNECTED] == true\\n✕ - [STATUS] == 200\",\"title\":\"custom-title\",\"priority\":0}", endpointName, description),
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&scenario.Provider.DefaultConfig,
				&endpoint.Endpoint{Name: endpointName},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
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
	provider := AlertProvider{DefaultAlert: &alert.Alert{}}
	if provider.GetDefaultAlert() != provider.DefaultAlert {
		t.Error("expected default alert to be returned")
	}
}

func TestAlertProvider_GetConfig(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputAlert     alert.Alert
		ExpectedOutput Config
	}{
		{
			Name: "provider-no-override-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{ServerURL: "https://gotify.example.com", Token: "12345"},
			},
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{ServerURL: "https://gotify.example.com", Token: "12345", Priority: DefaultPriority},
		},
		{
			Name: "provider-with-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{ServerURL: "https://gotify.example.com", Token: "12345"},
			},
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"server-url": "https://gotify.group-example.com", "token": "54321", "title": "alert-title", "priority": 3}},
			ExpectedOutput: Config{ServerURL: "https://gotify.group-example.com", Token: "54321", Title: "alert-title", Priority: 3},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig("", &scenario.InputAlert)
			if err != nil {
				t.Error("expected no error, got:", err.Error())
			}
			if got.ServerURL != scenario.ExpectedOutput.ServerURL {
				t.Errorf("expected server URL to be %s, got %s", scenario.ExpectedOutput.ServerURL, got.ServerURL)
			}
			if got.Token != scenario.ExpectedOutput.Token {
				t.Errorf("expected token to be %s, got %s", scenario.ExpectedOutput.Token, got.Token)
			}
			if got.Title != scenario.ExpectedOutput.Title {
				t.Errorf("expected title to be %s, got %s", scenario.ExpectedOutput.Title, got.Title)
			}
			if got.Priority != scenario.ExpectedOutput.Priority {
				t.Errorf("expected priority to be %d, got %d", scenario.ExpectedOutput.Priority, got.Priority)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides("", &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

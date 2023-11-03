package gotify

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/core"
)

func TestAlertProvider_IsValid(t *testing.T) {
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected bool
	}{
		{
			name:     "valid",
			provider: AlertProvider{ServerURL: "https://gotify.example.com", Token: "faketoken"},
			expected: true,
		},
		{
			name:     "invalid-server-url",
			provider: AlertProvider{ServerURL: "", Token: "faketoken"},
			expected: false,
		},
		{
			name:     "invalid-app-token",
			provider: AlertProvider{ServerURL: "https://gotify.example.com", Token: ""},
			expected: false,
		},
		{
			name:     "no-priority-should-use-default-value",
			provider: AlertProvider{ServerURL: "https://gotify.example.com", Token: "faketoken"},
			expected: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if scenario.provider.IsValid() != scenario.expected {
				t.Errorf("expected %t, got %t", scenario.expected, scenario.provider.IsValid())
			}
		})
	}
}

func TestAlertProvider_buildRequestBody(t *testing.T) {
	var (
		description = "custom-description"
		//title       = "custom-title"
		endpoint = "custom-endpoint"
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
			Provider:     AlertProvider{ServerURL: "https://gotify.example.com", Token: "faketoken"},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: fmt.Sprintf("{\"message\":\"An alert for `%s` has been triggered due to having failed 3 time(s) in a row with the following description: %s\\n✕ - [CONNECTED] == true\\n✕ - [STATUS] == 200\",\"title\":\"Gatus: custom-endpoint\",\"priority\":0}", endpoint, description),
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{ServerURL: "https://gotify.example.com", Token: "faketoken"},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: fmt.Sprintf("{\"message\":\"An alert for `%s` has been resolved after passing successfully 5 time(s) in a row with the following description: %s\\n✓ - [CONNECTED] == true\\n✓ - [STATUS] == 200\",\"title\":\"Gatus: custom-endpoint\",\"priority\":0}", endpoint, description),
		},
		{
			Name:         "custom-title",
			Provider:     AlertProvider{ServerURL: "https://gotify.example.com", Token: "faketoken", Title: "custom-title"},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: fmt.Sprintf("{\"message\":\"An alert for `%s` has been triggered due to having failed 3 time(s) in a row with the following description: %s\\n✕ - [CONNECTED] == true\\n✕ - [STATUS] == 200\",\"title\":\"custom-title\",\"priority\":0}", endpoint, description),
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&core.Endpoint{Name: endpoint},
				&scenario.Alert,
				&core.Result{
					ConditionResults: []*core.ConditionResult{
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

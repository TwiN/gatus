package ntfy

import (
	"encoding/json"
	"testing"

	"github.com/TwiN/gatus/v4/alerting/alert"
	"github.com/TwiN/gatus/v4/core"
)

func TestAlertDefaultProvider_IsValid(t *testing.T) {
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected bool
	}{
		{
			name:     "valid",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
			expected: true,
		},
		{
			name:     "no-url-should-use-default-value",
			provider: AlertProvider{Topic: "example", Priority: 1},
			expected: true,
		},
		{
			name:     "invalid-topic",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "", Priority: 1},
			expected: false,
		},
		{
			name:     "invalid-priority-too-high",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 6},
			expected: false,
		},
		{
			name:     "invalid-priority-too-low",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: -1},
			expected: false,
		},
		{
			name:     "no-priority-should-use-default-value",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example"},
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
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\n  \"topic\": \"example\",\n  \"title\": \"Gatus\",\n  \"message\": \"endpoint-name - description-1\",\n  \"tags\": [\"x\"],\n  \"priority\": 1\n}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 2},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\n  \"topic\": \"example\",\n  \"title\": \"Gatus\",\n  \"message\": \"endpoint-name - description-2\",\n  \"tags\": [\"white_check_mark\"],\n  \"priority\": 2\n}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&core.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&core.Result{
					ConditionResults: []*core.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			if body != scenario.ExpectedBody {
				t.Errorf("expected %s, got %s", scenario.ExpectedBody, body)
			}
			out := make(map[string]interface{})
			if err := json.Unmarshal([]byte(body), &out); err != nil {
				t.Error("expected body to be valid JSON, got error:", err.Error())
			}
		})
	}
}

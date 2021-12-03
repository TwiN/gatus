package mattermost

import (
	"encoding/json"
	"testing"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/core"
)

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{WebhookURL: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{WebhookURL: "http://example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
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
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\n  \"text\": \"\",\n  \"username\": \"gatus\",\n  \"icon_url\": \"https://raw.githubusercontent.com/TwiN/gatus/master/.github/assets/logo.png\",\n  \"attachments\": [\n    {\n      \"title\": \":rescue_worker_helmet: Gatus\",\n      \"fallback\": \"Gatus - An alert for *endpoint-name* has been triggered due to having failed 3 time(s) in a row\",\n      \"text\": \"An alert for *endpoint-name* has been triggered due to having failed 3 time(s) in a row:\\n> description-1\",\n      \"short\": false,\n      \"color\": \"#DD0000\",\n      \"fields\": [\n        {\n          \"title\": \"URL\",\n          \"value\": \"\",\n          \"short\": false\n        },\n        {\n          \"title\": \"Condition results\",\n          \"value\": \":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n\",\n          \"short\": false\n        }\n      ]\n    }\n  ]\n}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\n  \"text\": \"\",\n  \"username\": \"gatus\",\n  \"icon_url\": \"https://raw.githubusercontent.com/TwiN/gatus/master/.github/assets/logo.png\",\n  \"attachments\": [\n    {\n      \"title\": \":rescue_worker_helmet: Gatus\",\n      \"fallback\": \"Gatus - An alert for *endpoint-name* has been resolved after passing successfully 5 time(s) in a row\",\n      \"text\": \"An alert for *endpoint-name* has been resolved after passing successfully 5 time(s) in a row:\\n> description-2\",\n      \"short\": false,\n      \"color\": \"#36A64F\",\n      \"fields\": [\n        {\n          \"title\": \"URL\",\n          \"value\": \"\",\n          \"short\": false\n        },\n        {\n          \"title\": \"Condition results\",\n          \"value\": \":white_check_mark: - `[CONNECTED] == true`\\n:white_check_mark: - `[STATUS] == 200`\\n\",\n          \"short\": false\n        }\n      ]\n    }\n  ]\n}",
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

package telegram

import (
	"encoding/json"
	"testing"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/core"
)

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{Token: "", ID: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}
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
			Provider:     AlertProvider{ID: "123"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"chat_id\": \"123\", \"text\": \"⛑ *Gatus* \\nAn alert for *endpoint-name* has been triggered:\\n—\\n    _healthcheck failed 3 time(s) in a row_\\n—   \\n*Description* \\n_description-1_  \\n\\n*Condition results*\\n❌ - `[CONNECTED] == true`\\n❌ - `[STATUS] == 200`\\n\", \"parse_mode\": \"MARKDOWN\"}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{ID: "123"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"chat_id\": \"123\", \"text\": \"⛑ *Gatus* \\nAn alert for *endpoint-name* has been resolved:\\n—\\n    _healthcheck passing successfully 3 time(s) in a row_\\n—   \\n*Description* \\n_description-2_  \\n\\n*Condition results*\\n✅ - `[CONNECTED] == true`\\n✅ - `[STATUS] == 200`\\n\", \"parse_mode\": \"MARKDOWN\"}",
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

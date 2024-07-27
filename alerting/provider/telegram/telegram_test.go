package telegram

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertDefaultProvider_IsValid(t *testing.T) {
	t.Run("invalid-provider", func(t *testing.T) {
		invalidProvider := AlertProvider{Token: "", ID: ""}
		if invalidProvider.IsValid() {
			t.Error("provider shouldn't have been valid")
		}
	})
	t.Run("valid-provider", func(t *testing.T) {
		validProvider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}
		if validProvider.ClientConfig != nil {
			t.Error("provider client config should have been nil prior to IsValid() being executed")
		}
		if !validProvider.IsValid() {
			t.Error("provider should've been valid")
		}
		if validProvider.ClientConfig == nil {
			t.Error("provider client config should have been set after IsValid() was executed")
		}
	})
}

func TestAlertProvider_IsValidWithOverrides(t *testing.T) {
	t.Run("invalid-provider-override-nonexist-group", func(t *testing.T) {
		invalidProvider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678", Overrides: []*Override{{token: "token", id: "id"}}}
		if invalidProvider.IsValid() {
			t.Error("provider shouldn't have been valid")
		}
	})
	t.Run("invalid-provider-override-duplicate-group", func(t *testing.T) {
		invalidProvider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678", Overrides: []*Override{{group: "group1", token: "token", id: "id"}, {group: "group1", id: "id2"}}}
		if invalidProvider.IsValid() {
			t.Error("provider shouldn't have been valid")
		}
	})
	t.Run("valid-provider", func(t *testing.T) {
		validProvider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678", Overrides: []*Override{{group: "group", token: "token", id: "id"}}}
		if validProvider.ClientConfig != nil {
			t.Error("provider client config should have been nil prior to IsValid() being executed")
		}
		if !validProvider.IsValid() {
			t.Error("provider should've been valid")
		}
		if validProvider.ClientConfig == nil {
			t.Error("provider client config should have been set after IsValid() was executed")
		}
	})
}

func TestAlertProvider_getTokenAndIDForGroup(t *testing.T) {
	t.Run("get-token-with-override", func(t *testing.T) {
		provider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678", Overrides: []*Override{{group: "group", token: "overrideToken", id: "overrideID"}}}
		token := provider.getTokenForGroup("group")
		if token != "overrideToken" {
			t.Error("token should have been 'overrideToken'")
		}
		id := provider.getIDForGroup("group")
		if id != "overrideID" {
			t.Error("id should have been 'overrideID'")
		}
	})
	t.Run("get-default-token-with-overridden-id", func(t *testing.T) {
		provider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678", Overrides: []*Override{{group: "group", id: "overrideID"}}}
		token := provider.getTokenForGroup("group")
		if token != provider.Token {
			t.Error("token should have been the default token")
		}
		id := provider.getIDForGroup("group")
		if id != "overrideID" {
			t.Error("id should have been 'overrideID'")
		}
	})
	t.Run("get-default-token-with-overridden-token", func(t *testing.T) {
		provider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678", Overrides: []*Override{{group: "group", token: "overrideToken"}}}
		token := provider.getTokenForGroup("group")
		if token != "overrideToken" {
			t.Error("token should have been 'overrideToken'")
		}
		id := provider.getIDForGroup("group")
		if id != provider.ID {
			t.Error("id should have been the default id")
		}
	})
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
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{},
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
		Alert        alert.Alert
		NoConditions bool
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{ID: "123"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"chat_id\":\"123\",\"text\":\"⛑ *Gatus* \\nAn alert for *endpoint-name* has been triggered:\\n—\\n    _healthcheck failed 3 time(s) in a row_\\n—   \\n*Description* \\n_description-1_  \\n\\n*Condition results*\\n❌ - `[CONNECTED] == true`\\n❌ - `[STATUS] == 200`\\n\",\"parse_mode\":\"MARKDOWN\"}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{ID: "123"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"chat_id\":\"123\",\"text\":\"⛑ *Gatus* \\nAn alert for *endpoint-name* has been resolved:\\n—\\n    _healthcheck passing successfully 5 time(s) in a row_\\n—   \\n*Description* \\n_description-2_  \\n\\n*Condition results*\\n✅ - `[CONNECTED] == true`\\n✅ - `[STATUS] == 200`\\n\",\"parse_mode\":\"MARKDOWN\"}",
		},
		{
			Name:         "resolved-with-no-conditions",
			NoConditions: true,
			Provider:     AlertProvider{ID: "123"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"chat_id\":\"123\",\"text\":\"⛑ *Gatus* \\nAn alert for *endpoint-name* has been resolved:\\n—\\n    _healthcheck passing successfully 5 time(s) in a row_\\n—   \\n*Description* \\n_description-2_  \\n\",\"parse_mode\":\"MARKDOWN\"}",
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
			body := scenario.Provider.buildRequestBody(
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&endpoint.Result{ConditionResults: conditionResults},
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
	if (&AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

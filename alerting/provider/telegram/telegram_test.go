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

func TestAlertProvider_Validate(t *testing.T) {
	t.Run("invalid-provider", func(t *testing.T) {
		invalidProvider := AlertProvider{DefaultConfig: Config{Token: "", ID: ""}}
		if err := invalidProvider.Validate(); err == nil {
			t.Error("provider shouldn't have been valid")
		}
	})
	t.Run("valid-provider", func(t *testing.T) {
		validProvider := AlertProvider{DefaultConfig: Config{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}}
		if err := validProvider.Validate(); err != nil {
			t.Error("provider should've been valid")
		}
	})
	t.Run("invalid-provider-override-nonexist-group", func(t *testing.T) {
		invalidProvider := AlertProvider{DefaultConfig: Config{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}, Overrides: []*Override{{Config: Config{Token: "token", ID: "id"}}}}
		if err := invalidProvider.Validate(); err == nil {
			t.Error("provider shouldn't have been valid")
		}
	})
	t.Run("invalid-provider-override-duplicate-group", func(t *testing.T) {
		invalidProvider := AlertProvider{DefaultConfig: Config{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}, Overrides: []*Override{{Group: "group1", Config: Config{Token: "token", ID: "id"}}, {Group: "group1", Config: Config{ID: "id2"}}}}
		if err := invalidProvider.Validate(); err == nil {
			t.Error("provider shouldn't have been valid")
		}
	})
	t.Run("valid-provider-with-overrides", func(t *testing.T) {
		validProvider := AlertProvider{DefaultConfig: Config{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}, Overrides: []*Override{{Group: "group", Config: Config{Token: "token", ID: "id"}}}}
		if err := validProvider.Validate(); err != nil {
			t.Error("provider should've been valid")
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
			Provider: AlertProvider{DefaultConfig: Config{ID: "123", Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{ID: "123", Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{ID: "123", Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{ID: "123", Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11"}},
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
	descriptionWithLink := "[link](https://example.org/)"
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
			Provider:     AlertProvider{DefaultConfig: Config{ID: "123"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"chat_id\":\"123\",\"text\":\"⛑ *Gatus* \\nAn alert for *endpoint-name* has been triggered:\\n—\\n    _healthcheck failed 3 time(s) in a row_\\n—   \\n*Description* \\ndescription-1  \\n\\n*Condition results*\\n❌ - `[CONNECTED] == true`\\n❌ - `[STATUS] == 200`\\n\",\"parse_mode\":\"MARKDOWN\"}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{ID: "123"}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"chat_id\":\"123\",\"text\":\"⛑ *Gatus* \\nAn alert for *endpoint-name* has been resolved:\\n—\\n    _healthcheck passing successfully 5 time(s) in a row_\\n—   \\n*Description* \\ndescription-2  \\n\\n*Condition results*\\n✅ - `[CONNECTED] == true`\\n✅ - `[STATUS] == 200`\\n\",\"parse_mode\":\"MARKDOWN\"}",
		},
		{
			Name:         "resolved-with-no-conditions",
			NoConditions: true,
			Provider:     AlertProvider{DefaultConfig: Config{ID: "123"}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"chat_id\":\"123\",\"text\":\"⛑ *Gatus* \\nAn alert for *endpoint-name* has been resolved:\\n—\\n    _healthcheck passing successfully 5 time(s) in a row_\\n—   \\n*Description* \\ndescription-2  \\n\",\"parse_mode\":\"MARKDOWN\"}",
		},
		{
			Name:         "send to topic",
			Provider:     AlertProvider{DefaultConfig: Config{ID: "123", TopicID: "7"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"chat_id\":\"123\",\"text\":\"⛑ *Gatus* \\nAn alert for *endpoint-name* has been triggered:\\n—\\n    _healthcheck failed 3 time(s) in a row_\\n—   \\n*Description* \\ndescription-1  \\n\\n*Condition results*\\n❌ - `[CONNECTED] == true`\\n❌ - `[STATUS] == 200`\\n\",\"parse_mode\":\"MARKDOWN\",\"message_thread_id\":\"7\"}",
		},
		{
			Name:         "triggered with link in description",
			Provider:     AlertProvider{DefaultConfig: Config{ID: "123"}},
			Alert:        alert.Alert{Description: &descriptionWithLink, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"chat_id\":\"123\",\"text\":\"⛑ *Gatus* \\nAn alert for *endpoint-name* has been triggered:\\n—\\n    _healthcheck failed 3 time(s) in a row_\\n—   \\n*Description* \\n[link](https://example.org/)  \\n\\n*Condition results*\\n❌ - `[CONNECTED] == true`\\n❌ - `[STATUS] == 200`\\n\",\"parse_mode\":\"MARKDOWN\"}",
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
				&scenario.Provider.DefaultConfig,
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

func TestAlertProvider_GetConfig(t *testing.T) {
	t.Run("get-token-with-override", func(t *testing.T) {
		provider := AlertProvider{DefaultConfig: Config{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}, Overrides: []*Override{{Group: "group", Config: Config{Token: "groupToken", ID: "overrideID"}}}}
		cfg, err := provider.GetConfig("group", &alert.Alert{})
		if err != nil {
			t.Error("expected no error, got", err)
		}
		if cfg.Token != "groupToken" {
			t.Error("token should have been 'groupToken'")
		}
		if cfg.ID != "overrideID" {
			t.Error("id should have been 'overrideID'")
		}
	})
	t.Run("get-default-token-with-overridden-id", func(t *testing.T) {
		provider := AlertProvider{DefaultConfig: Config{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}, Overrides: []*Override{{Group: "group", Config: Config{ID: "overrideID"}}}}
		cfg, err := provider.GetConfig("group", &alert.Alert{})
		if err != nil {
			t.Error("expected no error, got", err)
		}
		if cfg.Token != provider.DefaultConfig.Token {
			t.Error("token should have been the default token")
		}
		if cfg.ID != "overrideID" {
			t.Error("id should have been 'overrideID'")
		}
	})
	t.Run("get-default-token-with-overridden-token", func(t *testing.T) {
		provider := AlertProvider{DefaultConfig: Config{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}, Overrides: []*Override{{Group: "group", Config: Config{Token: "groupToken"}}}}
		cfg, err := provider.GetConfig("group", &alert.Alert{})
		if err != nil {
			t.Error("expected no error, got", err)
		}
		if cfg.Token != "groupToken" {
			t.Error("token should have been 'groupToken'")
		}
		if cfg.ID != provider.DefaultConfig.ID {
			t.Error("id should have been the default id")
		}
	})
	t.Run("get-default-token-with-overridden-token-and-alert-token-override", func(t *testing.T) {
		provider := AlertProvider{DefaultConfig: Config{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}, Overrides: []*Override{{Group: "group", Config: Config{Token: "groupToken"}}}}
		alert := &alert.Alert{ProviderOverride: map[string]any{"token": "alertToken"}}
		cfg, err := provider.GetConfig("group", alert)
		if err != nil {
			t.Error("expected no error, got", err)
		}
		if cfg.Token != "alertToken" {
			t.Error("token should have been 'alertToken'")
		}
		if cfg.ID != provider.DefaultConfig.ID {
			t.Error("id should have been the default id")
		}
		// Test ValidateOverrides as well, since it really just calls GetConfig
		if err = provider.ValidateOverrides("group", alert); err != nil {
			t.Errorf("unexpected error: %s", err)
		}
	})
}

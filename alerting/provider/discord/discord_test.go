package discord

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
	invalidProvider := AlertProvider{DefaultConfig: Config{WebhookURL: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{WebhookURL: "http://example.com"},
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
				Config: Config{WebhookURL: ""},
				Group:  "group",
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{
			WebhookURL: "http://example.com",
		},
		Overrides: []Override{
			{
				Config: Config{WebhookURL: "http://example.com"},
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
	title := "provider-title"
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
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "triggered-with-modified-title",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com", Title: title}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-with-webhook-override",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3, ProviderOverride: map[string]any{"webhook-url": "http://example01.com"}},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-with-message-content",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com", MessageContent: "<@123456789>"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
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

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	title := "provider-title"
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
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"content\":\"\",\"embeds\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"description\":\"An alert for **endpoint-name** has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\",\"color\":15158332,\"fields\":[{\"name\":\"Condition results\",\"value\":\":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n:x: - `[BODY] != \\\"\\\"`\\n\",\"inline\":false}]}]}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"content\":\"\",\"embeds\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"description\":\"An alert for **endpoint-name** has been resolved after passing successfully 5 time(s) in a row:\\n\\u003e description-2\",\"color\":3066993,\"fields\":[{\"name\":\"Condition results\",\"value\":\":white_check_mark: - `[CONNECTED] == true`\\n:white_check_mark: - `[STATUS] == 200`\\n:white_check_mark: - `[BODY] != \\\"\\\"`\\n\",\"inline\":false}]}]}",
		},
		{
			Name:         "triggered-with-modified-title",
			Provider:     AlertProvider{DefaultConfig: Config{Title: title}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"content\":\"\",\"embeds\":[{\"title\":\"provider-title\",\"description\":\"An alert for **endpoint-name** has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\",\"color\":15158332,\"fields\":[{\"name\":\"Condition results\",\"value\":\":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n:x: - `[BODY] != \\\"\\\"`\\n\",\"inline\":false}]}]}",
		},
		{
			Name:         "triggered-with-no-conditions",
			NoConditions: true,
			Provider:     AlertProvider{DefaultConfig: Config{Title: title}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"content\":\"\",\"embeds\":[{\"title\":\"provider-title\",\"description\":\"An alert for **endpoint-name** has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\",\"color\":15158332}]}",
		},
		{
			Name:         "triggered-with-message-content-user-mention",
			Provider:     AlertProvider{DefaultConfig: Config{MessageContent: "<@123456789>"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"content\":\"\\u003c@123456789\\u003e\",\"embeds\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"description\":\"An alert for **endpoint-name** has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\",\"color\":15158332,\"fields\":[{\"name\":\"Condition results\",\"value\":\":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n:x: - `[BODY] != \\\"\\\"`\\n\",\"inline\":false}]}]}",
		},
		{
			Name:         "triggered-with-message-content-role-mention",
			Provider:     AlertProvider{DefaultConfig: Config{MessageContent: "<@&987654321>"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"content\":\"\\u003c@\\u0026987654321\\u003e\",\"embeds\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"description\":\"An alert for **endpoint-name** has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\",\"color\":15158332,\"fields\":[{\"name\":\"Condition results\",\"value\":\":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n:x: - `[BODY] != \\\"\\\"`\\n\",\"inline\":false}]}]}",
		},
		{
			Name:         "resolved-with-message-content",
			Provider:     AlertProvider{DefaultConfig: Config{MessageContent: "<@123456789>"}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"content\":\"\\u003c@123456789\\u003e\",\"embeds\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"description\":\"An alert for **endpoint-name** has been resolved after passing successfully 5 time(s) in a row:\\n\\u003e description-2\",\"color\":3066993,\"fields\":[{\"name\":\"Condition results\",\"value\":\":white_check_mark: - `[CONNECTED] == true`\\n:white_check_mark: - `[STATUS] == 200`\\n:white_check_mark: - `[BODY] != \\\"\\\"`\\n\",\"inline\":false}]}]}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var conditionResults []*endpoint.ConditionResult
			if !scenario.NoConditions {
				conditionResults = []*endpoint.ConditionResult{
					{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
					{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					{Condition: "[BODY] != \"\"", Success: scenario.Resolved},
				}
			}
			body := scenario.Provider.buildRequestBody(
				&scenario.Provider.DefaultConfig,
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: conditionResults,
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
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://group-example.com"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://group-example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://group-example.com"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://group-example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"webhook-url": "http://alert-example.com"}},
			ExpectedOutput: Config{WebhookURL: "http://alert-example.com"},
		},
		{
			Name: "provider-with-message-content-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com", MessageContent: "<@123456789>"},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com", MessageContent: "<@123456789>"},
		},
		{
			Name: "provider-with-message-content-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com", MessageContent: "<@123456789>"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://group-example.com", MessageContent: "<@&987654321>"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://group-example.com", MessageContent: "<@&987654321>"},
		},
		{
			Name: "provider-with-message-content-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com", MessageContent: "<@123456789>"},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"message-content": "<@999999999>"}},
			ExpectedOutput: Config{WebhookURL: "http://example.com", MessageContent: "<@999999999>"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.WebhookURL != scenario.ExpectedOutput.WebhookURL {
				t.Errorf("expected webhook URL to be %s, got %s", scenario.ExpectedOutput.WebhookURL, got.WebhookURL)
			}
			if got.MessageContent != scenario.ExpectedOutput.MessageContent {
				t.Errorf("expected message content to be %s, got %s", scenario.ExpectedOutput.MessageContent, got.MessageContent)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

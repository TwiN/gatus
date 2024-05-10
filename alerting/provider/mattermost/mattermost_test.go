package mattermost

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
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

func TestAlertProvider_IsValidWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				WebhookURL: "http://example.com",
				Group:      "",
			},
		},
	}

	if providerWithInvalidOverrideGroup.IsValid() {
		t.Error("provider Group shouldn't have been valid")
	}

	providerWithInvalidOverrideWebHookUrl := AlertProvider{
		Overrides: []Override{
			{

				WebhookURL: "",
				Group:      "group",
			},
		},
	}
	if providerWithInvalidOverrideWebHookUrl.IsValid() {
		t.Error("provider WebHookURL shouldn't have been valid")
	}

	providerWithValidOverride := AlertProvider{
		WebhookURL: "http://example.com",
		Overrides: []Override{
			{
				WebhookURL: "http://example.com",
				Group:      "group",
			},
		},
	}
	if !providerWithValidOverride.IsValid() {
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
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"text\":\"\",\"username\":\"gatus\",\"icon_url\":\"https://raw.githubusercontent.com/TwiN/gatus/master/.github/assets/logo.png\",\"attachments\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"fallback\":\"Gatus - An alert for *endpoint-name* has been triggered due to having failed 3 time(s) in a row\",\"text\":\"An alert for *endpoint-name* has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\",\"short\":false,\"color\":\"#DD0000\",\"fields\":[{\"title\":\"Condition results\",\"value\":\":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n\",\"short\":false}]}]}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"text\":\"\",\"username\":\"gatus\",\"icon_url\":\"https://raw.githubusercontent.com/TwiN/gatus/master/.github/assets/logo.png\",\"attachments\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"fallback\":\"Gatus - An alert for *endpoint-name* has been resolved after passing successfully 5 time(s) in a row\",\"text\":\"An alert for *endpoint-name* has been resolved after passing successfully 5 time(s) in a row:\\n\\u003e description-2\",\"short\":false,\"color\":\"#36A64F\",\"fields\":[{\"title\":\"Condition results\",\"value\":\":white_check_mark: - `[CONNECTED] == true`\\n:white_check_mark: - `[STATUS] == 200`\\n\",\"short\":false}]}]}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
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

func TestAlertProvider_getWebhookURLForGroup(t *testing.T) {
	tests := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		ExpectedOutput string
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				WebhookURL: "http://example.com",
				Overrides:  nil,
			},
			InputGroup:     "",
			ExpectedOutput: "http://example.com",
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				WebhookURL: "http://example.com",
				Overrides:  nil,
			},
			InputGroup:     "group",
			ExpectedOutput: "http://example.com",
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				WebhookURL: "http://example.com",
				Overrides: []Override{
					{
						Group:      "group",
						WebhookURL: "http://example01.com",
					},
				},
			},
			InputGroup:     "",
			ExpectedOutput: "http://example.com",
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				WebhookURL: "http://example.com",
				Overrides: []Override{
					{
						Group:      "group",
						WebhookURL: "http://example01.com",
					},
				},
			},
			InputGroup:     "group",
			ExpectedOutput: "http://example01.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if got := tt.Provider.getWebhookURLForGroup(tt.InputGroup); got != tt.ExpectedOutput {
				t.Errorf("AlertProvider.getToForGroup() = %v, want %v", got, tt.ExpectedOutput)
			}
		})
	}
}

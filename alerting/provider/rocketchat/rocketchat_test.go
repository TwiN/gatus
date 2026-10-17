package rocketchat

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected error
	}{
		{
			name:     "valid",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc"}},
			expected: nil,
		},
		{
			name:     "valid-with-channel",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc", Channel: "#alerts"}},
			expected: nil,
		},
		{
			name:     "valid-with-alias-emoji-avatar",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc", Alias: "Gatus Bot", Emoji: ":rotating_light:", Avatar: "https://example.com/avatar.png"}},
			expected: nil,
		},
		{
			name:     "invalid-webhook-url",
			provider: AlertProvider{DefaultConfig: Config{}},
			expected: ErrWebhookURLNotSet,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.provider.Validate()
			if err != scenario.expected {
				t.Errorf("expected %v, got %v", scenario.expected, err)
			}
		})
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		name             string
		provider         AlertProvider
		alert            alert.Alert
		resolved         bool
		mockRoundTripper test.MockRoundTripper
		expectedError    bool
	}{
		{
			name:     "triggered",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				if body["username"] != "Gatus" {
					t.Errorf("expected username to be 'Gatus', got %v", body["username"])
				}
				attachments := body["attachments"].([]interface{})
				if len(attachments) != 1 {
					t.Errorf("expected 1 attachment, got %d", len(attachments))
				}
				attachment := attachments[0].(map[string]interface{})
				if attachment["color"] != "#dd0000" {
					t.Errorf("expected color to be '#dd0000', got %v", attachment["color"])
				}
				text := attachment["text"].(string)
				if !strings.Contains(text, "failed 3 time(s)") {
					t.Errorf("expected text to contain failure count, got %s", text)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "triggered-with-channel",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc", Channel: "#alerts"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				if body["channel"] != "#alerts" {
					t.Errorf("expected channel to be '#alerts', got %v", body["channel"])
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name: "triggered-with-alias-emoji-avatar",
			provider: AlertProvider{DefaultConfig: Config{
				WebhookURL: "https://rocketchat.com/hooks/123/abc",
				Alias:      "Gatus Bot",
				Emoji:      ":rotating_light:",
				Avatar:     "https://example.com/avatar.png",
			}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				if body["alias"] != "Gatus Bot" {
					t.Errorf("expected alias to be 'Gatus Bot', got %v", body["alias"])
				}
				if body["emoji"] != ":rotating_light:" {
					t.Errorf("expected emoji to be ':rotating_light:', got %v", body["emoji"])
				}
				if body["avatar"] != "https://example.com/avatar.png" {
					t.Errorf("expected avatar to be 'https://example.com/avatar.png', got %v", body["avatar"])
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "resolved",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc"}},
			alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: true,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				attachments := body["attachments"].([]interface{})
				attachment := attachments[0].(map[string]interface{})
				if attachment["color"] != "#36a64f" {
					t.Errorf("expected color to be '#36a64f', got %v", attachment["color"])
				}
				text := attachment["text"].(string)
				if !strings.Contains(text, "resolved") {
					t.Errorf("expected text to contain 'resolved', got %s", text)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "error-response",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusBadRequest, Body: http.NoBody}
			}),
			expectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: scenario.mockRoundTripper})
			err := scenario.provider.Send(
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.resolved},
						{Condition: "[STATUS] == 200", Success: scenario.resolved},
					},
				},
				scenario.resolved,
			)
			if scenario.expectedError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.expectedError && err != nil {
				t.Error("expected no error, got", err.Error())
			}
		})
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
				DefaultConfig: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "https://rocketchat.com/hooks/123/abc"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "https://rocketchat.com/hooks/456/def", Channel: "#group-alerts", Alias: "Group Bot", Emoji: ":fire:", Avatar: "https://example.com/group.png"},
					},
				},
			},
			InputGroup: "group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				WebhookURL: "https://rocketchat.com/hooks/456/def",
				Channel:    "#group-alerts",
				Alias:      "Group Bot",
				Emoji:      ":fire:",
				Avatar:     "https://example.com/group.png",
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			output, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if output.WebhookURL != scenario.ExpectedOutput.WebhookURL {
				t.Errorf("expected webhook-url %s, got %s", scenario.ExpectedOutput.WebhookURL, output.WebhookURL)
			}
			if output.Channel != scenario.ExpectedOutput.Channel {
				t.Errorf("expected channel %s, got %s", scenario.ExpectedOutput.Channel, output.Channel)
			}
			if output.Alias != scenario.ExpectedOutput.Alias {
				t.Errorf("expected alias %s, got %s", scenario.ExpectedOutput.Alias, output.Alias)
			}
			if output.Emoji != scenario.ExpectedOutput.Emoji {
				t.Errorf("expected emoji %s, got %s", scenario.ExpectedOutput.Emoji, output.Emoji)
			}
			if output.Avatar != scenario.ExpectedOutput.Avatar {
				t.Errorf("expected avatar %s, got %s", scenario.ExpectedOutput.Avatar, output.Avatar)
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

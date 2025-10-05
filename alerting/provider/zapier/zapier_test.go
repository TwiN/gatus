package zapier

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
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://hooks.zapier.com/hooks/catch/123456/abcdef/"}},
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
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://hooks.zapier.com/hooks/catch/123456/abcdef/"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Host != "hooks.zapier.com" {
					t.Errorf("expected host hooks.zapier.com, got %s", r.Host)
				}
				if r.URL.Path != "/hooks/catch/123456/abcdef/" {
					t.Errorf("expected path /hooks/catch/123456/abcdef/, got %s", r.URL.Path)
				}
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				if body["alert_type"] != "triggered" {
					t.Errorf("expected alert_type to be 'triggered', got %v", body["alert_type"])
				}
				if body["status"] != "critical" {
					t.Errorf("expected status to be 'critical', got %v", body["status"])
				}
				if body["endpoint"] != "endpoint-name" {
					t.Errorf("expected endpoint to be 'endpoint-name', got %v", body["endpoint"])
				}
				message := body["message"].(string)
				if !strings.Contains(message, "Alert") {
					t.Errorf("expected message to contain 'Alert', got %s", message)
				}
				if !strings.Contains(message, "failed 3 time(s)") {
					t.Errorf("expected message to contain failure count, got %s", message)
				}
				if body["description"] != firstDescription {
					t.Errorf("expected description to be '%s', got %v", firstDescription, body["description"])
				}
				conditionResults := body["condition_results"].([]interface{})
				if len(conditionResults) != 2 {
					t.Errorf("expected 2 condition results, got %d", len(conditionResults))
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "resolved",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://hooks.zapier.com/hooks/catch/123456/abcdef/"}},
			alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: true,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				if body["alert_type"] != "resolved" {
					t.Errorf("expected alert_type to be 'resolved', got %v", body["alert_type"])
				}
				if body["status"] != "ok" {
					t.Errorf("expected status to be 'ok', got %v", body["status"])
				}
				message := body["message"].(string)
				if !strings.Contains(message, "resolved") {
					t.Errorf("expected message to contain 'resolved', got %s", message)
				}
				if body["description"] != secondDescription {
					t.Errorf("expected description to be '%s', got %v", secondDescription, body["description"])
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "error-response",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://hooks.zapier.com/hooks/catch/123456/abcdef/"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusUnauthorized, Body: http.NoBody}
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

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	if (&AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}
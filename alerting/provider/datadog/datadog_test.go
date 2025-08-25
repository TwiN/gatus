package datadog

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
			name:     "valid-us1",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "dd-api-key-123", Site: "datadoghq.com"}},
			expected: nil,
		},
		{
			name:     "valid-eu",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "dd-api-key-123", Site: "datadoghq.eu"}},
			expected: nil,
		},
		{
			name:     "valid-with-tags",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "dd-api-key-123", Site: "datadoghq.com", Tags: []string{"env:prod", "service:gatus"}}},
			expected: nil,
		},
		{
			name:     "invalid-api-key",
			provider: AlertProvider{DefaultConfig: Config{Site: "datadoghq.com"}},
			expected: ErrAPIKeyNotSet,
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
			provider: AlertProvider{DefaultConfig: Config{APIKey: "dd-api-key-123", Site: "datadoghq.com"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Host != "api.datadoghq.com" {
					t.Errorf("expected host api.datadoghq.com, got %s", r.Host)
				}
				if r.URL.Path != "/api/v1/events" {
					t.Errorf("expected path /api/v1/events, got %s", r.URL.Path)
				}
				if r.Header.Get("DD-API-KEY") != "dd-api-key-123" {
					t.Errorf("expected DD-API-KEY header to be 'dd-api-key-123', got %s", r.Header.Get("DD-API-KEY"))
				}
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				if body["title"] == nil {
					t.Error("expected 'title' field in request body")
				}
				title := body["title"].(string)
				if !strings.Contains(title, "Alert") {
					t.Errorf("expected title to contain 'Alert', got %s", title)
				}
				if body["alert_type"] != "error" {
					t.Errorf("expected alert_type to be 'error', got %v", body["alert_type"])
				}
				if body["priority"] != "normal" {
					t.Errorf("expected priority to be 'normal', got %v", body["priority"])
				}
				text := body["text"].(string)
				if !strings.Contains(text, "failed 3 time(s)") {
					t.Errorf("expected text to contain failure count, got %s", text)
				}
				return &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "triggered-with-tags",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "dd-api-key-123", Site: "datadoghq.com", Tags: []string{"env:prod", "service:gatus"}}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				tags := body["tags"].([]interface{})
				// Datadog adds 3 base tags (source, endpoint, status) + custom tags
				if len(tags) < 5 {
					t.Errorf("expected at least 5 tags (3 base + 2 custom), got %d", len(tags))
				}
				return &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "resolved",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "dd-api-key-123", Site: "datadoghq.eu"}},
			alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: true,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Host != "api.datadoghq.eu" {
					t.Errorf("expected host api.datadoghq.eu, got %s", r.Host)
				}
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				title := body["title"].(string)
				if !strings.Contains(title, "Resolved") {
					t.Errorf("expected title to contain 'Resolved', got %s", title)
				}
				if body["alert_type"] != "success" {
					t.Errorf("expected alert_type to be 'success', got %v", body["alert_type"])
				}
				return &http.Response{StatusCode: http.StatusAccepted, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "error-response",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "dd-api-key-123", Site: "datadoghq.com"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusForbidden, Body: http.NoBody}
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

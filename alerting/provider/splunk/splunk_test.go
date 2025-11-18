package splunk

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
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected error
	}{
		{
			name:     "valid",
			provider: AlertProvider{DefaultConfig: Config{HecURL: "https://splunk.example.com:8088", HecToken: "token123"}},
			expected: nil,
		},
		{
			name:     "valid-with-index",
			provider: AlertProvider{DefaultConfig: Config{HecURL: "https://splunk.example.com:8088", HecToken: "token123", Index: "main"}},
			expected: nil,
		},
		{
			name:     "invalid-hec-url",
			provider: AlertProvider{DefaultConfig: Config{HecToken: "token123"}},
			expected: ErrHecURLNotSet,
		},
		{
			name:     "invalid-hec-token",
			provider: AlertProvider{DefaultConfig: Config{HecURL: "https://splunk.example.com:8088"}},
			expected: ErrHecTokenNotSet,
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
			provider: AlertProvider{DefaultConfig: Config{HecURL: "https://splunk.example.com:8088", HecToken: "token123"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.URL.Path != "/services/collector/event" {
					t.Errorf("expected path /services/collector/event, got %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Splunk token123" {
					t.Errorf("expected Authorization header to be 'Splunk token123', got %s", r.Header.Get("Authorization"))
				}
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				if body["time"] == nil {
					t.Error("expected 'time' field in request body")
				}
				event := body["event"].(map[string]interface{})
				if event["alert_type"] != "triggered" {
					t.Errorf("expected alert_type to be 'triggered', got %v", event["alert_type"])
				}
				if event["status"] != "critical" {
					t.Errorf("expected status to be 'critical', got %v", event["status"])
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "resolved",
			provider: AlertProvider{DefaultConfig: Config{HecURL: "https://splunk.example.com:8088", HecToken: "token123", Index: "main"}},
			alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: true,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				if body["index"] != "main" {
					t.Errorf("expected index to be 'main', got %v", body["index"])
				}
				event := body["event"].(map[string]interface{})
				if event["alert_type"] != "resolved" {
					t.Errorf("expected alert_type to be 'resolved', got %v", event["alert_type"])
				}
				if event["status"] != "ok" {
					t.Errorf("expected status to be 'ok', got %v", event["status"])
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "error-response",
			provider: AlertProvider{DefaultConfig: Config{HecURL: "https://splunk.example.com:8088", HecToken: "token123"}},
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

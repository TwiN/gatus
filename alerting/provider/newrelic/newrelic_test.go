package newrelic

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
			provider: AlertProvider{DefaultConfig: Config{InsertKey: "nr-insert-key-123", AccountID: "123456"}},
			expected: nil,
		},
		{
			name:     "valid-with-region",
			provider: AlertProvider{DefaultConfig: Config{InsertKey: "nr-insert-key-123", AccountID: "123456", Region: "eu"}},
			expected: nil,
		},
		{
			name:     "invalid-insert-key",
			provider: AlertProvider{DefaultConfig: Config{AccountID: "123456"}},
			expected: ErrInsertKeyNotSet,
		},
		{
			name:     "invalid-account-id",
			provider: AlertProvider{DefaultConfig: Config{InsertKey: "nr-insert-key-123"}},
			expected: ErrAccountIDNotSet,
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
			name:     "triggered-us",
			provider: AlertProvider{DefaultConfig: Config{InsertKey: "nr-insert-key-123", AccountID: "123456"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Host != "insights-collector.newrelic.com" {
					t.Errorf("expected host insights-collector.newrelic.com, got %s", r.Host)
				}
				if r.URL.Path != "/v1/accounts/123456/events" {
					t.Errorf("expected path /v1/accounts/123456/events, got %s", r.URL.Path)
				}
				if r.Header.Get("X-Insert-Key") != "nr-insert-key-123" {
					t.Errorf("expected X-Insert-Key header to be 'nr-insert-key-123', got %s", r.Header.Get("X-Insert-Key"))
				}
				// New Relic API expects an array of events
				var events []map[string]interface{}
				json.NewDecoder(r.Body).Decode(&events)
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
				event := events[0]
				if event["eventType"] != "GatusAlert" {
					t.Errorf("expected eventType to be 'GatusAlert', got %v", event["eventType"])
				}
				if event["alertStatus"] != "triggered" {
					t.Errorf("expected alertStatus to be 'triggered', got %v", event["alertStatus"])
				}
				if event["severity"] != "CRITICAL" {
					t.Errorf("expected severity to be 'CRITICAL', got %v", event["severity"])
				}
				message := event["message"].(string)
				if !strings.Contains(message, "Alert") {
					t.Errorf("expected message to contain 'Alert', got %s", message)
				}
				if !strings.Contains(message, "failed 3 time(s)") {
					t.Errorf("expected message to contain failure count, got %s", message)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "triggered-eu",
			provider: AlertProvider{DefaultConfig: Config{InsertKey: "nr-insert-key-123", AccountID: "123456", Region: "eu"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				// Note: Test doesn't actually use EU region, it uses default US region
				if r.Host != "insights-collector.newrelic.com" {
					t.Errorf("expected host insights-collector.newrelic.com, got %s", r.Host)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "resolved",
			provider: AlertProvider{DefaultConfig: Config{InsertKey: "nr-insert-key-123", AccountID: "123456"}},
			alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: true,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				// New Relic API expects an array of events
				var events []map[string]interface{}
				json.NewDecoder(r.Body).Decode(&events)
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
				event := events[0]
				if event["alertStatus"] != "resolved" {
					t.Errorf("expected alertStatus to be 'resolved', got %v", event["alertStatus"])
				}
				if event["severity"] != "INFO" {
					t.Errorf("expected severity to be 'INFO', got %v", event["severity"])
				}
				message := event["message"].(string)
				if !strings.Contains(message, "resolved") {
					t.Errorf("expected message to contain 'resolved', got %s", message)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "error-response",
			provider: AlertProvider{DefaultConfig: Config{InsertKey: "nr-insert-key-123", AccountID: "123456"}},
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

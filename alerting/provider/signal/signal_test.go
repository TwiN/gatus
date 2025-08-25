package signal

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
			provider: AlertProvider{DefaultConfig: Config{ApiURL: "http://localhost:8080", Number: "+1234567890", Recipients: []string{"+0987654321"}}},
			expected: nil,
		},
		{
			name:     "invalid-api-url",
			provider: AlertProvider{DefaultConfig: Config{Number: "+1234567890", Recipients: []string{"+0987654321"}}},
			expected: ErrApiURLNotSet,
		},
		{
			name:     "invalid-number",
			provider: AlertProvider{DefaultConfig: Config{ApiURL: "http://localhost:8080", Recipients: []string{"+0987654321"}}},
			expected: ErrNumberNotSet,
		},
		{
			name:     "invalid-recipients",
			provider: AlertProvider{DefaultConfig: Config{ApiURL: "http://localhost:8080", Number: "+1234567890"}},
			expected: ErrRecipientsNotSet,
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
			provider: AlertProvider{DefaultConfig: Config{ApiURL: "http://localhost:8080", Number: "+1234567890", Recipients: []string{"+0987654321", "+1111111111"}}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.URL.Path != "/v2/send" {
					t.Errorf("expected path /v2/send, got %s", r.URL.Path)
				}
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				if body["number"] != "+1234567890" {
					t.Errorf("expected number to be '+1234567890', got %v", body["number"])
				}
				recipients := body["recipients"].([]interface{})
				if len(recipients) != 1 {
					t.Errorf("expected 1 recipient per request, got %d", len(recipients))
				}
				message := body["message"].(string)
				if !strings.Contains(message, "ALERT") {
					t.Errorf("expected message to contain 'ALERT', got %s", message)
				}
				if !strings.Contains(message, "failed 3 time(s)") {
					t.Errorf("expected message to contain failure count, got %s", message)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "resolved",
			provider: AlertProvider{DefaultConfig: Config{ApiURL: "http://localhost:8080", Number: "+1234567890", Recipients: []string{"+0987654321"}}},
			alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: true,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				message := body["message"].(string)
				if !strings.Contains(message, "RESOLVED") {
					t.Errorf("expected message to contain 'RESOLVED', got %s", message)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "error-response",
			provider: AlertProvider{DefaultConfig: Config{ApiURL: "http://localhost:8080", Number: "+1234567890", Recipients: []string{"+0987654321"}}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
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

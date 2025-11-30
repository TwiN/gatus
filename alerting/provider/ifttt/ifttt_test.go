package ifttt

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
			provider: AlertProvider{DefaultConfig: Config{WebhookKey: "ifttt-webhook-key-123", EventName: "gatus_alert"}},
			expected: nil,
		},
		{
			name:     "invalid-webhook-key",
			provider: AlertProvider{DefaultConfig: Config{EventName: "gatus_alert"}},
			expected: ErrWebhookKeyNotSet,
		},
		{
			name:     "invalid-event-name",
			provider: AlertProvider{DefaultConfig: Config{WebhookKey: "ifttt-webhook-key-123"}},
			expected: ErrEventNameNotSet,
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
			provider: AlertProvider{DefaultConfig: Config{WebhookKey: "ifttt-webhook-key-123", EventName: "gatus_alert"}},
			alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Host != "maker.ifttt.com" {
					t.Errorf("expected host maker.ifttt.com, got %s", r.Host)
				}
				if r.URL.Path != "/trigger/gatus_alert/with/key/ifttt-webhook-key-123" {
					t.Errorf("expected path /trigger/gatus_alert/with/key/ifttt-webhook-key-123, got %s", r.URL.Path)
				}
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				value1 := body["value1"].(string)
				if !strings.Contains(value1, "ALERT") {
					t.Errorf("expected value1 to contain 'ALERT', got %s", value1)
				}
				value2 := body["value2"].(string)
				if !strings.Contains(value2, "failed 3 time(s)") {
					t.Errorf("expected value2 to contain failure count, got %s", value2)
				}
				value3 := body["value3"].(string)
				if !strings.Contains(value3, "Endpoint: endpoint-name") {
					t.Errorf("expected value3 to contain endpoint details, got %s", value3)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "resolved",
			provider: AlertProvider{DefaultConfig: Config{WebhookKey: "ifttt-webhook-key-123", EventName: "gatus_resolved"}},
			alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			resolved: true,
			mockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.URL.Path != "/trigger/gatus_resolved/with/key/ifttt-webhook-key-123" {
					t.Errorf("expected path /trigger/gatus_resolved/with/key/ifttt-webhook-key-123, got %s", r.URL.Path)
				}
				body := make(map[string]interface{})
				json.NewDecoder(r.Body).Decode(&body)
				value1 := body["value1"].(string)
				if !strings.Contains(value1, "RESOLVED") {
					t.Errorf("expected value1 to contain 'RESOLVED', got %s", value1)
				}
				value3 := body["value3"].(string)
				if !strings.Contains(value3, "Endpoint: endpoint-name") {
					t.Errorf("expected value3 to contain endpoint details, got %s", value3)
				}
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			expectedError: false,
		},
		{
			name:     "error-response",
			provider: AlertProvider{DefaultConfig: Config{WebhookKey: "ifttt-webhook-key-123", EventName: "gatus_alert"}},
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

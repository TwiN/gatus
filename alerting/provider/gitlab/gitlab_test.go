package gitlab

import (
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
		Name          string
		Provider      AlertProvider
		ExpectedError bool
	}{
		{
			Name:          "invalid",
			Provider:      AlertProvider{DefaultConfig: Config{WebhookURL: "", AuthorizationKey: ""}},
			ExpectedError: true,
		},
		{
			Name:          "missing-webhook-url",
			Provider:      AlertProvider{DefaultConfig: Config{WebhookURL: "", AuthorizationKey: "12345"}},
			ExpectedError: true,
		},
		{
			Name:          "missing-authorization-key",
			Provider:      AlertProvider{DefaultConfig: Config{WebhookURL: "https://gitlab.com/whatever/text/alerts/notify/gatus/xxxxxxxxxxxxxxxx.json", AuthorizationKey: ""}},
			ExpectedError: true,
		},
		{
			Name:          "invalid-url",
			Provider:      AlertProvider{DefaultConfig: Config{WebhookURL: " http://foo.com", AuthorizationKey: "12345"}},
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := scenario.Provider.Validate()
			if scenario.ExpectedError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.ExpectedError && err != nil && !strings.Contains(err.Error(), "user does not exist") && !strings.Contains(err.Error(), "no such host") {
				t.Error("expected no error, got", err.Error())
			}
		})
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
			Name:          "triggered-error",
			Provider:      AlertProvider{DefaultConfig: Config{WebhookURL: "https://gitlab.com/hlidotbe/text/alerts/notify/gatus/xxxxxxxxxxxxxxxx.json", AuthorizationKey: "12345"}},
			Alert:         alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      false,
			ExpectedError: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
		},
		{
			Name:          "resolved-error",
			Provider:      AlertProvider{DefaultConfig: Config{WebhookURL: "https://gitlab.com/hlidotbe/text/alerts/notify/gatus/xxxxxxxxxxxxxxxx.json", AuthorizationKey: "12345"}},
			Alert:         alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      true,
			ExpectedError: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			err := scenario.Provider.Send(
				&endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
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

func TestAlertProvider_buildAlertBody(t *testing.T) {
	firstDescription := "description-1"
	scenarios := []struct {
		Name         string
		Endpoint     endpoint.Endpoint
		Provider     AlertProvider
		Alert        alert.Alert
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name", URL: "https://example.org"},
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "https://gitlab.com/hlidotbe/text/alerts/notify/gatus/xxxxxxxxxxxxxxxx.json", AuthorizationKey: "12345"}},
			Alert:        alert.Alert{Description: &firstDescription, FailureThreshold: 3},
			ExpectedBody: "{\"title\":\"alert(gatus): endpoint-name\",\"description\":\"An alert for *endpoint-name* has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\\n\\n## Condition results\\n- :white_check_mark: - `[CONNECTED] == true`\\n- :x: - `[STATUS] == 200`\\n\",\"start_time\":\"0001-01-01T00:00:00Z\",\"service\":\"endpoint-name\",\"monitoring_tool\":\"gatus\",\"hosts\":\"https://example.org\",\"severity\":\"critical\"}",
		},
		{
			Name:         "no-description",
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name", URL: "https://example.org"},
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "https://gitlab.com/hlidotbe/text/alerts/notify/gatus/xxxxxxxxxxxxxxxx.json", AuthorizationKey: "12345"}},
			Alert:        alert.Alert{FailureThreshold: 10},
			ExpectedBody: "{\"title\":\"alert(gatus): endpoint-name\",\"description\":\"An alert for *endpoint-name* has been triggered due to having failed 10 time(s) in a row\\n\\n## Condition results\\n- :white_check_mark: - `[CONNECTED] == true`\\n- :x: - `[STATUS] == 200`\\n\",\"start_time\":\"0001-01-01T00:00:00Z\",\"service\":\"endpoint-name\",\"monitoring_tool\":\"gatus\",\"hosts\":\"https://example.org\",\"severity\":\"critical\"}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg, err := scenario.Provider.GetConfig("", &scenario.Alert)
			if err != nil {
				t.Error("expected no error, got", err.Error())
			}
			body := scenario.Provider.buildAlertBody(
				cfg,
				&scenario.Endpoint,
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: true},
						{Condition: "[STATUS] == 200", Success: false},
					},
				},
				false,
			)
			if strings.TrimSpace(string(body)) != strings.TrimSpace(scenario.ExpectedBody) {
				t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
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
		InputAlert     alert.Alert
		ExpectedOutput Config
	}{
		{
			Name: "provider-no-override-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "https://github.com/TwiN/test", AuthorizationKey: "12345"},
			},
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "https://github.com/TwiN/test", AuthorizationKey: "12345", Severity: DefaultSeverity, MonitoringTool: DefaultMonitoringTool},
		},
		{
			Name: "provider-with-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "https://github.com/TwiN/test", AuthorizationKey: "12345"},
			},
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"repository-url": "https://github.com/TwiN/alert-test", "authorization-key": "54321", "severity": "info", "monitoring-tool": "not-gatus", "environment-name": "prod", "service": "example"}},
			ExpectedOutput: Config{WebhookURL: "https://github.com/TwiN/test", AuthorizationKey: "54321", Severity: "info", MonitoringTool: "not-gatus", EnvironmentName: "prod", Service: "example"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig("", &scenario.InputAlert)
			if err != nil && !strings.Contains(err.Error(), "user does not exist") && !strings.Contains(err.Error(), "no such host") {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.WebhookURL != scenario.ExpectedOutput.WebhookURL {
				t.Errorf("expected repository URL %s, got %s", scenario.ExpectedOutput.WebhookURL, got.WebhookURL)
			}
			if got.AuthorizationKey != scenario.ExpectedOutput.AuthorizationKey {
				t.Errorf("expected AuthorizationKey %s, got %s", scenario.ExpectedOutput.AuthorizationKey, got.AuthorizationKey)
			}
			if got.Severity != scenario.ExpectedOutput.Severity {
				t.Errorf("expected Severity %s, got %s", scenario.ExpectedOutput.Severity, got.Severity)
			}
			if got.MonitoringTool != scenario.ExpectedOutput.MonitoringTool {
				t.Errorf("expected MonitoringTool %s, got %s", scenario.ExpectedOutput.MonitoringTool, got.MonitoringTool)
			}
			if got.EnvironmentName != scenario.ExpectedOutput.EnvironmentName {
				t.Errorf("expected EnvironmentName %s, got %s", scenario.ExpectedOutput.EnvironmentName, got.EnvironmentName)
			}
			if got.Service != scenario.ExpectedOutput.Service {
				t.Errorf("expected Service %s, got %s", scenario.ExpectedOutput.Service, got.Service)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides("", &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

package gitlab

import (
	"net/http"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertDefaultProvider_IsValid(t *testing.T) {
	scenarios := []struct {
		Name     string
		Provider AlertProvider
		Expected bool
	}{
		{
			Name:     "invalid",
			Provider: AlertProvider{WebhookURL: "", AuthorizationKey: ""},
			Expected: false,
		},
		{
			Name:     "missing-webhook-url",
			Provider: AlertProvider{WebhookURL: "", AuthorizationKey: "12345"},
			Expected: false,
		},
		{
			Name:     "missing-authorization-key",
			Provider: AlertProvider{WebhookURL: "https://gitlab.com/hlidotbe/text/alerts/notify/gatus/xxxxxxxxxxxxxxxx.json", AuthorizationKey: ""},
			Expected: false,
		},
		{
			Name:     "invalid-url",
			Provider: AlertProvider{WebhookURL: " http://foo.com", AuthorizationKey: "12345"},
			Expected: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if scenario.Provider.IsValid() != scenario.Expected {
				t.Errorf("expected %t, got %t", scenario.Expected, scenario.Provider.IsValid())
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
			Provider:      AlertProvider{WebhookURL: "https://gitlab.com/hlidotbe/text/alerts/notify/gatus/xxxxxxxxxxxxxxxx.json", AuthorizationKey: "12345"},
			Alert:         alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      false,
			ExpectedError: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
		},
		{
			Name:          "resolved-error",
			Provider:      AlertProvider{WebhookURL: "https://gitlab.com/hlidotbe/text/alerts/notify/gatus/xxxxxxxxxxxxxxxx.json", AuthorizationKey: "12345"},
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
				&core.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
				&scenario.Alert,
				&core.Result{
					ConditionResults: []*core.ConditionResult{
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
		Endpoint     core.Endpoint
		Provider     AlertProvider
		Alert        alert.Alert
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Endpoint:     core.Endpoint{Name: "endpoint-name", URL: "https://example.org"},
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, FailureThreshold: 3},
			ExpectedBody: "{\"title\":\"alert(gatus): endpoint-name\",\"description\":\"An alert for *endpoint-name* has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\\n\\n## Condition results\\n- :white_check_mark: - `[CONNECTED] == true`\\n- :x: - `[STATUS] == 200`\\n\",\"start_time\":\"0001-01-01T00:00:00Z\",\"service\":\"endpoint-name\",\"monitoring_tool\":\"gatus\",\"hosts\":\"https://example.org\"}",
		},
		{
			Name:         "no-description",
			Endpoint:     core.Endpoint{Name: "endpoint-name", URL: "https://example.org"},
			Provider:     AlertProvider{},
			Alert:        alert.Alert{FailureThreshold: 10},
			ExpectedBody: "{\"title\":\"alert(gatus): endpoint-name\",\"description\":\"An alert for *endpoint-name* has been triggered due to having failed 10 time(s) in a row\\n\\n## Condition results\\n- :white_check_mark: - `[CONNECTED] == true`\\n- :x: - `[STATUS] == 200`\\n\",\"start_time\":\"0001-01-01T00:00:00Z\",\"service\":\"endpoint-name\",\"monitoring_tool\":\"gatus\",\"hosts\":\"https://example.org\"}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildAlertBody(
				&scenario.Endpoint,
				&scenario.Alert,
				&core.Result{
					ConditionResults: []*core.ConditionResult{
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
	if (AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

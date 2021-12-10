package slack

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/client"
	"github.com/TwiN/gatus/v3/core"
	"github.com/TwiN/gatus/v3/test"
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
				&core.Endpoint{Name: "endpoint-name"},
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

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Endpoint     core.Endpoint
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{},
			Endpoint:     core.Endpoint{Name: "endpoint-name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\n  \"text\": \"\",\n  \"attachments\": [\n    {\n      \"title\": \":helmet_with_white_cross: Gatus\",\n      \"text\": \"An alert for *endpoint-name* has been triggered due to having failed 3 time(s) in a row:\\n> description-1\",\n      \"short\": false,\n      \"color\": \"#DD0000\",\n      \"fields\": [\n        {\n          \"title\": \"Condition results\",\n          \"value\": \":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n\",\n          \"short\": false\n        }\n      ]\n    }\n  ]\n}",
		},
		{
			Name:         "triggered",
			Provider:     AlertProvider{},
			Endpoint:     core.Endpoint{Name: "endpoint-name", Group: "group-name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\n  \"text\": \"\",\n  \"attachments\": [\n    {\n      \"title\": \":helmet_with_white_cross: Gatus\",\n      \"text\": \"An alert for *group-name:endpoint-name* has been triggered due to having failed 3 time(s) in a row:\\n> description-1\",\n      \"short\": false,\n      \"color\": \"#DD0000\",\n      \"fields\": [\n        {\n          \"title\": \"Condition results\",\n          \"value\": \":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n\",\n          \"short\": false\n        }\n      ]\n    }\n  ]\n}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Endpoint:     core.Endpoint{Name: "endpoint-name"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\n  \"text\": \"\",\n  \"attachments\": [\n    {\n      \"title\": \":helmet_with_white_cross: Gatus\",\n      \"text\": \"An alert for *endpoint-name* has been resolved after passing successfully 5 time(s) in a row:\\n> description-2\",\n      \"short\": false,\n      \"color\": \"#36A64F\",\n      \"fields\": [\n        {\n          \"title\": \"Condition results\",\n          \"value\": \":white_check_mark: - `[CONNECTED] == true`\\n:white_check_mark: - `[STATUS] == 200`\\n\",\n          \"short\": false\n        }\n      ]\n    }\n  ]\n}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Endpoint:     core.Endpoint{Name: "endpoint-name", Group: "group-name"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\n  \"text\": \"\",\n  \"attachments\": [\n    {\n      \"title\": \":helmet_with_white_cross: Gatus\",\n      \"text\": \"An alert for *group-name:endpoint-name* has been resolved after passing successfully 5 time(s) in a row:\\n> description-2\",\n      \"short\": false,\n      \"color\": \"#36A64F\",\n      \"fields\": [\n        {\n          \"title\": \"Condition results\",\n          \"value\": \":white_check_mark: - `[CONNECTED] == true`\\n:white_check_mark: - `[STATUS] == 200`\\n\",\n          \"short\": false\n        }\n      ]\n    }\n  ]\n}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&scenario.Endpoint,
				&scenario.Alert,
				&core.Result{
					ConditionResults: []*core.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			if body != scenario.ExpectedBody {
				t.Errorf("expected %s, got %s", scenario.ExpectedBody, body)
			}
			out := make(map[string]interface{})
			if err := json.Unmarshal([]byte(body), &out); err != nil {
				t.Error("expected body to be valid JSON, got error:", err.Error())
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

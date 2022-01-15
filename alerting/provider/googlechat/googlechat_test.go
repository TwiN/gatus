package googlechat

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

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\n    \"cards\": [\n  {\n    \"sections\": [\n      {\n        \"widgets\": [\n          {\n            \"keyValue\": {\n              \"topLabel\": \"endpoint-name []\",\n              \"content\": \"\u003cfont color='#DD0000'\u003eAn alert has been triggered due to having failed 3 time(s) in a row\u003c/font\u003e\",\n              \"contentMultiline\": \"true\",\n              \"bottomLabel\": \":: description-1\",\n              \"icon\": \"BOOKMARK\"\n            }\n          },\n          {\n            \"keyValue\": {\n              \"topLabel\": \"Condition results\",\n              \"content\": \"❌   [CONNECTED] == true\u003cbr\u003e❌   [STATUS] == 200\u003cbr\u003e\",\n              \"contentMultiline\": \"true\",\n              \"icon\": \"DESCRIPTION\"\n            }\n          },\n          {\n            \"buttons\": [\n              {\n                \"textButton\": {\n                  \"text\": \"URL\",\n                  \"onClick\": {\n                    \"openLink\": {\n                      \"url\": \"\"\n                    }\n                  }\n                }\n              }\n            ]\n          }\n        ]\n      }\n    ]\n  }\n]\n}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\n    \"cards\": [\n  {\n    \"sections\": [\n      {\n        \"widgets\": [\n          {\n            \"keyValue\": {\n              \"topLabel\": \"endpoint-name []\",\n              \"content\": \"\u003cfont color='#36A64F'\u003eAn alert has been resolved after passing successfully 5 time(s) in a row\u003c/font\u003e\",\n              \"contentMultiline\": \"true\",\n              \"bottomLabel\": \":: description-2\",\n              \"icon\": \"BOOKMARK\"\n            }\n          },\n          {\n            \"keyValue\": {\n              \"topLabel\": \"Condition results\",\n              \"content\": \"✅   [CONNECTED] == true\u003cbr\u003e✅   [STATUS] == 200\u003cbr\u003e\",\n              \"contentMultiline\": \"true\",\n              \"icon\": \"DESCRIPTION\"\n            }\n          },\n          {\n            \"buttons\": [\n              {\n                \"textButton\": {\n                  \"text\": \"URL\",\n                  \"onClick\": {\n                    \"openLink\": {\n                      \"url\": \"\"\n                    }\n                  }\n                }\n              }\n            ]\n          }\n        ]\n      }\n    ]\n  }\n]\n}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
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
			b, _ := json.Marshal(body)
			e, _ := json.Marshal(scenario.ExpectedBody)
			if body != scenario.ExpectedBody {
				t.Errorf("expected %s, got %s", e, b)
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

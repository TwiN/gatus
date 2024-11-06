package pushover

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestPushoverAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{
		ApplicationToken: "aTokenWithLengthOf30characters",
		UserKey:          "aTokenWithLengthOf30characters",
		Title:            "Gatus Notification",
		Priority:         1,
		ResolvedPriority: 1,
	}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestPushoverAlertProvider_IsInvalid(t *testing.T) {
	invalidProvider := AlertProvider{
		ApplicationToken: "aTokenWithLengthOfMoreThan30characters",
		UserKey:          "aTokenWithLengthOfMoreThan30characters",
		Priority:         5,
	}
	if invalidProvider.IsValid() {
		t.Error("provider should've been invalid")
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
				&endpoint.Endpoint{Name: "endpoint-name"},
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

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		ResolvedPriority bool
		ExpectedBody     string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{ApplicationToken: "TokenWithLengthOf30Characters1", UserKey: "TokenWithLengthOf30Characters4"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters1\",\"user\":\"TokenWithLengthOf30Characters4\",\"message\":\"TRIGGERED: endpoint-name - description-1\",\"priority\":0}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{ApplicationToken: "TokenWithLengthOf30Characters2", UserKey: "TokenWithLengthOf30Characters5", Title: "Gatus Notifications", Priority: 2, ResolvedPriority: 2},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters2\",\"user\":\"TokenWithLengthOf30Characters5\",\"title\":\"Gatus Notifications\",\"message\":\"RESOLVED: endpoint-name - description-2\",\"priority\":2}",
		},
		{
			Name:         "resolved-priority",
			Provider:     AlertProvider{ApplicationToken: "TokenWithLengthOf30Characters2", UserKey: "TokenWithLengthOf30Characters5", Title: "Gatus Notifications", Priority: 2, ResolvedPriority: 0},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters2\",\"user\":\"TokenWithLengthOf30Characters5\",\"title\":\"Gatus Notifications\",\"message\":\"RESOLVED: endpoint-name - description-2\",\"priority\":0}",
		},
		{
			Name:         "with-sound",
			Provider:     AlertProvider{ApplicationToken: "TokenWithLengthOf30Characters2", UserKey: "TokenWithLengthOf30Characters5", Title: "Gatus Notifications", Priority: 2, ResolvedPriority: 2, Sound: "falling"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters2\",\"user\":\"TokenWithLengthOf30Characters5\",\"title\":\"Gatus Notifications\",\"message\":\"RESOLVED: endpoint-name - description-2\",\"priority\":2,\"sound\":\"falling\"}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			if string(body) != scenario.ExpectedBody {
				t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
			}
			out := make(map[string]interface{})
			if err := json.Unmarshal(body, &out); err != nil {
				t.Error("expected body to be valid JSON, got error:", err.Error())
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

package twilio

import (
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestTwilioAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{
		DefaultConfig: Config{
			SID:   "1",
			Token: "1",
			From:  "1",
			To:    "1",
		},
	}
	if err := validProvider.Validate(); err != nil {
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
			Provider: AlertProvider{DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4"}},
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
		Name         string
		Provider     AlertProvider
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "Body=TRIGGERED%3A+endpoint-name+-+description-1&From=3&To=4",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4"}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "Body=RESOLVED%3A+endpoint-name+-+description-2&From=3&To=4",
		},
		{
			Name:         "triggered-with-old-placeholders",
			Provider:     AlertProvider{DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4", TextTwilioTriggered: "Alert: {endpoint} - {description}"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "Body=Alert%3A+endpoint-name+-+description-1&From=3&To=4",
		},
		{
			Name:         "triggered-with-new-placeholders",
			Provider:     AlertProvider{DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4", TextTwilioTriggered: "Alert: [ENDPOINT] - [ALERT_DESCRIPTION]"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "Body=Alert%3A+endpoint-name+-+description-1&From=3&To=4",
		},
		{
			Name:         "resolved-with-mixed-placeholders",
			Provider:     AlertProvider{DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4", TextTwilioResolved: "Resolved: {endpoint} and [ENDPOINT] - {description} and [ALERT_DESCRIPTION]"}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "Body=Resolved%3A+endpoint-name+and+endpoint-name+-+description-2+and+description-2&From=3&To=4",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&scenario.Provider.DefaultConfig,
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
			if body != scenario.ExpectedBody {
				t.Errorf("expected %s, got %s", scenario.ExpectedBody, body)
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
				DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4"},
			},
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{SID: "1", Token: "2", From: "3", To: "4"},
		},
		{
			Name: "provider-with-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{SID: "1", Token: "2", From: "3", To: "4"},
			},
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"sid": "5", "token": "6", "from": "7", "to": "8"}},
			ExpectedOutput: Config{SID: "5", Token: "6", From: "7", To: "8"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig("", &scenario.InputAlert)
			if err != nil {
				t.Error("expected no error, got:", err.Error())
			}
			if got.SID != scenario.ExpectedOutput.SID {
				t.Errorf("expected SID to be %s, got %s", scenario.ExpectedOutput.SID, got.SID)
			}
			if got.Token != scenario.ExpectedOutput.Token {
				t.Errorf("expected token to be %s, got %s", scenario.ExpectedOutput.Token, got.Token)
			}
			if got.From != scenario.ExpectedOutput.From {
				t.Errorf("expected from to be %s, got %s", scenario.ExpectedOutput.From, got.From)
			}
			if got.To != scenario.ExpectedOutput.To {
				t.Errorf("expected to to be %s, got %s", scenario.ExpectedOutput.To, got.To)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides("", &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

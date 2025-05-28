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
	t.Run("empty-invalid-provider", func(t *testing.T) {
		invalidProvider := AlertProvider{}
		if err := invalidProvider.Validate(); err == nil {
			t.Error("provider shouldn't have been valid")
		}
	})
	t.Run("valid-provider", func(t *testing.T) {
		validProvider := AlertProvider{
			DefaultConfig: Config{
				ApplicationToken: "aTokenWithLengthOf30characters",
				UserKey:          "aTokenWithLengthOf30characters",
				Title:            "Gatus Notification",
				Priority:         1,
				ResolvedPriority: 1,
			},
		}
		if err := validProvider.Validate(); err != nil {
			t.Error("provider should've been valid")
		}
	})
	t.Run("invalid-provider", func(t *testing.T) {
		invalidProvider := AlertProvider{
			DefaultConfig: Config{
				ApplicationToken: "aTokenWithLengthOfMoreThan30characters",
				UserKey:          "aTokenWithLengthOfMoreThan30characters",
				Priority:         5,
			},
		}
		if err := invalidProvider.Validate(); err == nil {
			t.Error("provider should've been invalid")
		}
	})
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
			Provider: AlertProvider{DefaultConfig: Config{ApplicationToken: "aTokenWithLengthOf30characters", UserKey: "aTokenWithLengthOf30characters"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{ApplicationToken: "aTokenWithLengthOf30characters", UserKey: "aTokenWithLengthOf30characters"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{ApplicationToken: "aTokenWithLengthOf30characters", UserKey: "aTokenWithLengthOf30characters"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{ApplicationToken: "aTokenWithLengthOf30characters", UserKey: "aTokenWithLengthOf30characters"}},
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
			Provider:     AlertProvider{DefaultConfig: Config{ApplicationToken: "TokenWithLengthOf30Characters1", UserKey: "TokenWithLengthOf30Characters4"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters1\",\"user\":\"TokenWithLengthOf30Characters4\",\"title\":\"Gatus: endpoint-name\",\"message\":\"An alert for \\u003cb\\u003eendpoint-name\\u003c/b\\u003e has been triggered due to having failed 3 time(s) in a row with the following description: description-1\\n❌ - [CONNECTED] == true\\n❌ - [STATUS] == 200\",\"priority\":0,\"html\":1}",
		},
		{
			Name:         "triggered-customtitle",
			Provider:     AlertProvider{DefaultConfig: Config{ApplicationToken: "TokenWithLengthOf30Characters1", UserKey: "TokenWithLengthOf30Characters4", Title: "Gatus Notifications"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters1\",\"user\":\"TokenWithLengthOf30Characters4\",\"title\":\"Gatus Notifications\",\"message\":\"An alert for \\u003cb\\u003eendpoint-name\\u003c/b\\u003e has been triggered due to having failed 3 time(s) in a row with the following description: description-1\\n❌ - [CONNECTED] == true\\n❌ - [STATUS] == 200\",\"priority\":0,\"html\":1}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{ApplicationToken: "TokenWithLengthOf30Characters2", UserKey: "TokenWithLengthOf30Characters5", Priority: 2, ResolvedPriority: 2}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters2\",\"user\":\"TokenWithLengthOf30Characters5\",\"title\":\"Gatus: endpoint-name\",\"message\":\"An alert for \\u003cb\\u003eendpoint-name\\u003c/b\\u003e has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\\n✅ - [CONNECTED] == true\\n✅ - [STATUS] == 200\",\"priority\":2,\"html\":1}",
		},
		{
			Name:         "resolved-priority",
			Provider:     AlertProvider{DefaultConfig: Config{ApplicationToken: "TokenWithLengthOf30Characters2", UserKey: "TokenWithLengthOf30Characters5", Title: "Gatus Notifications", Priority: 2, ResolvedPriority: 0}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters2\",\"user\":\"TokenWithLengthOf30Characters5\",\"title\":\"Gatus Notifications\",\"message\":\"An alert for \\u003cb\\u003eendpoint-name\\u003c/b\\u003e has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\\n✅ - [CONNECTED] == true\\n✅ - [STATUS] == 200\",\"priority\":0,\"html\":1}",
		},
		{
			Name:         "with-sound",
			Provider:     AlertProvider{DefaultConfig: Config{ApplicationToken: "TokenWithLengthOf30Characters2", UserKey: "TokenWithLengthOf30Characters5", Title: "Gatus Notifications", Priority: 2, ResolvedPriority: 2, Sound: "falling"}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters2\",\"user\":\"TokenWithLengthOf30Characters5\",\"title\":\"Gatus Notifications\",\"message\":\"An alert for \\u003cb\\u003eendpoint-name\\u003c/b\\u003e has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\\n✅ - [CONNECTED] == true\\n✅ - [STATUS] == 200\",\"priority\":2,\"html\":1,\"sound\":\"falling\"}",
		},
		{
			Name:         "with-ttl",
			Provider:     AlertProvider{DefaultConfig: Config{ApplicationToken: "TokenWithLengthOf30Characters2", UserKey: "TokenWithLengthOf30Characters5", Title: "Gatus Notifications", Priority: 2, ResolvedPriority: 2, TTL: 3600}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters2\",\"user\":\"TokenWithLengthOf30Characters5\",\"title\":\"Gatus Notifications\",\"message\":\"An alert for \\u003cb\\u003eendpoint-name\\u003c/b\\u003e has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\\n✅ - [CONNECTED] == true\\n✅ - [STATUS] == 200\",\"priority\":2,\"html\":1,\"ttl\":3600}",
		},
        {
            Name:       "with-device",
            Provider:   AlertProvider{DefaultConfig: Config{ApplicationToken: "TokenWithLengthOf30Characters2", UserKey: "TokenWithLengthOf30Characters5", Title: "Gatus Notifications", Priority: 2, ResolvedPriority: 2, TTL: 3600, Device: "iphone15pro",}},
            Alert:      alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
            Resolved:   true,
            ExpectedBody: "{\"token\":\"TokenWithLengthOf30Characters2\",\"user\":\"TokenWithLengthOf30Characters5\",\"title\":\"Gatus Notifications\",\"message\":\"An alert for \\u003cb\\u003eendpoint-name\\u003c/b\\u003e has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\\n✅ - [CONNECTED] == true\\n✅ - [STATUS] == 200\",\"priority\":2,\"html\":1,\"ttl\":3600,\"device\":\"iphone15pro\"}",
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

func TestAlertProvider_GetConfig(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		InputAlert     alert.Alert
		ExpectedOutput Config
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{ApplicationToken: "aTokenWithLengthOf30characters", UserKey: "aTokenWithLengthOf30characters"},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{ApplicationToken: "aTokenWithLengthOf30characters", UserKey: "aTokenWithLengthOf30characters"},
		},
		{
			Name: "provider-with-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{ApplicationToken: "aTokenWithLengthOf30characters", UserKey: "aTokenWithLengthOf30characters"},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"application-token": "TokenWithLengthOf30Characters2", "user-key": "TokenWithLengthOf30Characters3"}},
			ExpectedOutput: Config{ApplicationToken: "TokenWithLengthOf30Characters2", UserKey: "TokenWithLengthOf30Characters3"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.ApplicationToken != scenario.ExpectedOutput.ApplicationToken {
				t.Errorf("expected application token to be %s, got %s", scenario.ExpectedOutput.ApplicationToken, got.ApplicationToken)
			}
			if got.UserKey != scenario.ExpectedOutput.UserKey {
				t.Errorf("expected user key to be %s, got %s", scenario.ExpectedOutput.UserKey, got.UserKey)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

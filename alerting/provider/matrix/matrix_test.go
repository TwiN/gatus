package matrix

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{
		DefaultConfig: Config{
			AccessToken:    "",
			InternalRoomID: "",
		},
	}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{
		DefaultConfig: Config{
			AccessToken:    "1",
			InternalRoomID: "!a:example.com",
		},
	}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
	validProviderWithHomeserver := AlertProvider{
		DefaultConfig: Config{
			ServerURL:      "https://example.com",
			AccessToken:    "1",
			InternalRoomID: "!a:example.com",
		},
	}
	if err := validProviderWithHomeserver.Validate(); err != nil {
		t.Error("provider with homeserver should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Group: "",
				Config: Config{
					AccessToken:    "",
					InternalRoomID: "",
				},
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				Group: "group",
				Config: Config{
					AccessToken:    "",
					InternalRoomID: "",
				},
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{
			AccessToken:    "1",
			InternalRoomID: "!a:example.com",
		},
		Overrides: []Override{
			{
				Group: "group",
				Config: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
			},
		},
	}
	if err := providerWithValidOverride.Validate(); err != nil {
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
			Name:     "triggered-with-bad-config",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "triggered",
			Provider: AlertProvider{DefaultConfig: Config{AccessToken: "1", InternalRoomID: "!a:example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{AccessToken: "1", InternalRoomID: "!a:example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{AccessToken: "1", InternalRoomID: "!a:example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{AccessToken: "1", InternalRoomID: "!a:example.com"}},
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
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"msgtype\":\"m.text\",\"format\":\"org.matrix.custom.html\",\"body\":\"An alert for `endpoint-name` has been triggered due to having failed 3 time(s) in a row\\ndescription-1\\n\\n✕ - [CONNECTED] == true\\n✕ - [STATUS] == 200\",\"formatted_body\":\"\\u003ch3\\u003eAn alert for \\u003ccode\\u003eendpoint-name\\u003c/code\\u003e has been triggered due to having failed 3 time(s) in a row\\u003c/h3\\u003e\\n\\u003cblockquote\\u003edescription-1\\u003c/blockquote\\u003e\\n\\u003ch5\\u003eCondition results\\u003c/h5\\u003e\\u003cul\\u003e\\u003cli\\u003e❌ - \\u003ccode\\u003e[CONNECTED] == true\\u003c/code\\u003e\\u003c/li\\u003e\\u003cli\\u003e❌ - \\u003ccode\\u003e[STATUS] == 200\\u003c/code\\u003e\\u003c/li\\u003e\\u003c/ul\\u003e\"}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"msgtype\":\"m.text\",\"format\":\"org.matrix.custom.html\",\"body\":\"An alert for `endpoint-name` has been resolved after passing successfully 5 time(s) in a row\\ndescription-2\\n\\n✓ - [CONNECTED] == true\\n✓ - [STATUS] == 200\",\"formatted_body\":\"\\u003ch3\\u003eAn alert for \\u003ccode\\u003eendpoint-name\\u003c/code\\u003e has been resolved after passing successfully 5 time(s) in a row\\u003c/h3\\u003e\\n\\u003cblockquote\\u003edescription-2\\u003c/blockquote\\u003e\\n\\u003ch5\\u003eCondition results\\u003c/h5\\u003e\\u003cul\\u003e\\u003cli\\u003e✅ - \\u003ccode\\u003e[CONNECTED] == true\\u003c/code\\u003e\\u003c/li\\u003e\\u003cli\\u003e✅ - \\u003ccode\\u003e[STATUS] == 200\\u003c/code\\u003e\\u003c/li\\u003e\\u003c/ul\\u003e\"}",
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
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: nil,
			},
			InputGroup: "",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				ServerURL:      "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: nil,
			},
			InputGroup: "group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				ServerURL:      "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: []Override{
					{
						Group: "group",
						Config: Config{
							ServerURL:      "https://group-example.com",
							AccessToken:    "12",
							InternalRoomID: "!a:group-example.com",
						},
					},
				},
			},
			InputGroup: "",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				ServerURL:      "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: []Override{
					{
						Group: "group",
						Config: Config{
							ServerURL:      "https://group-example.com",
							AccessToken:    "12",
							InternalRoomID: "!a:group-example.com",
						},
					},
				},
			},
			InputGroup: "group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				ServerURL:      "https://group-example.com",
				AccessToken:    "12",
				InternalRoomID: "!a:group-example.com",
			},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: []Override{
					{
						Group: "group",
						Config: Config{
							ServerURL:      "https://group-example.com",
							AccessToken:    "12",
							InternalRoomID: "!a:example01.com",
						},
					},
				},
			},
			InputGroup: "group",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{"server-url": "https://alert-example.com", "access-token": "123", "internal-room-id": "!a:alert-example.com"}},
			ExpectedOutput: Config{
				ServerURL:      "https://alert-example.com",
				AccessToken:    "123",
				InternalRoomID: "!a:alert-example.com",
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			outputConfig, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
			if outputConfig.ServerURL != scenario.ExpectedOutput.ServerURL {
				t.Errorf("expected ServerURL to be %s, got %s", scenario.ExpectedOutput.ServerURL, outputConfig.ServerURL)
			}
			if outputConfig.AccessToken != scenario.ExpectedOutput.AccessToken {
				t.Errorf("expected AccessToken to be %s, got %s", scenario.ExpectedOutput.AccessToken, outputConfig.AccessToken)
			}
			if outputConfig.InternalRoomID != scenario.ExpectedOutput.InternalRoomID {
				t.Errorf("expected InternalRoomID to be %s, got %s", scenario.ExpectedOutput.InternalRoomID, outputConfig.InternalRoomID)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

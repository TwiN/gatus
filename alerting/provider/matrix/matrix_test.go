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

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{
		ProviderConfig: ProviderConfig{
			AccessToken:    "",
			InternalRoomID: "",
		},
	}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{
		ProviderConfig: ProviderConfig{
			AccessToken:    "1",
			InternalRoomID: "!a:example.com",
		},
	}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
	validProviderWithHomeserver := AlertProvider{
		ProviderConfig: ProviderConfig{
			ServerURL:      "https://example.com",
			AccessToken:    "1",
			InternalRoomID: "!a:example.com",
		},
	}
	if !validProviderWithHomeserver.IsValid() {
		t.Error("provider with homeserver should've been valid")
	}
}

func TestAlertProvider_IsValidWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Group: "",
				ProviderConfig: ProviderConfig{
					AccessToken:    "",
					InternalRoomID: "",
				},
			},
		},
	}
	if providerWithInvalidOverrideGroup.IsValid() {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				Group: "group",
				ProviderConfig: ProviderConfig{
					AccessToken:    "",
					InternalRoomID: "",
				},
			},
		},
	}
	if providerWithInvalidOverrideTo.IsValid() {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		ProviderConfig: ProviderConfig{
			AccessToken:    "1",
			InternalRoomID: "!a:example.com",
		},
		Overrides: []Override{
			{
				Group: "group",
				ProviderConfig: ProviderConfig{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
			},
		},
	}
	if !providerWithValidOverride.IsValid() {
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

func TestAlertProvider_getConfigForGroup(t *testing.T) {
	tests := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		ExpectedOutput ProviderConfig
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				ProviderConfig: ProviderConfig{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: nil,
			},
			InputGroup: "",
			ExpectedOutput: ProviderConfig{
				ServerURL:      "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				ProviderConfig: ProviderConfig{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: nil,
			},
			InputGroup: "group",
			ExpectedOutput: ProviderConfig{
				ServerURL:      "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				ProviderConfig: ProviderConfig{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: []Override{
					{
						Group: "group",
						ProviderConfig: ProviderConfig{
							ServerURL:      "https://example01.com",
							AccessToken:    "12",
							InternalRoomID: "!a:example01.com",
						},
					},
				},
			},
			InputGroup: "",
			ExpectedOutput: ProviderConfig{
				ServerURL:      "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				ProviderConfig: ProviderConfig{
					ServerURL:      "https://example.com",
					AccessToken:    "1",
					InternalRoomID: "!a:example.com",
				},
				Overrides: []Override{
					{
						Group: "group",
						ProviderConfig: ProviderConfig{
							ServerURL:      "https://example01.com",
							AccessToken:    "12",
							InternalRoomID: "!a:example01.com",
						},
					},
				},
			},
			InputGroup: "group",
			ExpectedOutput: ProviderConfig{
				ServerURL:      "https://example01.com",
				AccessToken:    "12",
				InternalRoomID: "!a:example01.com",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if got := tt.Provider.getConfigForGroup(tt.InputGroup); got != tt.ExpectedOutput {
				t.Errorf("AlertProvider.getConfigForGroup() = %v, want %v", got, tt.ExpectedOutput)
			}
		})
	}
}

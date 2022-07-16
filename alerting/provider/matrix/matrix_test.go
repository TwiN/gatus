package matrix

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v4/alerting/alert"
	"github.com/TwiN/gatus/v4/client"
	"github.com/TwiN/gatus/v4/core"
	"github.com/TwiN/gatus/v4/test"
)

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{AccessToken: "", InternalRoomID: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{AccessToken: "1", InternalRoomID: "!a:example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
	validProviderWithHomeserver := AlertProvider{HomeserverURL: "https://example.com", AccessToken: "1", InternalRoomID: "!a:example.com"}
	if !validProviderWithHomeserver.IsValid() {
		t.Error("provider with homeserver should've been valid")
	}
}

func TestAlertProvider_IsValidWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				AccessToken:    "",
				InternalRoomID: "",
				Group:          "",
			},
		},
	}
	if providerWithInvalidOverrideGroup.IsValid() {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				AccessToken:    "",
				InternalRoomID: "",
				Group:          "group",
			},
		},
	}
	if providerWithInvalidOverrideTo.IsValid() {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		AccessToken:    "1",
		InternalRoomID: "!a:example.com",
		Overrides: []Override{
			{
				HomeserverURL:  "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
				Group:          "group",
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
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\n\t\"msgtype\": \"m.text\",\n\t\"format\": \"org.matrix.custom.html\",\n\t\"body\": \"An alert for `endpoint-name` has been triggered due to having failed 3 time(s) in a row\\ndescription-1\\n\\n✕ - [CONNECTED] == true\\n✕ - [STATUS] == 200\",\n\t\"formatted_body\": \"<h3>An alert for <code>endpoint-name</code> has been triggered due to having failed 3 time(s) in a row</h3>\\n<blockquote>description-1</blockquote>\\n<h5>Condition results</h5><ul><li>❌ - <code>[CONNECTED] == true</code></li><li>❌ - <code>[STATUS] == 200</code></li></ul>\"\n}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\n\t\"msgtype\": \"m.text\",\n\t\"format\": \"org.matrix.custom.html\",\n\t\"body\": \"An alert for `endpoint-name` has been resolved after passing successfully 5 time(s) in a row\\ndescription-2\\n\\n✓ - [CONNECTED] == true\\n✓ - [STATUS] == 200\",\n\t\"formatted_body\": \"<h3>An alert for <code>endpoint-name</code> has been resolved after passing successfully 5 time(s) in a row</h3>\\n<blockquote>description-2</blockquote>\\n<h5>Condition results</h5><ul><li>✅ - <code>[CONNECTED] == true</code></li><li>✅ - <code>[STATUS] == 200</code></li></ul>\"\n}",
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

func TestAlertProvider_getConfigForGroup(t *testing.T) {
	tests := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		ExpectedOutput matrixProviderConfig
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				HomeserverURL:  "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
				Overrides:      nil,
			},
			InputGroup: "",
			ExpectedOutput: matrixProviderConfig{
				HomeserverURL:  "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				HomeserverURL:  "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
				Overrides:      nil,
			},
			InputGroup: "group",
			ExpectedOutput: matrixProviderConfig{
				HomeserverURL:  "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				HomeserverURL:  "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
				Overrides: []Override{
					{
						Group:          "group",
						HomeserverURL:  "https://example01.com",
						AccessToken:    "12",
						InternalRoomID: "!a:example01.com",
					},
				},
			},
			InputGroup: "",
			ExpectedOutput: matrixProviderConfig{
				HomeserverURL:  "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
			},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				HomeserverURL:  "https://example.com",
				AccessToken:    "1",
				InternalRoomID: "!a:example.com",
				Overrides: []Override{
					{
						Group:          "group",
						HomeserverURL:  "https://example01.com",
						AccessToken:    "12",
						InternalRoomID: "!a:example01.com",
					},
				},
			},
			InputGroup: "group",
			ExpectedOutput: matrixProviderConfig{
				HomeserverURL:  "https://example01.com",
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

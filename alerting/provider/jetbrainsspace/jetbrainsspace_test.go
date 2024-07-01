package jetbrainsspace

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertDefaultProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{Project: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{Project: "foo", ChannelID: "bar", Token: "baz"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_IsValidWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Project: "foobar",
		Overrides: []Override{
			{
				ChannelID: "http://example.com",
				Group:     "",
			},
		},
	}
	if providerWithInvalidOverrideGroup.IsValid() {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Project: "foobar",
		Overrides: []Override{
			{
				ChannelID: "",
				Group:     "group",
			},
		},
	}
	if providerWithInvalidOverrideTo.IsValid() {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		Project:   "foo",
		ChannelID: "bar",
		Token:     "baz",
		Overrides: []Override{
			{
				ChannelID: "foobar",
				Group:     "group",
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
		Endpoint     endpoint.Endpoint
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: `{"channel":"id:","content":{"className":"ChatMessage.Block","style":"WARNING","sections":[{"className":"MessageSection","elements":[{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"warning"},"style":"WARNING"},"style":"WARNING","size":"REGULAR","content":"[CONNECTED] == true"},{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"warning"},"style":"WARNING"},"style":"WARNING","size":"REGULAR","content":"[STATUS] == 200"}],"header":"An alert for *name* has been triggered due to having failed 3 time(s) in a row"}]}}`,
		},
		{
			Name:         "triggered-with-group",
			Provider:     AlertProvider{},
			Endpoint:     endpoint.Endpoint{Name: "name", Group: "group"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: `{"channel":"id:","content":{"className":"ChatMessage.Block","style":"WARNING","sections":[{"className":"MessageSection","elements":[{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"warning"},"style":"WARNING"},"style":"WARNING","size":"REGULAR","content":"[CONNECTED] == true"},{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"warning"},"style":"WARNING"},"style":"WARNING","size":"REGULAR","content":"[STATUS] == 200"}],"header":"An alert for *group/name* has been triggered due to having failed 3 time(s) in a row"}]}}`,
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: `{"channel":"id:","content":{"className":"ChatMessage.Block","style":"SUCCESS","sections":[{"className":"MessageSection","elements":[{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"success"},"style":"SUCCESS"},"style":"SUCCESS","size":"REGULAR","content":"[CONNECTED] == true"},{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"success"},"style":"SUCCESS"},"style":"SUCCESS","size":"REGULAR","content":"[STATUS] == 200"}],"header":"An alert for *name* has been resolved after passing successfully 5 time(s) in a row"}]}}`,
		},
		{
			Name:         "resolved-with-group",
			Provider:     AlertProvider{},
			Endpoint:     endpoint.Endpoint{Name: "name", Group: "group"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: `{"channel":"id:","content":{"className":"ChatMessage.Block","style":"SUCCESS","sections":[{"className":"MessageSection","elements":[{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"success"},"style":"SUCCESS"},"style":"SUCCESS","size":"REGULAR","content":"[CONNECTED] == true"},{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"success"},"style":"SUCCESS"},"style":"SUCCESS","size":"REGULAR","content":"[STATUS] == 200"}],"header":"An alert for *group/name* has been resolved after passing successfully 5 time(s) in a row"}]}}`,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&scenario.Endpoint,
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

func TestAlertProvider_getChannelIDForGroup(t *testing.T) {
	tests := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		ExpectedOutput string
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				ChannelID: "bar",
			},
			InputGroup:     "",
			ExpectedOutput: "bar",
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				ChannelID: "bar",
			},
			InputGroup:     "group",
			ExpectedOutput: "bar",
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				ChannelID: "bar",
				Overrides: []Override{
					{
						Group:     "group",
						ChannelID: "foobar",
					},
				},
			},
			InputGroup:     "",
			ExpectedOutput: "bar",
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				ChannelID: "bar",
				Overrides: []Override{
					{
						Group:     "group",
						ChannelID: "foobar",
					},
				},
			},
			InputGroup:     "group",
			ExpectedOutput: "foobar",
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			if got := tt.Provider.getChannelIDForGroup(tt.InputGroup); got != tt.ExpectedOutput {
				t.Errorf("AlertProvider.getChannelIDForGroup() = %v, want %v", got, tt.ExpectedOutput)
			}
		})
	}
}

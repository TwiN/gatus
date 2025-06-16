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

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{DefaultConfig: Config{Project: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{Project: "foo", ChannelID: "bar", Token: "baz"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		DefaultConfig: Config{Project: "foobar"},
		Overrides: []Override{
			{
				Config: Config{ChannelID: "http://example.com"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		DefaultConfig: Config{Project: "foobar"},
		Overrides: []Override{
			{
				Config: Config{ChannelID: ""},
				Group:  "group",
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{
			Project:   "foo",
			ChannelID: "bar",
			Token:     "baz",
		},
		Overrides: []Override{
			{
				Config: Config{ChannelID: "foobar"},
				Group:  "group",
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
			Name:     "triggered",
			Provider: AlertProvider{DefaultConfig: Config{ChannelID: "1", Project: "project", Token: "token"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{ChannelID: "1", Project: "project", Token: "token"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{ChannelID: "1", Project: "project", Token: "token"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{ChannelID: "1", Project: "project", Token: "token"}},
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
			Provider:     AlertProvider{DefaultConfig: Config{ChannelID: "1", Project: "project"}},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: `{"channel":"id:1","content":{"className":"ChatMessage.Block","style":"WARNING","sections":[{"className":"MessageSection","elements":[{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"warning"},"style":"WARNING"},"style":"WARNING","size":"REGULAR","content":"[CONNECTED] == true"},{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"warning"},"style":"WARNING"},"style":"WARNING","size":"REGULAR","content":"[STATUS] == 200"}],"header":"An alert for *name* has been triggered due to having failed 3 time(s) in a row"}]}}`,
		},
		{
			Name:         "triggered-with-group",
			Provider:     AlertProvider{DefaultConfig: Config{ChannelID: "1", Project: "project"}},
			Endpoint:     endpoint.Endpoint{Name: "name", Group: "group"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: `{"channel":"id:1","content":{"className":"ChatMessage.Block","style":"WARNING","sections":[{"className":"MessageSection","elements":[{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"warning"},"style":"WARNING"},"style":"WARNING","size":"REGULAR","content":"[CONNECTED] == true"},{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"warning"},"style":"WARNING"},"style":"WARNING","size":"REGULAR","content":"[STATUS] == 200"}],"header":"An alert for *group/name* has been triggered due to having failed 3 time(s) in a row"}]}}`,
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{ChannelID: "1", Project: "project"}},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: `{"channel":"id:1","content":{"className":"ChatMessage.Block","style":"SUCCESS","sections":[{"className":"MessageSection","elements":[{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"success"},"style":"SUCCESS"},"style":"SUCCESS","size":"REGULAR","content":"[CONNECTED] == true"},{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"success"},"style":"SUCCESS"},"style":"SUCCESS","size":"REGULAR","content":"[STATUS] == 200"}],"header":"An alert for *name* has been resolved after passing successfully 5 time(s) in a row"}]}}`,
		},
		{
			Name:         "resolved-with-group",
			Provider:     AlertProvider{DefaultConfig: Config{ChannelID: "1", Project: "project"}},
			Endpoint:     endpoint.Endpoint{Name: "name", Group: "group"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: `{"channel":"id:1","content":{"className":"ChatMessage.Block","style":"SUCCESS","sections":[{"className":"MessageSection","elements":[{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"success"},"style":"SUCCESS"},"style":"SUCCESS","size":"REGULAR","content":"[CONNECTED] == true"},{"className":"MessageText","accessory":{"className":"MessageIcon","icon":{"icon":"success"},"style":"SUCCESS"},"style":"SUCCESS","size":"REGULAR","content":"[STATUS] == 200"}],"header":"An alert for *group/name* has been resolved after passing successfully 5 time(s) in a row"}]}}`,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&scenario.Provider.DefaultConfig,
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
				DefaultConfig: Config{ChannelID: "default", Project: "project", Token: "token"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{ChannelID: "default", Project: "project", Token: "token"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{ChannelID: "default", Project: "project", Token: "token"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{ChannelID: "default", Project: "project", Token: "token"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{ChannelID: "default", Project: "project", Token: "token"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{ChannelID: "group-channel"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{ChannelID: "default", Project: "project", Token: "token"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{ChannelID: "default", Project: "project", Token: "token"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{ChannelID: "group-channel"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{ChannelID: "group-channel", Project: "project", Token: "token"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{ChannelID: "default", Project: "project", Token: "token"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{ChannelID: "group-channel"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"channel-id": "alert-channel", "project": "alert-project", "token": "alert-token"}},
			ExpectedOutput: Config{ChannelID: "alert-channel", Project: "alert-project", Token: "alert-token"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.ChannelID != scenario.ExpectedOutput.ChannelID {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput.ChannelID, got.ChannelID)
			}
			if got.Project != scenario.ExpectedOutput.Project {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput.Project, got.Project)
			}
			if got.Token != scenario.ExpectedOutput.Token {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput.Token, got.Token)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

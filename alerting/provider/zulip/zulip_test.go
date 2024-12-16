package zulip

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	scenarios := []struct {
		Name          string
		AlertProvider AlertProvider
		ExpectedError error
	}{
		{
			Name:          "Empty provider",
			AlertProvider: AlertProvider{},
			ExpectedError: ErrBotEmailNotSet,
		},
		{
			Name: "Empty channel id",
			AlertProvider: AlertProvider{
				DefaultConfig: Config{
					BotEmail:  "something",
					BotAPIKey: "something",
					Domain:    "something",
				},
			},
			ExpectedError: ErrChannelIDNotSet,
		},
		{
			Name: "Empty domain",
			AlertProvider: AlertProvider{
				DefaultConfig: Config{
					BotEmail:  "something",
					BotAPIKey: "something",
					ChannelID: "something",
				},
			},
			ExpectedError: ErrDomainNotSet,
		},
		{
			Name: "Empty bot api key",
			AlertProvider: AlertProvider{
				DefaultConfig: Config{
					BotEmail:  "something",
					Domain:    "something",
					ChannelID: "something",
				},
			},
			ExpectedError: ErrBotAPIKeyNotSet,
		},
		{
			Name: "Empty bot email",
			AlertProvider: AlertProvider{
				DefaultConfig: Config{
					BotAPIKey: "something",
					Domain:    "something",
					ChannelID: "something",
				},
			},
			ExpectedError: ErrBotEmailNotSet,
		},
		{
			Name: "Valid provider",
			AlertProvider: AlertProvider{
				DefaultConfig: Config{
					BotEmail:  "something",
					BotAPIKey: "something",
					Domain:    "something",
					ChannelID: "something",
				},
			},
			ExpectedError: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if err := scenario.AlertProvider.Validate(); !errors.Is(err, scenario.ExpectedError) {
				t.Errorf("ExpectedError error %v, got %v", scenario.ExpectedError, err)
			}
		})
	}
}

func TestAlertProvider_buildRequestBody(t *testing.T) {
	basicConfig := Config{
		BotEmail:  "bot-email",
		BotAPIKey: "bot-api-key",
		Domain:    "domain",
		ChannelID: "channel-id",
	}
	alertDesc := "Description"
	basicAlert := alert.Alert{
		SuccessThreshold: 2,
		FailureThreshold: 3,
		Description:      &alertDesc,
	}
	testCases := []struct {
		name          string
		provider      AlertProvider
		alert         alert.Alert
		resolved      bool
		hasConditions bool
		expectedBody  url.Values
	}{
		{
			name: "Resolved alert with no conditions",
			provider: AlertProvider{
				DefaultConfig: basicConfig,
			},
			alert:         basicAlert,
			resolved:      true,
			hasConditions: false,
			expectedBody: url.Values{
				"content": {`An alert for **endpoint-Name** has been resolved after passing successfully 2 time(s) in a row
> Description
`},
				"to":    {"channel-id"},
				"topic": {"Gatus"},
				"type":  {"channel"},
			},
		},
		{
			name: "Resolved alert with conditions",
			provider: AlertProvider{
				DefaultConfig: basicConfig,
			},
			alert:         basicAlert,
			resolved:      true,
			hasConditions: true,
			expectedBody: url.Values{
				"content": {`An alert for **endpoint-Name** has been resolved after passing successfully 2 time(s) in a row
> Description

:check: - ` + "`[CONNECTED] == true`" + `
:check: - ` + "`[STATUS] == 200`" + `
:check: - ` + "`[BODY] != \"\"`"},
				"to":    {"channel-id"},
				"topic": {"Gatus"},
				"type":  {"channel"},
			},
		},
		{
			name: "Failed alert with no conditions",
			provider: AlertProvider{
				DefaultConfig: basicConfig,
			},
			alert:         basicAlert,
			resolved:      false,
			hasConditions: false,
			expectedBody: url.Values{
				"content": {`An alert for **endpoint-Name** has been triggered due to having failed 3 time(s) in a row
> Description
`},
				"to":    {"channel-id"},
				"topic": {"Gatus"},
				"type":  {"channel"},
			},
		},
		{
			name: "Failed alert with conditions",
			provider: AlertProvider{
				DefaultConfig: basicConfig,
			},
			alert:         basicAlert,
			resolved:      false,
			hasConditions: true,
			expectedBody: url.Values{
				"content": {`An alert for **endpoint-Name** has been triggered due to having failed 3 time(s) in a row
> Description

:cross_mark: - ` + "`[CONNECTED] == true`" + `
:cross_mark: - ` + "`[STATUS] == 200`" + `
:cross_mark: - ` + "`[BODY] != \"\"`"},
				"to":    {"channel-id"},
				"topic": {"Gatus"},
				"type":  {"channel"},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var conditionResults []*endpoint.ConditionResult
			if tc.hasConditions {
				conditionResults = []*endpoint.ConditionResult{
					{Condition: "[CONNECTED] == true", Success: tc.resolved},
					{Condition: "[STATUS] == 200", Success: tc.resolved},
					{Condition: "[BODY] != \"\"", Success: tc.resolved},
				}
			}
			body := tc.provider.buildRequestBody(
				&tc.provider.DefaultConfig,
				&endpoint.Endpoint{Name: "endpoint-Name"},
				&tc.alert,
				&endpoint.Result{
					ConditionResults: conditionResults,
				},
				tc.resolved,
			)
			valuesResult, err := url.ParseQuery(body)
			if err != nil {
				t.Error(err)
			}
			if fmt.Sprintf("%v", valuesResult) != fmt.Sprintf("%v", tc.expectedBody) {
				t.Errorf("Expected body:\n%v\ngot:\n%v", tc.expectedBody, valuesResult)
			}
		})
	}
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	if (&AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("ExpectedError default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("ExpectedError default alert to be nil")
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	validateRequest := func(req *http.Request) {
		if req.URL.String() != "https://custom-domain/api/v1/messages" {
			t.Errorf("ExpectedError url https://custom-domain.zulipchat.com/api/v1/messages, got %s", req.URL.String())
		}
		if req.Method != http.MethodPost {
			t.Errorf("ExpectedError POST request, got %s", req.Method)
		}
		if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("ExpectedError Content-Type header to be application/x-www-form-urlencoded, got %s", req.Header.Get("Content-Type"))
		}
		if req.Header.Get("User-Agent") != "Gatus" {
			t.Errorf("ExpectedError User-Agent header to be Gatus, got %s", req.Header.Get("User-Agent"))
		}
	}
	basicConfig := Config{
		BotEmail:  "bot-email",
		BotAPIKey: "bot-api-key",
		Domain:    "custom-domain",
		ChannelID: "channel-id",
	}
	basicAlert := alert.Alert{
		SuccessThreshold: 2,
		FailureThreshold: 3,
	}
	testCases := []struct {
		name             string
		provider         AlertProvider
		alert            alert.Alert
		resolved         bool
		mockRoundTripper test.MockRoundTripper
		expectedError    bool
	}{
		{
			name: "resolved",
			provider: AlertProvider{
				DefaultConfig: basicConfig,
			},
			alert:    basicAlert,
			resolved: true,
			mockRoundTripper: test.MockRoundTripper(func(req *http.Request) *http.Response {
				validateRequest(req)
				return &http.Response{StatusCode: http.StatusOK}
			}),
			expectedError: false,
		},
		{
			name: "resolved error",
			provider: AlertProvider{
				DefaultConfig: basicConfig,
			},
			alert:    basicAlert,
			resolved: true,
			mockRoundTripper: test.MockRoundTripper(func(req *http.Request) *http.Response {
				validateRequest(req)
				return &http.Response{StatusCode: http.StatusInternalServerError}
			}),
			expectedError: true,
		},
		{
			name: "triggered",
			provider: AlertProvider{
				DefaultConfig: basicConfig,
			},
			alert:    basicAlert,
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(req *http.Request) *http.Response {
				validateRequest(req)
				return &http.Response{StatusCode: http.StatusOK}
			}),
			expectedError: false,
		},
		{
			name: "triggered error",
			provider: AlertProvider{
				DefaultConfig: basicConfig,
			},
			alert:    basicAlert,
			resolved: false,
			mockRoundTripper: test.MockRoundTripper(func(req *http.Request) *http.Response {
				validateRequest(req)
				return &http.Response{StatusCode: http.StatusInternalServerError}
			}),
			expectedError: true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: tc.mockRoundTripper})
			err := tc.provider.Send(
				&endpoint.Endpoint{Name: "endpoint-Name"},
				&tc.alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: tc.resolved},
						{Condition: "[STATUS] == 200", Success: tc.resolved},
					},
				},
				tc.resolved,
			)
			if tc.expectedError && err == nil {
				t.Error("ExpectedError error, got none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("ExpectedError no error, got: %v", err)
			}
		})
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
			Name: "provider-no-overrides",
			Provider: AlertProvider{
				DefaultConfig: Config{
					BotEmail:  "default-bot-email",
					BotAPIKey: "default-bot-api-key",
					Domain:    "default-domain",
					ChannelID: "default-channel-id",
				},
				Overrides: nil,
			},
			InputGroup: "group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				BotEmail:  "default-bot-email",
				BotAPIKey: "default-bot-api-key",
				Domain:    "default-domain",
				ChannelID: "default-channel-id",
			},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{
					BotEmail:  "default-bot-email",
					BotAPIKey: "default-bot-api-key",
					Domain:    "default-domain",
					ChannelID: "default-channel-id",
				},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{ChannelID: "group-channel-id"},
					},
				},
			},
			InputGroup: "",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				BotEmail:  "default-bot-email",
				BotAPIKey: "default-bot-api-key",
				Domain:    "default-domain",
				ChannelID: "default-channel-id",
			},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					BotEmail:  "default-bot-email",
					BotAPIKey: "default-bot-api-key",
					Domain:    "default-domain",
					ChannelID: "default-channel-id",
				},
				Overrides: []Override{
					{
						Group: "group",
						Config: Config{
							BotEmail:  "group-bot-email",
							BotAPIKey: "group-bot-api-key",
							Domain:    "group-domain",
							ChannelID: "group-channel-id",
						},
					},
				},
			},
			InputGroup: "group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				BotEmail:  "group-bot-email",
				BotAPIKey: "group-bot-api-key",
				Domain:    "group-domain",
				ChannelID: "group-channel-id",
			},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{
					BotEmail:  "default-bot-email",
					BotAPIKey: "default-bot-api-key",
					Domain:    "default-domain",
					ChannelID: "default-channel-id",
				},
				Overrides: []Override{
					{
						Group: "group",
						Config: Config{
							BotEmail:  "group-bot-email",
							BotAPIKey: "group-bot-api-key",
							Domain:    "group-domain",
							ChannelID: "group-channel-id",
						},
					},
				},
			},
			InputGroup: "group",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{
				"bot-email":   "alert-bot-email",
				"bot-api-key": "alert-bot-api-key",
				"domain":      "alert-domain",
				"channel-id":  "alert-channel-id",
			}},
			ExpectedOutput: Config{
				BotEmail:  "alert-bot-email",
				BotAPIKey: "alert-bot-api-key",
				Domain:    "alert-domain",
				ChannelID: "alert-channel-id",
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.BotEmail != scenario.ExpectedOutput.BotEmail {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput.BotEmail, got.BotEmail)
			}
			if got.BotAPIKey != scenario.ExpectedOutput.BotAPIKey {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput.BotAPIKey, got.BotAPIKey)
			}
			if got.Domain != scenario.ExpectedOutput.Domain {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput.Domain, got.Domain)
			}
			if got.ChannelID != scenario.ExpectedOutput.ChannelID {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput.ChannelID, got.ChannelID)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

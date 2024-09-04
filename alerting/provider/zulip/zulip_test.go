package zulip

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_IsValid(t *testing.T) {
	testCase := []struct {
		name          string
		alertProvider AlertProvider
		expected      bool
	}{
		{
			name:          "Empty provider",
			alertProvider: AlertProvider{},
			expected:      false,
		},
		{
			name: "Empty channel id",
			alertProvider: AlertProvider{
				Config: Config{
					BotEmail:  "something",
					BotAPIKey: "something",
					Domain:    "something",
				},
			},
			expected: false,
		},
		{
			name: "Empty domain",
			alertProvider: AlertProvider{
				Config: Config{
					BotEmail:  "something",
					BotAPIKey: "something",
					ChannelID: "something",
				},
			},
			expected: false,
		},
		{
			name: "Empty bot api key",
			alertProvider: AlertProvider{
				Config: Config{
					BotEmail:  "something",
					Domain:    "something",
					ChannelID: "something",
				},
			},
			expected: false,
		},
		{
			name: "Empty bot email",
			alertProvider: AlertProvider{
				Config: Config{
					BotAPIKey: "something",
					Domain:    "something",
					ChannelID: "something",
				},
			},
			expected: false,
		},
		{
			name: "Valid provider",
			alertProvider: AlertProvider{
				Config: Config{
					BotEmail:  "something",
					BotAPIKey: "something",
					Domain:    "something",
					ChannelID: "something",
				},
			},
			expected: true,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			if tc.alertProvider.IsValid() != tc.expected {
				t.Errorf("IsValid assertion failed (expected %v, got %v)", tc.expected, !tc.expected)
			}
		})
	}
}

func TestAlertProvider_IsValidWithOverride(t *testing.T) {
	validConfig := Config{
		BotEmail:  "something",
		BotAPIKey: "something",
		Domain:    "something",
		ChannelID: "something",
	}

	testCase := []struct {
		name          string
		alertProvider AlertProvider
		expected      bool
	}{
		{
			name: "Empty group",
			alertProvider: AlertProvider{
				Config: validConfig,
				Overrides: []Override{
					{
						Config: validConfig,
						Group:  "",
					},
				},
			},
			expected: false,
		},
		{
			name: "Empty override config",
			alertProvider: AlertProvider{
				Config: validConfig,
				Overrides: []Override{
					{
						Group: "something",
					},
				},
			},
			expected: false,
		},
		{
			name: "Empty channel id",
			alertProvider: AlertProvider{
				Config: validConfig,
				Overrides: []Override{
					{
						Group: "something",
						Config: Config{
							BotEmail:  "something",
							BotAPIKey: "something",
							Domain:    "something",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Empty domain",
			alertProvider: AlertProvider{
				Config: validConfig,
				Overrides: []Override{
					{
						Group: "something",
						Config: Config{
							BotEmail:  "something",
							BotAPIKey: "something",
							ChannelID: "something",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Empty bot api key",
			alertProvider: AlertProvider{
				Config: validConfig,
				Overrides: []Override{
					{
						Group: "something",
						Config: Config{
							BotEmail:  "something",
							Domain:    "something",
							ChannelID: "something",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Empty bot email",
			alertProvider: AlertProvider{
				Config: validConfig,
				Overrides: []Override{
					{
						Group: "something",
						Config: Config{
							BotAPIKey: "something",
							Domain:    "something",
							ChannelID: "something",
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Valid provider",
			alertProvider: AlertProvider{
				Config: validConfig,
				Overrides: []Override{
					{
						Group:  "something",
						Config: validConfig,
					},
				},
			},
			expected: true,
		},
	}
	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			if tc.alertProvider.IsValid() != tc.expected {
				t.Errorf("IsValid assertion failed (expected %v, got %v)", tc.expected, !tc.expected)
			}
		})
	}
}

func TestAlertProvider_GetChannelIdForGroup(t *testing.T) {
	provider := AlertProvider{
		Config: Config{
			ChannelID: "default",
		},
		Overrides: []Override{
			{
				Group:  "group1",
				Config: Config{ChannelID: "group1"},
			},
			{
				Group:  "group2",
				Config: Config{ChannelID: "group2"},
			},
		},
	}
	if provider.getChannelIdForGroup("") != "default" {
		t.Error("Expected default channel ID")
	}
	if provider.getChannelIdForGroup("group2") != "group2" {
		t.Error("Expected group2 channel ID")
	}
}

func TestAlertProvider_BuildRequestBody(t *testing.T) {
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
				Config: basicConfig,
			},
			alert:         basicAlert,
			resolved:      true,
			hasConditions: false,
			expectedBody: url.Values{
				"content": {`An alert for **endpoint-name** has been resolved after passing successfully 2 time(s) in a row
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
				Config: basicConfig,
			},
			alert:         basicAlert,
			resolved:      true,
			hasConditions: true,
			expectedBody: url.Values{
				"content": {`An alert for **endpoint-name** has been resolved after passing successfully 2 time(s) in a row
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
				Config: basicConfig,
			},
			alert:         basicAlert,
			resolved:      false,
			hasConditions: false,
			expectedBody: url.Values{
				"content": {`An alert for **endpoint-name** has been triggered due to having failed 3 time(s) in a row
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
				Config: basicConfig,
			},
			alert:         basicAlert,
			resolved:      false,
			hasConditions: true,
			expectedBody: url.Values{
				"content": {`An alert for **endpoint-name** has been triggered due to having failed 3 time(s) in a row
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
				&endpoint.Endpoint{Name: "endpoint-name"},
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
		t.Error("expected default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	validateRequest := func(req *http.Request) {
		if req.URL.String() != "https://custom-domain/api/v1/messages" {
			t.Errorf("expected url https://custom-domain.zulipchat.com/api/v1/messages, got %s", req.URL.String())
		}
		if req.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", req.Method)
		}
		if req.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("expected Content-Type header to be application/x-www-form-urlencoded, got %s", req.Header.Get("Content-Type"))
		}
		if req.Header.Get("User-Agent") != "Gatus" {
			t.Errorf("expected User-Agent header to be Gatus, got %s", req.Header.Get("User-Agent"))
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
				Config: basicConfig,
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
				Config: basicConfig,
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
				Config: basicConfig,
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
				Config: basicConfig,
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
				&endpoint.Endpoint{Name: "endpoint-name"},
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
				t.Error("expected error, got none")
			}
			if !tc.expectedError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

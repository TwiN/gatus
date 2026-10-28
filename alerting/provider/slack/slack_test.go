package slack

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestConfig_Validate(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		name string
		cfg  Config
		err  error
	}{
		{name: "missing credentials", err: ErrWebhookURLNotSet},
		{name: "webhook", cfg: Config{WebhookURL: "https://example.com"}},
		{name: "bot", cfg: Config{BotToken: "test-token", ChannelID: "C123"}},
		{name: "missing channel", cfg: Config{BotToken: "test-token"}, err: ErrChannelIDNotSet},
		{name: "channel without credentials", cfg: Config{ChannelID: "C123"}, err: ErrWebhookURLNotSet},
		{
			name: "both authentication methods",
			cfg:  Config{WebhookURL: "https://example.com", BotToken: "test-token", ChannelID: "C123"},
			err:  ErrMultipleAuthMethods,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			if err := scenario.cfg.Validate(); !errors.Is(err, scenario.err) {
				t.Errorf("expected %v, got %v", scenario.err, err)
			}
		})
	}
}

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{DefaultConfig: Config{WebhookURL: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{WebhookURL: "https://example.com"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{WebhookURL: "http://example.com"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{WebhookURL: ""},
				Group:  "group",
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{WebhookURL: "http://example.com"},
		Overrides: []Override{
			{
				Config: Config{WebhookURL: "http://example.com"},
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
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
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
			client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.URL.String() != scenario.Provider.DefaultConfig.WebhookURL {
					t.Errorf("unexpected webhook URL: %s", r.URL)
				}
				if r.Header.Get("Authorization") != "" {
					t.Error("webhook request must not contain an Authorization header")
				}
				var body map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				if _, exists := body["channel"]; exists {
					t.Error("webhook request must not contain a channel")
				}
				return scenario.MockRoundTripper(r)
			})})
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

func TestAlertProvider_SendWithBotToken(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	scenarios := []struct {
		name     string
		response string
		status   int
		group    string
		channel  string
		resolved bool
		wantErr  string
	}{
		{name: "triggered", response: `{"ok":true}`, status: http.StatusOK, channel: "C123"},
		{name: "resolved with group override", response: `{"ok":true}`, status: http.StatusOK, group: "production", channel: "C456", resolved: true},
		{name: "unmatched group", response: `{"ok":true}`, status: http.StatusOK, group: "staging", channel: "C123"},
		{name: "API error", response: `{"ok":false,"error":"channel_not_found"}`, status: http.StatusOK, channel: "C123", wantErr: "channel_not_found"},
		{name: "HTTP error", response: `rate limited`, status: http.StatusTooManyRequests, channel: "C123", wantErr: "status code 429"},
		{name: "malformed response", response: `{`, status: http.StatusOK, channel: "C123", wantErr: "failed to decode Slack API response"},
		{name: "empty response", status: http.StatusOK, channel: "C123", wantErr: "failed to decode Slack API response"},
		{name: "missing success flag", response: `{}`, status: http.StatusOK, channel: "C123", wantErr: "Slack API returned an error"},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			provider := AlertProvider{
				DefaultConfig: Config{BotToken: "test-token", ChannelID: "C123"},
				Overrides:     []Override{{Group: "production", Config: Config{ChannelID: "C456"}}},
			}
			client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Method != http.MethodPost || r.URL.String() != "https://slack.com/api/chat.postMessage" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL)
				}
				if r.Header.Get("Authorization") != "Bearer test-token" {
					t.Error("missing bot token Authorization header")
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("expected JSON Content-Type")
				}
				bodyBytes, err := io.ReadAll(r.Body)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(bodyBytes), "test-token") {
					t.Error("bot token must not appear in the request body")
				}
				var body Body
				if err := json.Unmarshal(bodyBytes, &body); err != nil {
					t.Fatal(err)
				}
				if body.Channel != scenario.channel {
					t.Errorf("expected channel %s, got %s", scenario.channel, body.Channel)
				}
				expectedColor := "#DD0000"
				if scenario.resolved {
					expectedColor = "#36A64F"
				}
				if len(body.Attachments) != 1 || body.Attachments[0].Color != expectedColor {
					t.Errorf("unexpected attachments: %+v", body.Attachments)
				}
				return &http.Response{StatusCode: scenario.status, Body: io.NopCloser(strings.NewReader(scenario.response))}
			})})
			err := provider.Send(&endpoint.Endpoint{Name: "website", Group: scenario.group}, &alert.Alert{}, &endpoint.Result{}, scenario.resolved)
			if scenario.wantErr == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), scenario.wantErr) {
				t.Errorf("expected error containing %q, got %v", scenario.wantErr, err)
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
		NoConditions bool
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"text\":\"\",\"attachments\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"text\":\"An alert for *name* has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\",\"short\":false,\"color\":\"#DD0000\",\"fields\":[{\"title\":\"Condition results\",\"value\":\":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n\",\"short\":false}]}]}",
		},
		{
			Name:         "triggered-with-group",
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Endpoint:     endpoint.Endpoint{Name: "name", Group: "group"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"text\":\"\",\"attachments\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"text\":\"An alert for *group/name* has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\",\"short\":false,\"color\":\"#DD0000\",\"fields\":[{\"title\":\"Condition results\",\"value\":\":x: - `[CONNECTED] == true`\\n:x: - `[STATUS] == 200`\\n\",\"short\":false}]}]}",
		},
		{
			Name:         "triggered-with-no-conditions",
			NoConditions: true,
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"text\":\"\",\"attachments\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"text\":\"An alert for *name* has been triggered due to having failed 3 time(s) in a row:\\n\\u003e description-1\",\"short\":false,\"color\":\"#DD0000\"}]}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Endpoint:     endpoint.Endpoint{Name: "name"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"text\":\"\",\"attachments\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"text\":\"An alert for *name* has been resolved after passing successfully 5 time(s) in a row:\\n\\u003e description-2\",\"short\":false,\"color\":\"#36A64F\",\"fields\":[{\"title\":\"Condition results\",\"value\":\":white_check_mark: - `[CONNECTED] == true`\\n:white_check_mark: - `[STATUS] == 200`\\n\",\"short\":false}]}]}",
		},
		{
			Name:         "resolved-with-group",
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Endpoint:     endpoint.Endpoint{Name: "name", Group: "group"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"text\":\"\",\"attachments\":[{\"title\":\":helmet_with_white_cross: Gatus\",\"text\":\"An alert for *group/name* has been resolved after passing successfully 5 time(s) in a row:\\n\\u003e description-2\",\"short\":false,\"color\":\"#36A64F\",\"fields\":[{\"title\":\"Condition results\",\"value\":\":white_check_mark: - `[CONNECTED] == true`\\n:white_check_mark: - `[STATUS] == 200`\\n\",\"short\":false}]}]}",
		},
		{
			Name:         "resolved-with-group-and-custom-title",
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com", Title: "custom title"}},
			Endpoint:     endpoint.Endpoint{Name: "name", Group: "group"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"text\":\"\",\"attachments\":[{\"title\":\"custom title\",\"text\":\"An alert for *group/name* has been resolved after passing successfully 5 time(s) in a row:\\n\\u003e description-2\",\"short\":false,\"color\":\"#36A64F\",\"fields\":[{\"title\":\"Condition results\",\"value\":\":white_check_mark: - `[CONNECTED] == true`\\n:white_check_mark: - `[STATUS] == 200`\\n\",\"short\":false}]}]}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var conditionResults []*endpoint.ConditionResult
			if !scenario.NoConditions {
				conditionResults = []*endpoint.ConditionResult{
					{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
					{Condition: "[STATUS] == 200", Success: scenario.Resolved},
				}
			}
			cfg, err := scenario.Provider.GetConfig(scenario.Endpoint.Group, &scenario.Alert)
			if err != nil {
				t.Fatal("couldn't get config:", err.Error())
			}
			body := scenario.Provider.buildRequestBody(
				cfg,
				&scenario.Endpoint,
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: conditionResults,
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
			Name:           "bot-default",
			Provider:       AlertProvider{DefaultConfig: Config{BotToken: "test-token", ChannelID: "C123"}},
			ExpectedOutput: Config{BotToken: "test-token", ChannelID: "C123"},
		},
		{
			Name: "bot-group-channel-override",
			Provider: AlertProvider{
				DefaultConfig: Config{BotToken: "test-token", ChannelID: "C123", Title: "Default title"},
				Overrides:     []Override{{Group: "group", Config: Config{ChannelID: "C456"}}},
			},
			InputGroup:     "group",
			ExpectedOutput: Config{BotToken: "test-token", ChannelID: "C456", Title: "Default title"},
		},
		{
			Name: "bot-alert-override-takes-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{BotToken: "test-token", ChannelID: "C123"},
				Overrides:     []Override{{Group: "group", Config: Config{ChannelID: "C456"}}},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"bot-token": "override-token", "channel-id": "C789", "title": "Alert title"}},
			ExpectedOutput: Config{BotToken: "override-token", ChannelID: "C789", Title: "Alert title"},
		},
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://example01.com"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://group-example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://group-example.com"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://group-example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"webhook-url": "http://alert-example.com"}},
			ExpectedOutput: Config{WebhookURL: "http://alert-example.com"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if *got != scenario.ExpectedOutput {
				t.Error("merged configuration does not match expected configuration")
			}
			if err := scenario.Provider.Validate(); err != nil {
				t.Errorf("unexpected provider validation error: %s", err)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

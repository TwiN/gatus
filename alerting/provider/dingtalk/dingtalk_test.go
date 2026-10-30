package dingtalk

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{DefaultConfig: Config{Token: "", WebhookURL: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProviderWithToken := AlertProvider{DefaultConfig: Config{Token: "test-token"}}
	if err := validProviderWithToken.Validate(); err != nil {
		t.Error("provider with token should've been valid")
	}
	validProviderWithWebhookURL := AlertProvider{DefaultConfig: Config{WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=123"}}
	if err := validProviderWithWebhookURL.Validate(); err != nil {
		t.Error("provider with webhook-url should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{Token: "test-token"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithDuplicateGroup := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{Token: "test-token"},
				Group:  "core",
			},
			{
				Config: Config{Token: "test-token-2"},
				Group:  "core",
			},
		},
	}
	if err := providerWithDuplicateGroup.Validate(); err == nil {
		t.Error("provider with duplicate group override shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{Token: "test-token"},
		Overrides: []Override{
			{
				Config: Config{Token: "override-token"},
				Group:  "group",
			},
		},
	}
	if err := providerWithValidOverride.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestConfig_sign(t *testing.T) {
	cfg := Config{Secret: "dummy-secret"}
	expectedSign := "UMWp0Ns7cce1gjAUDPV9YAFBrt0eWwYa3Ea0gs%2BGtfg%3D"
	if actualSign := cfg.sign(1600000000000); actualSign != expectedSign {
		t.Errorf("expected sign %s, got %s", expectedSign, actualSign)
	}
}

func TestConfig_GetURL(t *testing.T) {
	// Case 1: Token only
	cfg1 := Config{Token: "my-token"}
	u1, err := cfg1.GetURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u1 != "https://oapi.dingtalk.com/robot/send?access_token=my-token" {
		t.Errorf("unexpected URL: %s", u1)
	}

	// Case 2: WebhookURL only
	cfg2 := Config{WebhookURL: "https://custom.webhook.url/robot/send"}
	u2, err := cfg2.GetURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u2 != "https://custom.webhook.url/robot/send" {
		t.Errorf("unexpected URL: %s", u2)
	}

	// Case 3: Token + Secret (HMAC-SHA256 signature)
	cfg3 := Config{Token: "my-token", Secret: "dummy-secret"}
	u3, err := cfg3.GetURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(u3, "https://oapi.dingtalk.com/robot/send?access_token=my-token&timestamp=") || !strings.Contains(u3, "&sign=") {
		t.Errorf("URL missing signature parameters: %s", u3)
	}

	// Case 4: WebhookURL with query + Secret
	cfg4 := Config{WebhookURL: "https://custom.webhook.url/robot/send?key=val", Secret: "dummy-secret"}
	u4, err := cfg4.GetURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(u4, "https://custom.webhook.url/robot/send?key=val&timestamp=") || !strings.Contains(u4, "&sign=") {
		t.Errorf("URL missing signature parameters for URL with existing query: %s", u4)
	}

	// Case 5: Neither token nor webhook-url
	cfg5 := Config{}
	_, err = cfg5.GetURL()
	if err == nil {
		t.Error("expected error for empty config, got nil")
	}
}

func TestAlertProvider_GetConfig(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		InputAlert     alert.Alert
		ExpectedOutput Config
		ExpectedError  bool
	}{
		{
			Name: "default-config-without-overrides",
			Provider: AlertProvider{
				DefaultConfig: Config{Token: "default-token", Secret: "default-secret", Title: "Custom Title"},
			},
			InputGroup:     "any",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{Token: "default-token", Secret: "default-secret", Title: "Custom Title"},
		},
		{
			Name: "group-override-supplies-webhook-url-inherits-secret",
			Provider: AlertProvider{
				DefaultConfig: Config{Token: "default-token", Secret: "default-secret"},
				Overrides: []Override{
					{
						Group:  "ops",
						Config: Config{WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=ops-token"},
					},
				},
			},
			InputGroup:     "ops",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=ops-token", Secret: "default-secret"},
		},
		{
			Name: "group-override-supplies-secret-inherits-token",
			Provider: AlertProvider{
				DefaultConfig: Config{Token: "default-token", Secret: "default-secret"},
				Overrides: []Override{
					{
						Group:  "ops",
						Config: Config{Secret: "ops-secret"},
					},
				},
			},
			InputGroup:     "ops",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{Token: "default-token", Secret: "ops-secret"},
		},
		{
			Name: "group-override-supplies-token-replaces-default-webhook-url",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "https://oapi.dingtalk.com/robot/send?access_token=default"},
				Overrides: []Override{
					{
						Group:  "ops",
						Config: Config{Token: "ops-token"},
					},
				},
			},
			InputGroup:     "ops",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{Token: "ops-token"},
		},
		{
			Name: "alert-override-replaces-token",
			Provider: AlertProvider{
				DefaultConfig: Config{Token: "default-token", Secret: "default-secret"},
			},
			InputGroup: "core",
			InputAlert: alert.Alert{
				ProviderOverride: map[string]any{
					"token": "alert-token",
				},
			},
			ExpectedOutput: Config{Token: "alert-token", Secret: "default-secret"},
		},
		{
			Name: "invalid-merged-config-neither-token-nor-webhook-url",
			Provider: AlertProvider{
				DefaultConfig: Config{Secret: "only-secret"},
				Overrides: []Override{
					{
						Group:  "empty",
						Config: Config{Title: "Only Title"},
					},
				},
			},
			InputGroup:    "empty",
			InputAlert:    alert.Alert{},
			ExpectedError: true,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			cfg, err := s.Provider.GetConfig(s.InputGroup, &s.InputAlert)
			if (err != nil) != s.ExpectedError {
				t.Fatalf("expected error %v, got %v", s.ExpectedError, err)
			}
			if !s.ExpectedError && *cfg != s.ExpectedOutput {
				t.Errorf("expected config %+v, got %+v", s.ExpectedOutput, *cfg)
			}
			valErr := s.Provider.ValidateOverrides(s.InputGroup, &s.InputAlert)
			if (valErr != nil) != s.ExpectedError {
				t.Errorf("expected ValidateOverrides error %v, got %v", s.ExpectedError, valErr)
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

func TestAlertProvider_buildRequestBody(t *testing.T) {
	provider := AlertProvider{}
	ep := &endpoint.Endpoint{Name: "website", Group: "core", URL: "https://example.com"}
	alertDesc := "service is down"
	al := &alert.Alert{Description: &alertDesc, SuccessThreshold: 2, FailureThreshold: 3}
	res := &endpoint.Result{
		Errors: []string{"timeout"},
		ConditionResults: []*endpoint.ConditionResult{
			{Condition: "[STATUS] == 200", Success: false},
		},
	}

	// Case 1: Default title ("Gatus")
	bodyBytes := provider.buildRequestBody(&Config{}, ep, al, res, false)
	var body Body
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if body.MsgType != "markdown" {
		t.Errorf("expected msgtype markdown, got %s", body.MsgType)
	}
	if !strings.Contains(body.Markdown.Title, "[Gatus] 🔴 [TRIGGERED]") {
		t.Errorf("expected title to contain '[Gatus] 🔴 [TRIGGERED]', got: %s", body.Markdown.Title)
	}
	if !strings.Contains(body.Markdown.Text, "### [Gatus] 🔴 **TRIGGERED**: core/website") {
		t.Errorf("expected text to contain '### [Gatus] 🔴 **TRIGGERED**: core/website', got: %s", body.Markdown.Text)
	}

	// Case 2: Custom title
	bodyCustomBytes := provider.buildRequestBody(&Config{Title: "MyAlert"}, ep, al, res, true)
	var bodyCustom Body
	if err := json.Unmarshal(bodyCustomBytes, &bodyCustom); err != nil {
		t.Fatalf("invalid json body: %v", err)
	}
	if !strings.Contains(bodyCustom.Markdown.Title, "[MyAlert] 🟢 [RESOLVED]") {
		t.Errorf("expected title to contain '[MyAlert] 🟢 [RESOLVED]', got: %s", bodyCustom.Markdown.Title)
	}
	if !strings.Contains(bodyCustom.Markdown.Text, "### [MyAlert] 🟢 **RESOLVED**: core/website") {
		t.Errorf("expected text to contain '### [MyAlert] 🟢 **RESOLVED**: core/website', got: %s", bodyCustom.Markdown.Text)
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	desc := "test alert"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:     "triggered-success",
			Provider: AlertProvider{DefaultConfig: Config{Token: "test-token"}},
			Alert:    alert.Alert{Description: &desc, SuccessThreshold: 2, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errcode":0,"errmsg":"ok"}`)),
				}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-dingtalk-errcode-error",
			Provider: AlertProvider{DefaultConfig: Config{Token: "test-token"}},
			Alert:    alert.Alert{Description: &desc, SuccessThreshold: 2, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errcode":310000,"errmsg":"sign not match"}`)),
				}
			}),
			ExpectedError: true,
		},
		{
			Name:     "triggered-malformed-2xx-body-error",
			Provider: AlertProvider{DefaultConfig: Config{Token: "test-token"}},
			Alert:    alert.Alert{Description: &desc, SuccessThreshold: 2, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`<html><body>Bad Gateway</body></html>`)),
				}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved-with-secret",
			Provider: AlertProvider{DefaultConfig: Config{Token: "test-token", Secret: "test-secret"}},
			Alert:    alert.Alert{Description: &desc, SuccessThreshold: 2, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"errcode":0,"errmsg":"ok"}`)),
				}
			}),
			ExpectedError: false,
		},
		{
			Name:     "http-error",
			Provider: AlertProvider{DefaultConfig: Config{Token: "test-token"}},
			Alert:    alert.Alert{Description: &desc, SuccessThreshold: 2, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`Internal Server Error`)),
				}
			}),
			ExpectedError: true,
		},
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: s.MockRoundTripper})
			ep := &endpoint.Endpoint{Name: "test-endpoint", Group: "test-group"}
			res := &endpoint.Result{
				Errors: []string{"connection refused"},
				ConditionResults: []*endpoint.ConditionResult{
					{Condition: "[STATUS] == 200", Success: false},
				},
			}
			err := s.Provider.Send(ep, &s.Alert, res, s.Resolved)
			if (err != nil) != s.ExpectedError {
				t.Errorf("expected error %v, got %v", s.ExpectedError, err)
			}
		})
	}
}

package dingtalk

import (
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
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{Token: ""},
				Group:  "group",
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider token shouldn't have been valid")
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

	// Case 2: Token + Secret (HMAC-SHA256 signature)
	cfg2 := Config{Token: "my-token", Secret: "SEC123456"}
	u2, err := cfg2.GetURL()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(u2, "https://oapi.dingtalk.com/robot/send?access_token=my-token") {
		t.Errorf("URL missing base: %s", u2)
	}
	if !strings.Contains(u2, "&timestamp=") || !strings.Contains(u2, "&sign=") {
		t.Errorf("URL missing signature parameters: %s", u2)
	}

	// Case 3: Neither token nor webhook-url
	cfg3 := Config{}
	_, err = cfg3.GetURL()
	if err == nil {
		t.Error("expected error for empty config, got nil")
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
			Name:     "resolved-with-secret",
			Provider: AlertProvider{DefaultConfig: Config{Token: "test-token", Secret: "SEC123"}},
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

package vonage

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

func TestVonageAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{
		DefaultConfig: Config{
			APIKey:    "test-key",
			APISecret: "test-secret",
			From:      "Gatus",
			To:        []string{"+1234567890"},
		},
	}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestVonageAlertProvider_IsValidWithOverride(t *testing.T) {
	validProvider := AlertProvider{
		DefaultConfig: Config{
			APIKey:    "test-key",
			APISecret: "test-secret",
			From:      "Gatus",
			To:        []string{"+1234567890"},
		},
		Overrides: []Override{
			{
				Group: "test-group",
				Config: Config{
					APIKey:    "override-key",
					APISecret: "override-secret",
					From:      "Override",
					To:        []string{"+9876543210"},
				},
			},
		},
	}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestVonageAlertProvider_IsNotValidWithInvalidOverrideGroup(t *testing.T) {
	invalidProvider := AlertProvider{
		DefaultConfig: Config{
			APIKey:    "test-key",
			APISecret: "test-secret",
			From:      "Gatus",
			To:        []string{"+1234567890"},
		},
		Overrides: []Override{
			{
				Group: "",
				Config: Config{
					APIKey:    "override-key",
					APISecret: "override-secret",
					From:      "Override",
					To:        []string{"+9876543210"},
				},
			},
		},
	}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
}

func TestVonageAlertProvider_IsNotValidWithDuplicateOverrideGroup(t *testing.T) {
	invalidProvider := AlertProvider{
		DefaultConfig: Config{
			APIKey:    "test-key",
			APISecret: "test-secret",
			From:      "Gatus",
			To:        []string{"+1234567890"},
		},
		Overrides: []Override{
			{
				Group: "test-group",
				Config: Config{
					APIKey:    "override-key1",
					APISecret: "override-secret1",
					From:      "Override1",
					To:        []string{"+9876543210"},
				},
			},
			{
				Group: "test-group",
				Config: Config{
					APIKey:    "override-key2",
					APISecret: "override-secret2",
					From:      "Override2",
					To:        []string{"+1234567890"},
				},
			},
		},
	}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
}

func TestVonageAlertProvider_IsValidWithInvalidFrom(t *testing.T) {
	invalidProvider := AlertProvider{
		DefaultConfig: Config{
			APIKey:    "test-key",
			APISecret: "test-secret",
			From:      "",
			To:        []string{"+1234567890"},
		},
	}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
}

func TestVonageAlertProvider_IsValidWithInvalidTo(t *testing.T) {
	invalidProvider := AlertProvider{
		DefaultConfig: Config{
			APIKey:    "test-key",
			APISecret: "test-secret",
			From:      "Gatus",
			To:        []string{},
		},
	}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
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
			Name: "triggered",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
			},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message-count":"1","messages":[{"to":"+1234567890","message-id":"test-id","status":"0","remaining-balance":"10.50","message-price":"0.10","network":"12345"}]}`)),
				}
			}),
			ExpectedError: false,
		},
		{
			Name: "triggered-error-status-code",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
			},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name: "triggered-error-vonage-response",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
			},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message-count":"1","messages":[{"to":"+1234567890","message-id":"","status":"2","error-text":"Missing from param"}]}`)),
				}
			}),
			ExpectedError: true,
		},
		{
			Name: "resolved",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
			},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message-count":"1","messages":[{"to":"+1234567890","message-id":"test-id","status":"0","remaining-balance":"10.40","message-price":"0.10","network":"12345"}]}`)),
				}
			}),
			ExpectedError: false,
		},
		{
			Name: "multiple-recipients",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890", "+0987654321"},
				},
			},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"message-count":"1","messages":[{"to":"+1234567890","message-id":"test-id","status":"0","remaining-balance":"10.30","message-price":"0.10","network":"12345"}]}`)),
				}
			}),
			ExpectedError: false,
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

func TestAlertProvider_buildMessage(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name            string
		Provider        AlertProvider
		Alert           alert.Alert
		Resolved        bool
		ExpectedMessage string
	}{
		{
			Name: "triggered",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
			},
			Alert:           alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        false,
			ExpectedMessage: "TRIGGERED: endpoint-name - description-1",
		},
		{
			Name: "resolved",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
			},
			Alert:           alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        true,
			ExpectedMessage: "RESOLVED: endpoint-name - description-2",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			message := scenario.Provider.buildMessage(
				&scenario.Provider.DefaultConfig,
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
			if message != scenario.ExpectedMessage {
				t.Errorf("expected %s, got %s", scenario.ExpectedMessage, message)
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
			Name: "provider-no-override-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
			},
			InputGroup: "",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				APIKey:    "test-key",
				APISecret: "test-secret",
				From:      "Gatus",
				To:        []string{"+1234567890"},
			},
		},
		{
			Name: "provider-with-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
				Overrides: []Override{
					{
						Group: "test-group",
						Config: Config{
							APIKey:    "group-override-key",
							APISecret: "group-override-secret",
							From:      "GroupOverride",
							To:        []string{"+9876543210"},
						},
					},
				},
			},
			InputGroup: "test-group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				APIKey:    "group-override-key",
				APISecret: "group-override-secret",
				From:      "GroupOverride",
				To:        []string{"+9876543210"},
			},
		},
		{
			Name: "provider-with-group-override-partial",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
				Overrides: []Override{
					{
						Group: "test-group",
						Config: Config{
							To: []string{"+9876543210"},
						},
					},
				},
			},
			InputGroup: "test-group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				APIKey:    "test-key",
				APISecret: "test-secret",
				From:      "Gatus",
				To:        []string{"+9876543210"},
			},
		},
		{
			Name: "provider-with-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
			},
			InputGroup: "",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{
				"api-key":    "override-key",
				"api-secret": "override-secret",
				"from":       "Override",
				"to":         []string{"+9876543210"},
			}},
			ExpectedOutput: Config{
				APIKey:    "override-key",
				APISecret: "override-secret",
				From:      "Override",
				To:        []string{"+9876543210"},
			},
		},
		{
			Name: "provider-with-both-group-and-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
				Overrides: []Override{
					{
						Group: "test-group",
						Config: Config{
							APIKey: "group-override-key",
							From:   "GroupOverride",
						},
					},
				},
			},
			InputGroup: "test-group",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{
				"api-secret": "alert-override-secret",
				"to":         []string{"+9876543210"},
			}},
			ExpectedOutput: Config{
				APIKey:    "group-override-key",
				APISecret: "alert-override-secret",
				From:      "GroupOverride",
				To:        []string{"+9876543210"},
			},
		},
		{
			Name: "provider-with-group-override-no-match",
			Provider: AlertProvider{
				DefaultConfig: Config{
					APIKey:    "test-key",
					APISecret: "test-secret",
					From:      "Gatus",
					To:        []string{"+1234567890"},
				},
				Overrides: []Override{
					{
						Group: "different-group",
						Config: Config{
							APIKey: "group-override-key",
						},
					},
				},
			},
			InputGroup: "test-group",
			InputAlert: alert.Alert{},
			ExpectedOutput: Config{
				APIKey:    "test-key",
				APISecret: "test-secret",
				From:      "Gatus",
				To:        []string{"+1234567890"},
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Error("expected no error, got:", err.Error())
			}
			if got.APIKey != scenario.ExpectedOutput.APIKey {
				t.Errorf("expected APIKey to be %s, got %s", scenario.ExpectedOutput.APIKey, got.APIKey)
			}
			if got.APISecret != scenario.ExpectedOutput.APISecret {
				t.Errorf("expected APISecret to be %s, got %s", scenario.ExpectedOutput.APISecret, got.APISecret)
			}
			if got.From != scenario.ExpectedOutput.From {
				t.Errorf("expected From to be %s, got %s", scenario.ExpectedOutput.From, got.From)
			}
			if len(got.To) != len(scenario.ExpectedOutput.To) {
				t.Errorf("expected To to have length %d, got %d", len(scenario.ExpectedOutput.To), len(got.To))
			} else {
				for i, to := range got.To {
					if to != scenario.ExpectedOutput.To[i] {
						t.Errorf("expected To[%d] to be %s, got %s", i, scenario.ExpectedOutput.To[i], to)
					}
				}
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}


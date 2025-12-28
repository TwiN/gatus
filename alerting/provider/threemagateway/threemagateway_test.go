package threemagateway

import (
	"errors"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		expected error
	}{
		{
			name: "valid config",
			input: Config{
				ApiIdentity:   "12345678",
				ApiAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}},
			},
			expected: nil,
		},
		{
			name: "missing ApiIdentity",
			input: Config{
				ApiAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}},
			},
			expected: ErrApiIdentityMissing,
		},
		{
			name: "missing ApiAuthSecret",
			input: Config{
				ApiIdentity: "12345678",
				Recipients:  []Recipient{{Value: "87654321", Type: RecipientTypeId}},
			},
			expected: ErrApiAuthSecretMissing,
		},
		{
			name: "missing Recipients",
			input: Config{
				ApiIdentity:   "12345678",
				ApiAuthSecret: "authsecret",
			},
			expected: ErrRecipientsMissing,
		},
		{
			name: "invalid ApiIdentity",
			input: Config{
				ApiIdentity:   "invalid-id",
				ApiAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}},
			},
			expected: ErrInvalidThreemaId,
		},
		{
			name: "invalid ID Recipient",
			input: Config{
				ApiIdentity:   "12345678",
				ApiAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "invalid-id", Type: RecipientTypeId}},
			},
			expected: ErrInvalidThreemaId,
		},
		{
			name: "invalid Phone Recipient",
			input: Config{
				ApiIdentity:   "12345678",
				ApiAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "a12345", Type: RecipientTypePhone}},
			},
			expected: ErrInvalidPhoneNumberFormat,
		},
		{
			name: "invalid Email Recipient",
			input: Config{
				ApiIdentity:   "12345678",
				ApiAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "invalid-email", Type: RecipientTypeEmail}},
			},
			expected: ErrInvalidEmailAddressFormat,
		},
		{
			name: "too many Recipients in basic mode",
			input: Config{
				ApiIdentity:   "12345678",
				ApiAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}, {Value: "ABCDEFGH", Type: RecipientTypeId}},
			},
			expected: ErrRecipientsTooMany,
		},
		{
			name: "not implemented E2EE mode",
			input: Config{
				ApiIdentity:   "12345678",
				ApiAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}},
				PrivateKey:    "someprivatekey",
			},
			expected: ErrE2EENotImplemented,
		},
		{
			name: "not implemented E2EE bulk mode",
			input: Config{
				ApiIdentity:   "12345678",
				ApiAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}, {Value: "ABCDEFGH", Type: RecipientTypeId}},
				PrivateKey:    "someprivatekey",
			},
			expected: ErrE2EENotImplemented,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.input.Validate()
			if errors.Is(err, test.expected) == false {
				t.Errorf("expected '%v', got '%v'", test.expected, err)
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	original := Config{
		ApiBaseUrl:    "https://api.threema.ch",
		ApiIdentity:   "12345678",
		Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}},
		ApiAuthSecret: "authsecret",
	}
	override := Config{
		ApiBaseUrl:    "https://custom.api.threema.ch",
		ApiIdentity:   "ABCDEFGH",
		Recipients:    []Recipient{{Value: "HGFEDCBA", Type: RecipientTypeId}},
		ApiAuthSecret: "newauthsecret",
	}

	change := original
	change.Merge(&Config{}) // Merging with empty config should not change anything
	if change.ApiBaseUrl != original.ApiBaseUrl {
		t.Errorf("expected ApiBaseUrl to remain %s, got %s", original.ApiBaseUrl, change.ApiBaseUrl)
	}
	if change.Mode != original.Mode {
		t.Errorf("expected Mode to remain %v, got %v", original.Mode, change.Mode)
	}
	if change.ApiIdentity != original.ApiIdentity {
		t.Errorf("expected ApiIdentity to remain %s, got %s", original.ApiIdentity, change.ApiIdentity)
	}
	if len(change.Recipients) != len(original.Recipients) || change.Recipients[0] != original.Recipients[0] {
		t.Errorf("expected Recipients to remain %v, got %v", original.Recipients, change.Recipients)
	}
	if change.ApiAuthSecret != original.ApiAuthSecret {
		t.Errorf("expected ApiAuthSecret to remain %s, got %s", original.ApiAuthSecret, change.ApiAuthSecret)
	}

	change = original
	change.Merge(&override)
	if change.ApiBaseUrl != override.ApiBaseUrl {
		t.Errorf("expected ApiBaseUrl to be %s, got %s", override.ApiBaseUrl, change.ApiBaseUrl)
	}
	if change.ApiIdentity != override.ApiIdentity {
		t.Errorf("expected ApiIdentity to be %s, got %s", override.ApiIdentity, change.ApiIdentity)
	}
	if len(change.Recipients) != len(override.Recipients) || change.Recipients[0] != override.Recipients[0] {
		t.Errorf("expected Recipients to be %v, got %v", override.Recipients, change.Recipients)
	}
	if change.ApiAuthSecret != override.ApiAuthSecret {
		t.Errorf("expected ApiAuthSecret to be %s, got %s", override.ApiAuthSecret, change.ApiAuthSecret)
	}
}

func TestAlertProvider_Validate(t *testing.T) {
	if err := (&AlertProvider{
		DefaultConfig: Config{
			ApiIdentity:   "12345678",
			ApiAuthSecret: "authsecret",
			Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}},
		},
	}).Validate(); err != nil {
		t.Errorf("expected valid config to not return an error, got %v", err)
	}

	if err := (&AlertProvider{
		DefaultConfig: Config{
			ApiIdentity:   "",
			ApiAuthSecret: "authsecret",
			Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}},
		},
	}).Validate(); !errors.Is(err, ErrApiIdentityMissing) {
		t.Errorf("expected missing ApiIdentity to return %v, got %v", ErrApiIdentityMissing, err)
	}
}

func TestAlertProvider_Send(t *testing.T) {
	testAlertDescription := "Test alert"
	provider := &AlertProvider{
		DefaultConfig: Config{
			ApiIdentity:   "12345678",
			ApiAuthSecret: "authsecret",
			Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeId}},
		},
		Overrides: []Override{{Group: "default", Config: Config{
			ApiIdentity: "HGFEDCBA",
		}}},
	}
	ep := &endpoint.Endpoint{Group: "default"}
	alert := &alert.Alert{Description: &testAlertDescription, ProviderOverride: map[string]any{
		"ApiAuthSecret": "someothersecret",
	}}
	result := &endpoint.Result{Success: false}

	err := provider.Send(ep, alert, result, false)
	if err == nil || strings.Contains(err.Error(), "401") == false {
		t.Error("expected error due to invalid credentials, got nil")
	}
}

func checkStringOccurenceCount(t *testing.T, body, expectedString string, expectedOccurrences int) {
	t.Helper()
	actualOccurrences := strings.Count(body, expectedString)
	if actualOccurrences != expectedOccurrences {
		t.Errorf("expected body to contain '%s' %d times, got %d: %s", expectedString, expectedOccurrences, actualOccurrences, body)
	}
}

func TestAlertProvider_buildMessageBody(t *testing.T) {
	testAlertDescription := "Test alert"
	provider := &AlertProvider{}
	ep := &endpoint.Endpoint{Name: "Test Endpoint", Group: "Custom Group"}
	alert := &alert.Alert{Description: &testAlertDescription}
	result := &endpoint.Result{Success: false, ConditionResults: []*endpoint.ConditionResult{
		{Condition: "[CONNECTED] == true", Success: true},
		{Condition: "[STATUS] == 200", Success: false},
	}, Errors: []string{
		"Failed to connect to host",
	}}

	body := provider.buildMessageBody(ep, alert, result, false)
	checkStringOccurenceCount(t, body, "Custom Group/Test Endpoint", 1)
	checkStringOccurenceCount(t, body, "triggered", 1)
	checkStringOccurenceCount(t, body, "🚨", 1)
	checkStringOccurenceCount(t, body, "✅", 1)
	checkStringOccurenceCount(t, body, "❌", 2)
	checkStringOccurenceCount(t, body, "Failed to connect to host", 1)

	result.Success = true
	result.ConditionResults[1].Success = true
	result.Errors = nil

	body = provider.buildMessageBody(ep, alert, result, true)
	checkStringOccurenceCount(t, body, "Custom Group/Test Endpoint", 1)
	checkStringOccurenceCount(t, body, "resolved", 1)
	checkStringOccurenceCount(t, body, "✅", 1)
	checkStringOccurenceCount(t, body, "🚨", 0)
	checkStringOccurenceCount(t, body, "❌", 0)
}

func TestAlertProvider_prepareRequest(t *testing.T) {
	provider := &AlertProvider{}
	cfg := Config{
		ApiIdentity:   "12345678",
		ApiAuthSecret: "authsecret",
		Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypePhone}},
	}
	cfg.Validate()
	body := "Test message body"

	request, err := provider.prepareRequest(&cfg, body)
	if err != nil {
		t.Errorf("expected no error preparing request, got %v", err)
	}
	if request.Method != "POST" {
		t.Errorf("expected request method to be POST, got %s", request.Method)
	}
	expectedUrl := cfg.ApiBaseUrl + "/send_simple"
	if request.URL.String() != expectedUrl {
		t.Errorf("expected request URL to be %s, got %s", expectedUrl, request.URL.String())
	}
	err = request.ParseForm()
	if err != nil {
		t.Errorf("expected no error parsing form, got %v", err)
	}
	if request.PostForm.Get("from") != cfg.ApiIdentity {
		t.Errorf("expected 'from' to be %s, got %s", cfg.ApiIdentity, request.PostForm.Get("from"))
	}
	if request.PostForm.Get("phone") != cfg.Recipients[0].Value {
		t.Errorf("expected 'phone' to be %s, got %s", cfg.Recipients[0].Value, request.PostForm.Get("to"))
	}
	if request.PostForm.Get("text") != body {
		t.Errorf("expected 'text' to be %s, got %s", body, request.PostForm.Get("text"))
	}
	if request.PostForm.Get("secret") != cfg.ApiAuthSecret {
		t.Errorf("expected 'secret' to be %s, got %s", cfg.ApiAuthSecret, request.PostForm.Get("secret"))
	}

	cfg.Recipients[0] = Recipient{Value: "test@mail.com", Type: RecipientTypeEmail}
	request, err = provider.prepareRequest(&cfg, body)
	if err != nil {
		t.Errorf("expected no error preparing request, got %v", err)
	}
	err = request.ParseForm()
	if err != nil {
		t.Errorf("expected no error parsing form, got %v", err)
	}
	if request.PostForm.Get("email") != cfg.Recipients[0].Value {
		t.Errorf("expected 'email' to be %s, got %s", cfg.Recipients[0].Value, request.PostForm.Get("to"))
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

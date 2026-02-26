package threemagateway

import (
	"errors"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestRecipient_UnmarshalText_And_MarshalText(t *testing.T) {
	scenarios := []struct {
		name        string
		input       string
		expected    Recipient
		expectError error
	}{
		{
			name:        "default recipient type",
			input:       "individual",
			expected:    Recipient{Type: defaultRecipientType, Value: "individual"},
			expectError: nil,
		},
		{
			name:        "empty input",
			input:       "",
			expected:    Recipient{Type: defaultRecipientType, Value: ""},
			expectError: nil,
		},
		{
			name:        "invalid format",
			input:       "type:value:extra",
			expectError: errInvalidRecipientFormat,
		},
		{
			name:        "invalid recipient type",
			input:       "unknown:value",
			expectError: errInvalidRecipientType,
		},
		{
			name:        "valid phone recipient",
			input:       "phone:+1234567890",
			expected:    Recipient{Type: RecipientTypePhone, Value: "+1234567890"},
			expectError: nil,
		},
		{
			name:        "valid email recipient",
			input:       "email:mail@mail.com",
			expected:    Recipient{Type: RecipientTypeEmail, Value: "mail@mail.com"},
			expectError: nil,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			var recipient Recipient
			err := recipient.UnmarshalText([]byte(scenario.input))

			if !errors.Is(err, scenario.expectError) {
				t.Fatalf("expected error for scenario '%s': %v, got: %v", scenario.name, scenario.expectError, err)
			}

			if scenario.expectError == nil && recipient != scenario.expected {
				t.Fatalf("expected recipient for scenario '%s': %+v, got: %+v", scenario.name, scenario.expected, recipient)
			}

			if scenario.expectError == nil {
				marshaled, err := recipient.MarshalText()
				if err != nil {
					t.Fatalf("unexpected error during marshaling for scenario '%s': %v", scenario.name, err)
				}
				expectedMarshaled := scenario.input
				if strings.Contains(scenario.input, ":") == false {
					expectedMarshaled = "id:" + scenario.input
				}
				if string(marshaled) != expectedMarshaled {
					t.Fatalf("expected marshaled text for scenario '%s': %s, got: %s", scenario.name, expectedMarshaled, string(marshaled))
				}
			}
		})
	}
}

func TestRecipient_Validate(t *testing.T) {
	scenarios := []struct {
		name        string
		input       Recipient
		expectError error
	}{
		{
			name:        "empty recipient",
			input:       Recipient{Type: defaultRecipientType, Value: ""},
			expectError: errInvalidRecipientFormat,
		},
		{
			name:        "valid id recipient",
			input:       Recipient{Type: RecipientTypeID, Value: "ABCDEFGH"},
			expectError: nil,
		},
		{
			name:        "valid phone recipient",
			input:       Recipient{Type: RecipientTypePhone, Value: "+1234567890"},
			expectError: nil,
		},
		{
			name:        "invalid phone recipient",
			input:       Recipient{Type: RecipientTypePhone, Value: "123-456-7890"},
			expectError: errInvalidPhoneNumberFormat,
		},
		{
			name:        "valid email recipient",
			input:       Recipient{Type: RecipientTypeEmail, Value: "mail@test.com"},
			expectError: nil,
		},
		{
			name:        "invalid email recipient",
			input:       Recipient{Type: RecipientTypeEmail, Value: "mailtest.com"},
			expectError: errInvalidEmailAddressFormat,
		},
		{
			name:        "invalid recipient type",
			input:       Recipient{Type: RecipientTypeInvalid, Value: "value"},
			expectError: errInvalidRecipientType,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.input.Validate()

			if !errors.Is(err, scenario.expectError) {
				t.Fatalf("expected error for scenario '%s': %v, got: %v", scenario.name, scenario.expectError, err)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		input    Config
		expected error
	}{
		{
			name: "valid config",
			input: Config{
				APIIdentity:   "12345678",
				APIAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}},
			},
			expected: nil,
		},
		{
			name: "missing ApiIdentity",
			input: Config{
				APIAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}},
			},
			expected: errAPIIdentityMissing,
		},
		{
			name: "missing ApiAuthSecret",
			input: Config{
				APIIdentity: "12345678",
				Recipients:  []Recipient{{Value: "87654321", Type: RecipientTypeID}},
			},
			expected: rrrApiAuthSecretMissing,
		},
		{
			name: "missing Recipients",
			input: Config{
				APIIdentity:   "12345678",
				APIAuthSecret: "authsecret",
			},
			expected: errRecipientsMissing,
		},
		{
			name: "invalid ApiIdentity",
			input: Config{
				APIIdentity:   "invalid-id",
				APIAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}},
			},
			expected: errInvalidThreemaID,
		},
		{
			name: "invalid ID Recipient",
			input: Config{
				APIIdentity:   "12345678",
				APIAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "invalid-id", Type: RecipientTypeID}},
			},
			expected: errInvalidThreemaID,
		},
		{
			name: "invalid Phone Recipient",
			input: Config{
				APIIdentity:   "12345678",
				APIAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "a12345", Type: RecipientTypePhone}},
			},
			expected: errInvalidPhoneNumberFormat,
		},
		{
			name: "invalid Email Recipient",
			input: Config{
				APIIdentity:   "12345678",
				APIAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "invalid-email", Type: RecipientTypeEmail}},
			},
			expected: errInvalidEmailAddressFormat,
		},
		{
			name: "too many Recipients in basic mode",
			input: Config{
				APIIdentity:   "12345678",
				APIAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}, {Value: "ABCDEFGH", Type: RecipientTypeID}},
			},
			expected: errRecipientsTooMany,
		},
		{
			name: "not implemented E2EE mode",
			input: Config{
				APIIdentity:   "12345678",
				APIAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}},
				PrivateKey:    "someprivatekey",
			},
			expected: errE2EENotImplemented,
		},
		{
			name: "not implemented E2EE bulk mode",
			input: Config{
				APIIdentity:   "12345678",
				APIAuthSecret: "authsecret",
				Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}, {Value: "ABCDEFGH", Type: RecipientTypeID}},
				PrivateKey:    "someprivatekey",
			},
			expected: errE2EENotImplemented,
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
		APIBaseURL:    "https://api.threema.ch",
		APIIdentity:   "12345678",
		Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}},
		APIAuthSecret: "authsecret",
	}
	override := Config{
		APIBaseURL:    "https://custom.api.threema.ch",
		APIIdentity:   "ABCDEFGH",
		Recipients:    []Recipient{{Value: "HGFEDCBA", Type: RecipientTypeID}},
		APIAuthSecret: "newauthsecret",
	}

	change := original
	change.Merge(&Config{}) // Merging with empty config should not change anything
	if change.APIBaseURL != original.APIBaseURL {
		t.Errorf("expected ApiBaseUrl to remain %s, got %s", original.APIBaseURL, change.APIBaseURL)
	}
	if change.Mode != original.Mode {
		t.Errorf("expected Mode to remain %v, got %v", original.Mode, change.Mode)
	}
	if change.APIIdentity != original.APIIdentity {
		t.Errorf("expected ApiIdentity to remain %s, got %s", original.APIIdentity, change.APIIdentity)
	}
	if len(change.Recipients) != len(original.Recipients) || change.Recipients[0] != original.Recipients[0] {
		t.Errorf("expected Recipients to remain %v, got %v", original.Recipients, change.Recipients)
	}
	if change.APIAuthSecret != original.APIAuthSecret {
		t.Errorf("expected ApiAuthSecret to remain %s, got %s", original.APIAuthSecret, change.APIAuthSecret)
	}

	change = original
	change.Merge(&override)
	if change.APIBaseURL != override.APIBaseURL {
		t.Errorf("expected ApiBaseUrl to be %s, got %s", override.APIBaseURL, change.APIBaseURL)
	}
	if change.APIIdentity != override.APIIdentity {
		t.Errorf("expected ApiIdentity to be %s, got %s", override.APIIdentity, change.APIIdentity)
	}
	if len(change.Recipients) != len(override.Recipients) || change.Recipients[0] != override.Recipients[0] {
		t.Errorf("expected Recipients to be %v, got %v", override.Recipients, change.Recipients)
	}
	if change.APIAuthSecret != override.APIAuthSecret {
		t.Errorf("expected ApiAuthSecret to be %s, got %s", override.APIAuthSecret, change.APIAuthSecret)
	}
}

func TestAlertProvider_Validate(t *testing.T) {
	if err := (&AlertProvider{
		DefaultConfig: Config{
			APIIdentity:   "12345678",
			APIAuthSecret: "authsecret",
			Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}},
		},
	}).Validate(); err != nil {
		t.Errorf("expected valid config to not return an error, got %v", err)
	}

	if err := (&AlertProvider{
		DefaultConfig: Config{
			APIIdentity:   "",
			APIAuthSecret: "authsecret",
			Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}},
		},
	}).Validate(); !errors.Is(err, errAPIIdentityMissing) {
		t.Errorf("expected missing ApiIdentity to return %v, got %v", errAPIIdentityMissing, err)
	}
}

func TestAlertProvider_Send(t *testing.T) {
	testAlertDescription := "Test alert"
	provider := &AlertProvider{
		DefaultConfig: Config{
			APIIdentity:   "12345678",
			APIAuthSecret: "authsecret",
			Recipients:    []Recipient{{Value: "87654321", Type: RecipientTypeID}},
		},
		Overrides: []Override{{Group: "default", Config: Config{
			APIIdentity: "HGFEDCBA",
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
		APIIdentity:   "12345678",
		APIAuthSecret: "authsecret",
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
	expectedUrl := cfg.APIBaseURL + "/send_simple"
	if request.URL.String() != expectedUrl {
		t.Errorf("expected request URL to be %s, got %s", expectedUrl, request.URL.String())
	}
	err = request.ParseForm()
	if err != nil {
		t.Errorf("expected no error parsing form, got %v", err)
	}
	if request.PostForm.Get("from") != cfg.APIIdentity {
		t.Errorf("expected 'from' to be %s, got %s", cfg.APIIdentity, request.PostForm.Get("from"))
	}
	if request.PostForm.Get("phone") != cfg.Recipients[0].Value {
		t.Errorf("expected 'phone' to be %s, got %s", cfg.Recipients[0].Value, request.PostForm.Get("to"))
	}
	if request.PostForm.Get("text") != body {
		t.Errorf("expected 'text' to be %s, got %s", body, request.PostForm.Get("text"))
	}
	if request.PostForm.Get("secret") != cfg.APIAuthSecret {
		t.Errorf("expected 'secret' to be %s, got %s", cfg.APIAuthSecret, request.PostForm.Get("secret"))
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

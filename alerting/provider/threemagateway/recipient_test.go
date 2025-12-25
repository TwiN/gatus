package threemagateway

import (
	"errors"
	"strings"
	"testing"
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
			expectError: ErrInvalidRecipientFormat,
		},
		{
			name:        "invalid recipient type",
			input:       "unknown:value",
			expectError: ErrInvalidRecipientType,
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
			expectError: ErrInvalidRecipientFormat,
		},
		{
			name:        "valid id recipient",
			input:       Recipient{Type: RecipientTypeId, Value: "ABCDEFGH"},
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
			expectError: ErrInvalidPhoneNumberFormat,
		},
		{
			name:        "valid email recipient",
			input:       Recipient{Type: RecipientTypeEmail, Value: "mail@test.com"},
			expectError: nil,
		},
		{
			name:        "invalid email recipient",
			input:       Recipient{Type: RecipientTypeEmail, Value: "mailtest.com"},
			expectError: ErrInvalidEmailAddressFormat,
		},
		{
			name:        "invalid recipient type",
			input:       Recipient{Type: RecipientTypeInvalid, Value: "value"},
			expectError: ErrInvalidRecipientType,
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

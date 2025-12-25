package threemagateway

import (
	"errors"
	"testing"
)

func TestSendMode_UnmarshalText_And_MarshalText(t *testing.T) {
	scenarios := []struct {
		name        string
		input       string
		expected    SendMode
		expectError error
	}{
		{
			name:        "default mode",
			input:       "",
			expected:    SendMode{Value: defaultMode, Type: validModeTypes[defaultMode]},
			expectError: nil,
		},
		{
			name:        "basic mode",
			input:       "basic",
			expected:    SendMode{Value: "basic", Type: ModeTypeBasic},
			expectError: nil,
		},
		{
			name:        "e2ee mode",
			input:       "e2ee",
			expected:    SendMode{Value: "e2ee", Type: ModeTypeE2EE},
			expectError: nil,
		},
		{
			name:        "invalid mode",
			input:       "invalid-mode",
			expected:    SendMode{Value: "invalid-mode", Type: ModeTypeInvalid},
			expectError: ErrModeTypeInvalid,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			var mode SendMode
			err := mode.UnmarshalText([]byte(scenario.input))

			if !errors.Is(err, scenario.expectError) {
				t.Fatalf("expected error for scenario '%s': %v, got: %v", scenario.name, scenario.expectError, err)
			}

			if scenario.expectError == nil && mode != scenario.expected {
				t.Fatalf("expected mode for scenario '%s': %+v, got: %+v", scenario.name, scenario.expected, mode)
			}

			if scenario.expectError == nil {
				marshaled, err := mode.MarshalText()
				if err != nil {
					t.Fatalf("unexpected error during marshaling for scenario '%s': %v", scenario.name, err)
				}
				expectedMarshaled := scenario.input
				if len(scenario.input) == 0 {
					expectedMarshaled = defaultMode
				}
				if string(marshaled) != expectedMarshaled {
					t.Fatalf("expected marshaled mode for scenario '%s': '%s', got: '%s'", scenario.name, expectedMarshaled, string(marshaled))
				}
			}
		})
	}
}

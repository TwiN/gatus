package state

import (
	"testing"
)

func TestState_GetDefaultConfig(t *testing.T) {
	defaultConfig := GetDefaultConfig()
	if len(defaultConfig) != 3 {
		t.Errorf("expected 3 default states, got %d", len(defaultConfig))
	}

	expectedNames := []string{
		DefaultHealthyStateName,
		DefaultUnhealthyStateName,
		DefaultMaintenanceStateName,
	}

	for i, state := range defaultConfig {
		if state.Name != expectedNames[i] {
			t.Errorf("expected state name %s, got %s", expectedNames[i], state.Name)
		}
	}

	for _, state := range defaultConfig {
		if err := state.ValidateAndSetDefaults(); err != nil {
			t.Errorf("default state %s failed validation: %v", state.Name, err)
		}
	}
}

func TestState_ValidateAndSetDefaults(t *testing.T) {
	t.Run("valid state", func(t *testing.T) {
		state := &State{
			Name:     "custom",
			Priority: 10,
		}
		if err := state.ValidateAndSetDefaults(); err != nil {
			t.Errorf("expected valid state, got error: %v", err)
		}
		if state.Name != "custom" {
			t.Errorf("expected name 'custom', got %s", state.Name)
		}
		if state.Priority != 10 {
			t.Errorf("expected priority 10, got %d", state.Priority)
		}
	})

	t.Run("valid name", func(t *testing.T) {
		state := &State{
			Name:     "valid-name_123",
			Priority: 5,
		}
		if err := state.ValidateAndSetDefaults(); err != nil {
			t.Errorf("expected valid state, got error: %v", err)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		state := &State{
			Name:     "",
			Priority: 10,
		}
		err := state.ValidateAndSetDefaults()
		if err != ErrInvalidName {
			t.Errorf("expected ErrInvalidName, got %v", err)
		}
	})

	t.Run("whitespace name", func(t *testing.T) {
		state := &State{
			Name:     "   ",
			Priority: 10,
		}
		err := state.ValidateAndSetDefaults()
		if err != ErrInvalidName {
			t.Errorf("expected ErrInvalidName, got %v", err)
		}
	})

	t.Run("special character name", func(t *testing.T) {
		state := &State{
			Name:     "custom@state",
			Priority: 10,
		}
		err := state.ValidateAndSetDefaults()
		if err != ErrInvalidName {
			t.Errorf("expected ErrInvalidName, got %v", err)
		}
	})

	t.Run("uppercase name", func(t *testing.T) {
		state := &State{
			Name:     "CustomState",
			Priority: 10,
		}
		err := state.ValidateAndSetDefaults()
		if err != ErrInvalidName {
			t.Errorf("expected ErrInvalidName, got %v", err)
		}
	})

	t.Run("zero priority", func(t *testing.T) {
		state := &State{
			Name:     "custom",
			Priority: 0,
		}
		err := state.ValidateAndSetDefaults()
		if err == ErrInvalidPriority {
			t.Errorf("did not expect ErrInvalidPriority for zero priority, got %v", err)
		}
		if state.Priority != 0 {
			t.Errorf("expected priority to be set to 0, got %d", state.Priority)
		}
	})

	t.Run("negative priority", func(t *testing.T) {
		state := &State{
			Name:     "custom",
			Priority: -1,
		}
		err := state.ValidateAndSetDefaults()
		if err != ErrInvalidPriority {
			t.Errorf("expected ErrInvalidPriority, got %v", err)
		}
	})
}

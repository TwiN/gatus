package ssh

import (
	"errors"
	"testing"
)

func TestSSH_validate(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err != nil {
		t.Error("didn't expect an error")
	}
	cfg.Username = "username"
	if err := cfg.Validate(); err == nil {
		t.Error("expected an error")
	} else if !errors.Is(err, ErrEndpointWithoutSSHPassword) {
		t.Errorf("expected error to be '%v', got '%v'", ErrEndpointWithoutSSHPassword, err)
	}
	cfg.Password = "password"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got '%v'", err)
	}
}

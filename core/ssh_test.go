package core

import (
	"errors"
	"testing"
)

func TestSSH_validate(t *testing.T) {
	ssh := &SSH{}
	if err := ssh.validate(); err == nil {
		t.Error("expected an error")
	} else if !errors.Is(err, ErrEndpointWithoutSSHUsername) {
		t.Errorf("expected error to be '%v', got '%v'", ErrEndpointWithoutSSHUsername, err)
	}
	ssh.Username = "username"
	if err := ssh.validate(); err == nil {
		t.Error("expected an error")
	} else if !errors.Is(err, ErrEndpointWithoutSSHPassword) {
		t.Errorf("expected error to be '%v', got '%v'", ErrEndpointWithoutSSHPassword, err)
	}
	ssh.Password = "password"
	if err := ssh.validate(); err != nil {
		t.Errorf("expected no error, got '%v'", err)
	}
}

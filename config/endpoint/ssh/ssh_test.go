package ssh

import (
	"errors"
	"testing"
)

func TestSSH_validatePasswordCfg(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err != nil {
		t.Error("didn't expect an error")
	}
	cfg.Username = "username"
	if err := cfg.Validate(); err == nil {
		t.Error("expected an error")
	} else if !errors.Is(err, ErrEndpointWithoutSSHAuth) {
		t.Errorf("expected error to be '%v', got '%v'", ErrEndpointWithoutSSHAuth, err)
	}
	cfg.Password = "password"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got '%v'", err)
	}
}

func TestSSH_validatePrivateKeyCfg(t *testing.T) {
	t.Run("fail when username missing but private key provided", func(t *testing.T) {
		cfg := &Config{PrivateKey: "-----BEGIN"}
		if err := cfg.Validate(); !errors.Is(err, ErrEndpointWithoutSSHUsername) {
			t.Fatalf("expected ErrEndpointWithoutSSHUsername, got %v", err)
		}
	})
	t.Run("success when username with private key", func(t *testing.T) {
		cfg := &Config{Username: "user", PrivateKey: "-----BEGIN"}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

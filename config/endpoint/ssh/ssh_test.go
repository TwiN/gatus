package ssh

import (
	"errors"
	"strings"
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

func TestSSH_validateHostKey(t *testing.T) {
	const validHostKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"
	t.Run("success when host key with username and password", func(t *testing.T) {
		cfg := &Config{Username: "user", Password: "password", HostKey: validHostKey}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
	t.Run("fail when host key is malformed", func(t *testing.T) {
		cfg := &Config{Username: "user", Password: "password", HostKey: "not-a-key"}
		if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "invalid ssh.host-key") {
			t.Fatalf("expected invalid ssh.host-key error, got %v", err)
		}
	})
	t.Run("fail when host key without credentials", func(t *testing.T) {
		cfg := &Config{HostKey: validHostKey}
		if err := cfg.Validate(); !errors.Is(err, ErrEndpointWithHostKeyWithoutSSHAuth) {
			t.Fatalf("expected ErrEndpointWithHostKeyWithoutSSHAuth, got %v", err)
		}
	})
}

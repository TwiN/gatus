package sftp

import (
	"errors"
	"testing"
)

func TestSFTP_validatePasswordCfg(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Validate(); err != nil {
		t.Error("didn't expect an error")
	}
	cfg.Username = "username"
	if err := cfg.Validate(); err == nil {
		t.Error("expected an error")
	} else if !errors.Is(err, ErrEndpointWithoutSFTPAuth) {
		t.Errorf("expected error to be '%v', got '%v'", ErrEndpointWithoutSFTPAuth, err)
	}
	cfg.Password = "password"
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got '%v'", err)
	}
}

func TestSFTP_validatePrivateKeyCfg(t *testing.T) {
	t.Run("fail when username missing but private key provided", func(t *testing.T) {
		cfg := &Config{PrivateKey: "-----BEGIN"}
		if err := cfg.Validate(); !errors.Is(err, ErrEndpointWithoutSFTPUsername) {
			t.Fatalf("expected ErrEndpointWithoutSFTPUsername, got %v", err)
		}
	})
	t.Run("success when username with private key", func(t *testing.T) {
		cfg := &Config{Username: "user", PrivateKey: "-----BEGIN"}
		if err := cfg.Validate(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

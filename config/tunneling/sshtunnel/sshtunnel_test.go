package sshtunnel

import (
	"testing"
)

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid SSH config with private key",
			config: &Config{
				Type:       "SSH",
				Host:       "example.com",
				Username:   "test",
				PrivateKey: "-----BEGIN RSA PRIVATE KEY-----\ntest\n-----END RSA PRIVATE KEY-----",
			},
			wantErr: false,
		},
		{
			name: "valid SSH config with password",
			config: &Config{
				Type:     "SSH",
				Host:     "example.com",
				Username: "test",
				Password: "secret",
			},
			wantErr: false,
		},
		{
			name: "valid SSH config with custom port",
			config: &Config{
				Type:     "SSH",
				Host:     "example.com",
				Port:     2222,
				Username: "test",
				Password: "secret",
			},
			wantErr: false,
		},
		{
			name: "sets default port 22",
			config: &Config{
				Type:     "SSH",
				Host:     "example.com",
				Username: "test",
				Password: "secret",
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			config: &Config{
				Type:     "INVALID",
				Host:     "example.com",
				Username: "test",
				Password: "secret",
			},
			wantErr: true,
			errMsg:  "unsupported tunnel type: INVALID",
		},
		{
			name: "missing host",
			config: &Config{
				Type:     "SSH",
				Username: "test",
				Password: "secret",
			},
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name: "missing username",
			config: &Config{
				Type:     "SSH",
				Host:     "example.com",
				Password: "secret",
			},
			wantErr: true,
			errMsg:  "username is required",
		},
		{
			name: "missing authentication",
			config: &Config{
				Type:     "SSH",
				Host:     "example.com",
				Username: "test",
			},
			wantErr: true,
			errMsg:  "either private-key or password is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalPort := tt.config.Port
			err := tt.config.ValidateAndSetDefaults()
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateAndSetDefaults() expected error but got none")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateAndSetDefaults() error = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("ValidateAndSetDefaults() unexpected error = %v", err)
				return
			}
			// Check that default port is set
			if originalPort == 0 && tt.config.Port != 22 {
				t.Errorf("ValidateAndSetDefaults() expected default port 22, got %d", tt.config.Port)
			}
		})
	}
}

func TestNew(t *testing.T) {
	config := &Config{
		Type:     "SSH",
		Host:     "example.com",
		Username: "test",
		Password: "secret",
	}
	tunnel := New(config)
	if tunnel == nil {
		t.Error("New() returned nil")
		return
	}
	if tunnel.config != config {
		t.Error("New() did not set config correctly")
	}
}

func TestSSHTunnel_Close(t *testing.T) {
	config := &Config{
		Type:     "SSH",
		Host:     "example.com",
		Username: "test",
		Password: "secret",
	}
	tunnel := New(config)
	// Test closing when no client is set
	err := tunnel.Close()
	if err != nil {
		t.Errorf("Close() with no client returned error: %v", err)
	}
	// Test closing multiple times
	err = tunnel.Close()
	if err != nil {
		t.Errorf("Close() called twice returned error: %v", err)
	}
}

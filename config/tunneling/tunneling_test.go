package tunneling

import (
	"testing"

	"github.com/TwiN/gatus/v5/config/tunneling/sshtunnel"
)

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with SSH tunnel",
			config: &Config{
				Tunnels: map[string]*sshtunnel.Config{
					"test": {
						Type:     "SSH",
						Host:     "example.com",
						Username: "test",
						Password: "secret",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "multiple valid tunnels",
			config: &Config{
				Tunnels: map[string]*sshtunnel.Config{
					"tunnel1": {
						Type:       "SSH",
						Host:       "host1.com",
						Username:   "user1",
						PrivateKey: "key1",
					},
					"tunnel2": {
						Type:     "SSH",
						Host:     "host2.com",
						Username: "user2",
						Password: "pass2",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid tunnel config",
			config: &Config{
				Tunnels: map[string]*sshtunnel.Config{
					"invalid": {
						Type:     "INVALID",
						Host:     "example.com",
						Username: "test",
						Password: "secret",
					},
				},
			},
			wantErr: true,
			errMsg:  "tunnel 'invalid': unsupported tunnel type: INVALID",
		},
		{
			name: "missing host in tunnel",
			config: &Config{
				Tunnels: map[string]*sshtunnel.Config{
					"nohost": {
						Type:     "SSH",
						Username: "test",
						Password: "secret",
					},
				},
			},
			wantErr: true,
			errMsg:  "tunnel 'nohost': host is required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			// Check that connections map is initialized
			if tt.config != nil && tt.config.connections == nil {
				t.Error("ValidateAndSetDefaults() did not initialize connections map")
			}
		})
	}
}

func TestConfig_GetTunnel(t *testing.T) {
	config := &Config{
		Tunnels: map[string]*sshtunnel.Config{
			"test": {
				Type:     "SSH",
				Host:     "example.com",
				Username: "test",
				Password: "secret",
			},
		},
	}
	err := config.ValidateAndSetDefaults()
	if err != nil {
		t.Fatalf("ValidateAndSetDefaults() failed: %v", err)
	}
	// Test getting existing tunnel
	tunnel1, err := config.GetTunnel("test")
	if err != nil {
		t.Errorf("GetTunnel() error = %v", err)
		return
	}
	if tunnel1 == nil {
		t.Error("GetTunnel() returned nil tunnel")
		return
	}
	// Test getting same tunnel again (should return same instance)
	tunnel2, err := config.GetTunnel("test")
	if err != nil {
		t.Errorf("GetTunnel() second call error = %v", err)
		return
	}
	if tunnel1 != tunnel2 {
		t.Error("GetTunnel() should return same instance for same tunnel name")
	}
	// Test getting non-existent tunnel
	_, err = config.GetTunnel("nonexistent")
	if err == nil {
		t.Error("GetTunnel() expected error for non-existent tunnel")
		return
	}
	expectedErr := "tunnel 'nonexistent' not found in configuration"
	if err.Error() != expectedErr {
		t.Errorf("GetTunnel() error = %v, want %v", err.Error(), expectedErr)
	}
}

func TestConfig_Close(t *testing.T) {
	// Test closing config with tunnels
	config := &Config{
		Tunnels: map[string]*sshtunnel.Config{
			"test1": {
				Type:     "SSH",
				Host:     "example1.com",
				Username: "test",
				Password: "secret",
			},
			"test2": {
				Type:     "SSH",
				Host:     "example2.com",
				Username: "test",
				Password: "secret",
			},
		},
	}
	err := config.ValidateAndSetDefaults()
	if err != nil {
		t.Fatalf("ValidateAndSetDefaults() failed: %v", err)
	}
	// Create some tunnels
	_, err = config.GetTunnel("test1")
	if err != nil {
		t.Fatalf("GetTunnel() failed: %v", err)
	}
	_, err = config.GetTunnel("test2")
	if err != nil {
		t.Fatalf("GetTunnel() failed: %v", err)
	}
	// Test closing
	err = config.Close()
	if err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
	// Verify connections map is empty
	if len(config.connections) != 0 {
		t.Errorf("Close() did not clear connections map, got %d connections", len(config.connections))
	}
}

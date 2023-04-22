package web

import (
	"testing"

	"github.com/TwiN/gatus/v5/test"
)

func TestGetDefaultConfig(t *testing.T) {
	defaultConfig := GetDefaultConfig()
	if defaultConfig.Port != DefaultPort {
		t.Error("expected default config to have the default port")
	}
	if defaultConfig.Address != DefaultAddress {
		t.Error("expected default config to have the default address")
	}
	if defaultConfig.Tls != (TLSConfig{}) {
		t.Error("expected default config to have TLS disabled")
	}
}

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	scenarios := []struct {
		name            string
		cfg             *Config
		expectedAddress string
		expectedPort    int
		expectedErr     bool
	}{
		{
			name:            "no-explicit-config",
			cfg:             &Config{},
			expectedAddress: "0.0.0.0",
			expectedPort:    8080,
			expectedErr:     false,
		},
		{
			name:        "invalid-port",
			cfg:         &Config{Port: 100000000},
			expectedErr: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.cfg.ValidateAndSetDefaults()
			if (err != nil) != scenario.expectedErr {
				t.Errorf("expected the existence of an error to be %v, got %v", scenario.expectedErr, err)
				return
			}
			if !scenario.expectedErr {
				if scenario.cfg.Port != scenario.expectedPort {
					t.Errorf("expected port to be %d, got %d", scenario.expectedPort, scenario.cfg.Port)
				}
				if scenario.cfg.Address != scenario.expectedAddress {
					t.Errorf("expected address to be %s, got %s", scenario.expectedAddress, scenario.cfg.Address)
				}
			}
		})
	}
}

func TestConfig_SocketAddress(t *testing.T) {
	web := &Config{
		Address: "0.0.0.0",
		Port:    8081,
	}
	if web.SocketAddress() != "0.0.0.0:8081" {
		t.Errorf("expected %s, got %s", "0.0.0.0:8081", web.SocketAddress())
	}
}

func TestConfig_TLSConfig(t *testing.T) {
	privateKeyPath, publicKeyPath := test.UnsafeSelfSignedCertificates(t.TempDir())

	scenarios := []struct {
		name        string
		cfg         *Config
		expectedErr bool
	}{
		{
			name:        "including TLS",
			cfg:         &Config{Tls: (TLSConfig{CertificateFile: publicKeyPath, PrivateKeyFile: privateKeyPath})},
			expectedErr: false,
		},
		{
			name:        "TLS with missing crt file",
			cfg:         &Config{Tls: (TLSConfig{CertificateFile: "doesnotexist", PrivateKeyFile: privateKeyPath})},
			expectedErr: true,
		},
		{
			name:        "TLS with missing key file",
			cfg:         &Config{Tls: (TLSConfig{CertificateFile: publicKeyPath, PrivateKeyFile: "doesnotexist"})},
			expectedErr: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			cfg, err := scenario.cfg.TLSConfig()
			if (err != nil) != scenario.expectedErr {
				t.Errorf("expected the existence of an error to be %v, got %v", scenario.expectedErr, err)
				return
			}
			if !scenario.expectedErr {
				if cfg == nil {
					t.Error("TLS configuration was not correctly loaded although no error was returned")
				}
			}
		})
	}
}

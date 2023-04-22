package web

import (
	"testing"
)

func TestGetDefaultConfig(t *testing.T) {
	defaultConfig := GetDefaultConfig()
	if defaultConfig.Port != DefaultPort {
		t.Error("expected default config to have the default port")
	}
	if defaultConfig.Address != DefaultAddress {
		t.Error("expected default config to have the default address")
	}
	if defaultConfig.TLS != nil {
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
	scenarios := []struct {
		name        string
		cfg         *Config
		expectedErr bool
	}{
		{
			name:        "good-tls-config",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../../testdata/cert.pem", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedErr: false,
		},
		{
			name:        "missing-crt-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "doesnotexist", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedErr: true,
		},
		{
			name:        "bad-crt-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../../testdata/badcert.pem", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedErr: true,
		},
		{
			name:        "missing-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../../testdata/cert.pem", PrivateKeyFile: "doesnotexist"}},
			expectedErr: true,
		},
		{
			name:        "bad-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../../testdata/cert.pem", PrivateKeyFile: "../../testdata/badcert.key"}},
			expectedErr: true,
		},
		{
			name:        "bad-cert-and-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../../testdata/badcert.pem", PrivateKeyFile: "../../testdata/badcert.key"}},
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
				if scenario.cfg.TLS.tlsConfig == nil {
					t.Error("TLS configuration was not correctly loaded although no error was returned")
				}
			}
		})
	}
}

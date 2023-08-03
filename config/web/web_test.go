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
		{
			name:            "with-good-tls-config",
			cfg:             &Config{Port: 443, TLS: &TLSConfig{CertificateFile: "../../testdata/cert.pem", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedAddress: "0.0.0.0",
			expectedPort:    443,
			expectedErr:     false,
		},
		{
			name:            "with-bad-tls-config",
			cfg:             &Config{Port: 443, TLS: &TLSConfig{CertificateFile: "../../testdata/badcert.pem", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedAddress: "0.0.0.0",
			expectedPort:    443,
			expectedErr:     true,
		},
		{
			name:            "with-partial-tls-config",
			cfg:             &Config{Port: 443, TLS: &TLSConfig{CertificateFile: "", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedAddress: "0.0.0.0",
			expectedPort:    443,
			expectedErr:     true,
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

func TestConfig_isValid(t *testing.T) {
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
			name:        "missing-certificate-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "doesnotexist", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedErr: true,
		},
		{
			name:        "bad-certificate-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../../testdata/badcert.pem", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedErr: true,
		},
		{
			name:        "no-certificate-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedErr: true,
		},
		{
			name:        "missing-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../../testdata/cert.pem", PrivateKeyFile: "doesnotexist"}},
			expectedErr: true,
		},
		{
			name:        "no-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../../testdata/cert.pem", PrivateKeyFile: ""}},
			expectedErr: true,
		},
		{
			name:        "bad-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../../testdata/cert.pem", PrivateKeyFile: "../../testdata/badcert.key"}},
			expectedErr: true,
		},
		{
			name:        "bad-certificate-and-private-key-file",
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
				if scenario.cfg.TLS.isValid() != nil {
					t.Error("cfg.TLS.isValid() returned an error even though no error was expected")
				}
			}
		})
	}
}

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
	if defaultConfig.ReadBufferSize != DefaultReadBufferSize {
		t.Error("expected default config to have the default read buffer size")
	}
	if defaultConfig.TLS != nil {
		t.Error("expected default config to have TLS disabled")
	}
}

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	scenarios := []struct {
		name                   string
		cfg                    *Config
		expectedAddress        string
		expectedPort           int
		expectedReadBufferSize int
		expectedErr            bool
	}{
		{
			name:                   "no-explicit-config",
			cfg:                    &Config{},
			expectedAddress:        "0.0.0.0",
			expectedPort:           8080,
			expectedReadBufferSize: 8192,
			expectedErr:            false,
		},
		{
			name:        "invalid-port",
			cfg:         &Config{Port: 100000000},
			expectedErr: true,
		},
		{
			name:                   "read-buffer-size-below-minimum",
			cfg:                    &Config{ReadBufferSize: 1024},
			expectedAddress:        "0.0.0.0",
			expectedPort:           8080,
			expectedReadBufferSize: MinimumReadBufferSize, // minimum is 4096, default is 8192.
			expectedErr:            false,
		},
		{
			name:                   "read-buffer-size-at-minimum",
			cfg:                    &Config{ReadBufferSize: MinimumReadBufferSize},
			expectedAddress:        "0.0.0.0",
			expectedPort:           8080,
			expectedReadBufferSize: 4096,
			expectedErr:            false,
		},
		{
			name:                   "custom-read-buffer-size",
			cfg:                    &Config{ReadBufferSize: 65536},
			expectedAddress:        "0.0.0.0",
			expectedPort:           8080,
			expectedReadBufferSize: 65536,
			expectedErr:            false,
		},
		{
			name:                   "with-good-tls-config",
			cfg:                    &Config{Port: 443, TLS: &TLSConfig{CertificateFile: "../../testdata/cert.pem", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedAddress:        "0.0.0.0",
			expectedPort:           443,
			expectedReadBufferSize: 8192,
			expectedErr:            false,
		},
		{
			name:                   "with-bad-tls-config",
			cfg:                    &Config{Port: 443, TLS: &TLSConfig{CertificateFile: "../../testdata/badcert.pem", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedAddress:        "0.0.0.0",
			expectedPort:           443,
			expectedReadBufferSize: 8192,
			expectedErr:            true,
		},
		{
			name:                   "with-partial-tls-config",
			cfg:                    &Config{Port: 443, TLS: &TLSConfig{CertificateFile: "", PrivateKeyFile: "../../testdata/cert.key"}},
			expectedAddress:        "0.0.0.0",
			expectedPort:           443,
			expectedReadBufferSize: 8192,
			expectedErr:            true,
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
					t.Errorf("expected Port to be %d, got %d", scenario.expectedPort, scenario.cfg.Port)
				}
				if scenario.cfg.ReadBufferSize != scenario.expectedReadBufferSize {
					t.Errorf("expected ReadBufferSize to be %d, got %d", scenario.expectedReadBufferSize, scenario.cfg.ReadBufferSize)
				}
				if scenario.cfg.Address != scenario.expectedAddress {
					t.Errorf("expected Address to be %s, got %s", scenario.expectedAddress, scenario.cfg.Address)
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

func TestCORSConfig_ValidateAndSetDefaults(t *testing.T) {
	t.Run("nil-cors", func(t *testing.T) {
		cfg := &Config{}
		if err := cfg.ValidateAndSetDefaults(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CORS != nil {
			t.Errorf("expected CORS to be nil")
		}
	})

	t.Run("empty-allow-origins-returns-error", func(t *testing.T) {
		cfg := &Config{
			CORS: &CORSConfig{},
		}
		err := cfg.ValidateAndSetDefaults()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "invalid cors config: allow-origins must be specified when cors is enabled" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("wildcard-origins-with-credentials-returns-error", func(t *testing.T) {
		cfg := &Config{
			CORS: &CORSConfig{
				AllowOrigins:     "*",
				AllowCredentials: true,
			},
		}
		err := cfg.ValidateAndSetDefaults()
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "invalid cors config: allow-origins cannot contain '*' when allow-credentials is true" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("valid-config-sets-defaults", func(t *testing.T) {
		cfg := &Config{
			CORS: &CORSConfig{
				AllowOrigins: "https://example.com",
			},
		}
		if err := cfg.ValidateAndSetDefaults(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CORS.AllowMethods != "GET,POST,OPTIONS" {
			t.Errorf("expected default AllowMethods, got %s", cfg.CORS.AllowMethods)
		}
		if cfg.CORS.AllowHeaders != "Origin,Content-Type,Accept,Authorization" {
			t.Errorf("expected default AllowHeaders, got %s", cfg.CORS.AllowHeaders)
		}
	})

	t.Run("valid-config-preserves-custom-values", func(t *testing.T) {
		cfg := &Config{
			CORS: &CORSConfig{
				AllowOrigins:     "https://example.com",
				AllowMethods:     "GET",
				AllowHeaders:     "Authorization",
				AllowCredentials: true,
				MaxAge:           300,
			},
		}
		if err := cfg.ValidateAndSetDefaults(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.CORS.AllowMethods != "GET" {
			t.Errorf("expected AllowMethods to be GET, got %s", cfg.CORS.AllowMethods)
		}
		if cfg.CORS.AllowHeaders != "Authorization" {
			t.Errorf("expected AllowHeaders to be Authorization, got %s", cfg.CORS.AllowHeaders)
		}
		if !cfg.CORS.AllowCredentials {
			t.Error("expected AllowCredentials to be true")
		}
		if cfg.CORS.MaxAge != 300 {
			t.Errorf("expected MaxAge to be 300, got %d", cfg.CORS.MaxAge)
		}
	})
}


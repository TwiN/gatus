package web

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"os"
	"testing"
	"time"
)

func TestGetDefaultConfig(t *testing.T) {
	defaultConfig := GetDefaultConfig()
	if defaultConfig.Port != DefaultPort {
		t.Error("expected default config to have the default port")
	}
	if defaultConfig.Address != DefaultAddress {
		t.Error("expected default config to have the default address")
	}
	if defaultConfig.Tls != (TlsConfig{}) {
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
	privateKeyPath, publicKeyPath := unsafeSelfSignedCertificates(t.TempDir())

	scenarios := []struct {
		name        string
		cfg         *Config
		expectedErr bool
	}{
		{
			name:        "including TLS",
			cfg:         &Config{Tls: (TlsConfig{CertFile: publicKeyPath, KeyFile: privateKeyPath})},
			expectedErr: false,
		},
		{
			name:        "TLS with missing crt file",
			cfg:         &Config{Tls: (TlsConfig{CertFile: "doesnotexist", KeyFile: privateKeyPath})},
			expectedErr: true,
		},
		{
			name:        "TLS with missing key file",
			cfg:         &Config{Tls: (TlsConfig{CertFile: publicKeyPath, KeyFile: "doesnotexist"})},
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

// unsafeSelfSignedCertificates creates a pair of simple test certificates
func unsafeSelfSignedCertificates(testfolder string) (privateKeyPath string, publicKeyPath string) {
	privateKeyPath = fmt.Sprintf("%s/cert.key", testfolder)
	publicKeyPath = fmt.Sprintf("%s/cert.pem", testfolder)

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generatekey: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1234),
		Subject: pkix.Name{
			Organization: []string{"Gatus test"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	certOut, err := os.Create(publicKeyPath)
	if err != nil {
		log.Fatalf("Failed to open cert.pem for writing: %v", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		log.Fatalf("Failed to write data to cert.pem: %v", err)
	}
	if err := certOut.Close(); err != nil {
		log.Fatalf("Error closing cert.pem: %v", err)
	}

	keyOut, err := os.OpenFile(privateKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Failed to open %s for writing: %v", privateKeyPath, err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		log.Fatalf("Unable to marshal private key: %v", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		log.Fatalf("Failed to write data to key.pem: %v", err)
	}
	if err := keyOut.Close(); err != nil {
		log.Fatalf("Error closing key.pem: %v", err)
	}
	log.Print("wrote key.pem\n")

	return
}

package web

import (
	"crypto/tls"
	"fmt"
	"math"
)

const (
	// DefaultAddress is the default address the application will bind to
	DefaultAddress = "0.0.0.0"

	// DefaultPort is the default port the application will listen on
	DefaultPort = 8080
)

// Config is the structure which supports the configuration of the endpoint
// which provides access to the web frontend
type Config struct {
	// Address to listen on (defaults to 0.0.0.0 specified by DefaultAddress)
	Address string `yaml:"address"`

	// Port to listen on (default to 8080 specified by DefaultPort)
	Port int `yaml:"port"`

	// TLS configuration
	Tls TlsConfig `yaml:"tls"`

	tlsConfig      *tls.Config
	tlsConfigError error
}

type TlsConfig struct {

	// Optional public certificate for TLS in PEM format.
	CertFile string `yaml:"certificate-file,omitempty"`

	// Optional private key file for TLS in PEM format.
	KeyFile string `yaml:"private-key-file,omitempty"`
}

// GetDefaultConfig returns a Config struct with the default values
func GetDefaultConfig() *Config {
	return &Config{Address: DefaultAddress, Port: DefaultPort}
}

// ValidateAndSetDefaults validates the web configuration and sets the default values if necessary.
func (web *Config) ValidateAndSetDefaults() error {
	// Validate the Address
	if len(web.Address) == 0 {
		web.Address = DefaultAddress
	}
	// Validate the Port
	if web.Port == 0 {
		web.Port = DefaultPort
	} else if web.Port < 0 || web.Port > math.MaxUint16 {
		return fmt.Errorf("invalid port: value should be between %d and %d", 0, math.MaxUint16)
	}
	// Try to load the TLS certificates
	_, err := web.TLSConfig()
	if err != nil {
		return fmt.Errorf("invalid tls config: %s", err)
	}
	return nil
}

// SocketAddress returns the combination of the Address and the Port
func (web *Config) SocketAddress() string {
	return fmt.Sprintf("%s:%d", web.Address, web.Port)
}

// TLSConfig returns a tls.Config object for serving over an encrypted channel
func (web *Config) TLSConfig() (*tls.Config, error) {
	if web.tlsConfig == nil && len(web.Tls.CertFile) > 0 && len(web.Tls.KeyFile) > 0 {
		web.loadTLSConfig()
	}
	return web.tlsConfig, web.tlsConfigError
}

func (web *Config) loadTLSConfig() {
	cer, err := tls.LoadX509KeyPair(web.Tls.CertFile, web.Tls.KeyFile)
	if err != nil {
		web.tlsConfigError = err
	}
	web.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cer}}
}

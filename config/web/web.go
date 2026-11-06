package web

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"strings"
)

const (
	// DefaultAddress is the default address the application will bind to
	DefaultAddress = "0.0.0.0"

	// DefaultPort is the default port the application will listen on
	DefaultPort = 8080

	// DefaultReadBufferSize is the default value for ReadBufferSize
	DefaultReadBufferSize = 8192

	// MinimumReadBufferSize is the minimum value for ReadBufferSize, and also the default value set
	// for fiber.Config.ReadBufferSize
	MinimumReadBufferSize = 4096
)

// Config is the structure which supports the configuration of the server listening to requests
type Config struct {
	// Address to listen on (defaults to 0.0.0.0 specified by DefaultAddress)
	Address string `yaml:"address"`

	// Port to listen on (default to 8080 specified by DefaultPort)
	Port int `yaml:"port"`

	// ReadBufferSize sets fiber.Config.ReadBufferSize, which is the buffer size for reading requests coming from a
	// single connection and also acts as a limit for the maximum header size.
	//
	// If you're getting occasional "Request Header Fields Too Large", you may want to try increasing this value.
	//
	// Defaults to DefaultReadBufferSize
	ReadBufferSize int `yaml:"read-buffer-size,omitempty"`

	// TLS configuration (optional)
	TLS *TLSConfig `yaml:"tls,omitempty"`

	// CORS configuration (optional)
	CORS *CORSConfig `yaml:"cors,omitempty"`
}

type TLSConfig struct {
	// CertificateFile is the public certificate for TLS in PEM format.
	CertificateFile string `yaml:"certificate-file,omitempty"`

	// PrivateKeyFile is the private key file for TLS in PEM format.
	PrivateKeyFile string `yaml:"private-key-file,omitempty"`
}

// CORSConfig is the configuration for CORS
type CORSConfig struct {
	// AllowOrigins is a comma-separated list of origins that may access the resource.
	// Example: "https://dashboard.example.com,https://other.example.com"
	AllowOrigins string `yaml:"allow-origins"`

	// AllowMethods is a comma-separated list of methods. Defaults to "GET,POST,OPTIONS" if empty.
	AllowMethods string `yaml:"allow-methods,omitempty"`

	// AllowHeaders is a comma-separated list of headers. Defaults to "Origin,Content-Type,Accept,Authorization" if empty.
	AllowHeaders string `yaml:"allow-headers,omitempty"`

	// AllowCredentials toggles Access-Control-Allow-Credentials. Defaults to false.
	AllowCredentials bool `yaml:"allow-credentials,omitempty"`

	// MaxAge sets Access-Control-Max-Age in seconds. Defaults to 0 (no caching).
	MaxAge int `yaml:"max-age,omitempty"`
}

// GetDefaultConfig returns a Config struct with the default values
func GetDefaultConfig() *Config {
	return &Config{
		Address:        DefaultAddress,
		Port:           DefaultPort,
		ReadBufferSize: DefaultReadBufferSize,
	}
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
	// Validate ReadBufferSize
	if web.ReadBufferSize == 0 {
		web.ReadBufferSize = DefaultReadBufferSize // Not set? Use the default value.
	} else if web.ReadBufferSize < MinimumReadBufferSize {
		web.ReadBufferSize = MinimumReadBufferSize // Below the minimum? Use the minimum value.
	}
	// Try to load the TLS certificates
	if web.TLS != nil {
		if err := web.TLS.isValid(); err != nil {
			return fmt.Errorf("invalid tls config: %w", err)
		}
	}
	// Validate CORS
	if web.CORS != nil {
		if err := web.CORS.ValidateAndSetDefaults(); err != nil {
			return fmt.Errorf("invalid cors config: %w", err)
		}
	}
	return nil
}

func (web *Config) HasTLS() bool {
	return web.TLS != nil && len(web.TLS.CertificateFile) > 0 && len(web.TLS.PrivateKeyFile) > 0
}

// SocketAddress returns the combination of the Address and the Port
func (web *Config) SocketAddress() string {
	return fmt.Sprintf("%s:%d", web.Address, web.Port)
}

func (t *TLSConfig) isValid() error {
	if len(t.CertificateFile) > 0 && len(t.PrivateKeyFile) > 0 {
		_, err := tls.LoadX509KeyPair(t.CertificateFile, t.PrivateKeyFile)
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("certificate-file and private-key-file must be specified")
}

// ValidateAndSetDefaults validates the CORS configuration and sets the default values if necessary.
func (c *CORSConfig) ValidateAndSetDefaults() error {
	if len(c.AllowOrigins) == 0 {
		return errors.New("allow-origins must be specified when cors is enabled")
	}
	if c.AllowCredentials && (c.AllowOrigins == "*" || strings.Contains(c.AllowOrigins, "*")) {
		return errors.New("allow-origins cannot contain '*' when allow-credentials is true")
	}
	if len(c.AllowMethods) == 0 {
		c.AllowMethods = "GET,POST,OPTIONS"
	}
	if len(c.AllowHeaders) == 0 {
		c.AllowHeaders = "Origin,Content-Type,Accept,Authorization"
	}
	return nil
}

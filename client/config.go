package client

import (
	"crypto/tls"
	"net/http"
	"time"
)

const (
	defaultHTTPTimeout = 10 * time.Second
)

var (
	// DefaultConfig is the default client configuration
	defaultConfig = Config{
		Insecure:       false,
		IgnoreRedirect: false,
		Timeout:        defaultHTTPTimeout,
	}
)

// GetDefaultConfig returns a copy of the default configuration
func GetDefaultConfig() *Config {
	cfg := defaultConfig
	return &cfg
}

// Config is the configuration for clients
type Config struct {
	// Insecure determines whether to skip verifying the server's certificate chain and host name
	Insecure bool `yaml:"insecure"`

	// IgnoreRedirect determines whether to ignore redirects (true) or follow them (false, default)
	IgnoreRedirect bool `yaml:"ignore-redirect"`

	// Timeout for the client
	Timeout time.Duration `yaml:"timeout"`

	httpClient *http.Client
}

// ValidateAndSetDefaults validates the client configuration and sets the default values if necessary
func (c *Config) ValidateAndSetDefaults() {
	if c.Timeout < time.Millisecond {
		c.Timeout = 10 * time.Second
	}
}

// GetHTTPClient return a HTTP client matching the Config's parameters.
func (c *Config) getHTTPClient() *http.Client {
	if c.httpClient == nil {
		c.httpClient = &http.Client{
			Timeout: c.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				Proxy:               http.ProxyFromEnvironment,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: c.Insecure,
				},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if c.IgnoreRedirect {
					// Don't follow redirects
					return http.ErrUseLastResponse
				}
				// Follow redirects
				return nil
			},
		}
	}
	return c.httpClient
}

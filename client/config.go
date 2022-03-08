package client

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
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

	ErrInvalidClientOAuth2Config = errors.New(
		"todo",
	)
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

	// OAuth2 configuration for the client
	OAuth2Config *OAuth2Config `yaml:"oauth2,omitempty"`

	httpClient *http.Client
}

type OAuth2Config struct {
	TokenURL     string   `yaml:"token-url"` // e.g. https://dev-12345678.okta.com/token
	ClientID     string   `yaml:"client-id"`
	ClientSecret string   `yaml:"client-secret"`
	Scopes       []string `yaml:"scopes"` // e.g. ["openid"]
}

// ValidateAndSetDefaults validates the client configuration and sets the default values if necessary
func (c *Config) ValidateAndSetDefaults() error {
	if c.Timeout < time.Millisecond {
		c.Timeout = 10 * time.Second
	}
	if c.HasOAuth2Config() && !c.OAuth2Config.isValid() {
		return ErrInvalidClientOAuth2Config
	}

	return nil
}

func (c *Config) HasOAuth2Config() bool {
	return c.OAuth2Config != nil
}

func (c *OAuth2Config) isValid() bool {
	return len(c.TokenURL) > 0 && len(c.ClientID) > 0 && len(c.ClientSecret) > 0 && len(c.Scopes) > 0
}

// GetHTTPClient return an HTTP client matching the Config's parameters.
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
		if c.HasOAuth2Config() {
			oauth2cfg := clientcredentials.Config{
				ClientID:     c.OAuth2Config.ClientID,
				ClientSecret: c.OAuth2Config.ClientSecret,
				Scopes:       c.OAuth2Config.Scopes,
				TokenURL:     c.OAuth2Config.TokenURL,
			}
			ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c.httpClient)
			c.httpClient = oauth2cfg.Client(ctx)
		}
	}
	return c.httpClient
}

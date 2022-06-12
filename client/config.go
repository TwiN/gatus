package client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	defaultHTTPTimeout = 10 * time.Second
)

var (
	ErrInvalidDNSResolver        = errors.New("invalid DNS resolver specified. Required format is {proto}://{ip}:{port}")
	ErrInvalidClientOAuth2Config = errors.New("invalid OAuth2 configuration, all fields are required")

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
	Insecure bool `yaml:"insecure,omitempty"`

	// IgnoreRedirect determines whether to ignore redirects (true) or follow them (false, default)
	IgnoreRedirect bool `yaml:"ignore-redirect,omitempty"`

	// Timeout for the client
	Timeout time.Duration `yaml:"timeout"`

	// DNSResolver override for the HTTPClient
	// Expected format is {protocol}://{host}:{port}
	DNSResolver string `yaml:"dns-resolver,omitempty"`

	// OAuth2Config is the OAuth2 configuration used for the client.
	//
	// If non-nil, the http.Client returned by getHTTPClient will automatically retrieve a token if necessary.
	// See configureOAuth2 for more details.
	OAuth2Config *OAuth2Config `yaml:"oauth2,omitempty"`

	httpClient *http.Client
}

// DNSResolverConfig is the parsed configuration from the DNSResolver config string.
type DNSResolverConfig struct {
	Protocol string
	Host     string
	Port     string
}

// OAuth2Config is the configuration for the OAuth2 client credentials flow
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
	if c.HasCustomDNSResolver() {
		_, err := c.ParseDNSResolver()
		if err != nil {
			return ErrInvalidDNSResolver
		}
	}
	if c.HasOAuth2Config() && !c.OAuth2Config.isValid() {
		return ErrInvalidClientOAuth2Config
	}
	return nil
}

// Returns true if the DNSResolver is set in the configuration
func (c *Config) HasCustomDNSResolver() bool {
	return len(c.DNSResolver) > 0
}

// Parses the DNSResolver configuration string into the DNSResolverConfig struct
func (c *Config) ParseDNSResolver() (DNSResolverConfig, error) {
	re := regexp.MustCompile(`^(?P<proto>(.*))://(?P<host>[A-Za-z0-9\-\.]+):(?P<port>[0-9]+)?(.*)$`)
	matches := re.FindStringSubmatch(c.DNSResolver)
	if len(matches) == 0 {
		return DNSResolverConfig{}, errors.New("ParseError")
	}
	r := make(map[string]string)
	for i, k := range re.SubexpNames() {
		if i != 0 && k != "" {
			r[k] = matches[i]
		}
	}

	return DNSResolverConfig{
		Protocol: r["proto"],
		Host:     r["host"],
		Port:     r["port"],
	}, nil
}

// HasOAuth2Config returns true if the client has OAuth2 configuration parameters
func (c *Config) HasOAuth2Config() bool {
	return c.OAuth2Config != nil
}

// isValid() returns true if the OAuth2 configuration is valid
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
		if c.HasCustomDNSResolver() {
			dnsResolver, _ := c.ParseDNSResolver()
			dialer := &net.Dialer{
				Resolver: &net.Resolver{
					PreferGo: true,
					Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
						d := net.Dialer{}
						return d.DialContext(ctx, dnsResolver.Protocol, fmt.Sprintf("%s:%s", dnsResolver.Host, dnsResolver.Port))
					},
				},
			}
			dialCtx := func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, addr)
			}
			c.httpClient.Transport.(*http.Transport).DialContext = dialCtx
		}
		if c.HasOAuth2Config() {
			c.httpClient = configureOAuth2(c.httpClient, *c.OAuth2Config)
		}
	}
	return c.httpClient
}

// configureOAuth2 returns an HTTP client that will obtain and refresh tokens as necessary.
// The returned Client and its Transport should not be modified.
func configureOAuth2(httpClient *http.Client, c OAuth2Config) *http.Client {
	oauth2cfg := clientcredentials.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Scopes:       c.Scopes,
		TokenURL:     c.TokenURL,
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	return oauth2cfg.Client(ctx)
}

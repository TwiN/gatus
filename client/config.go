package client

import (
	"context"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/api/idtoken"
)

const (
	defaultTimeout = 10 * time.Second
)

var (
	ErrInvalidDNSResolver        = errors.New("invalid DNS resolver specified. Required format is {proto}://{ip}:{port}")
	ErrInvalidDNSResolverPort    = errors.New("invalid DNS resolver port")
	ErrInvalidClientOAuth2Config = errors.New("invalid oauth2 configuration: must define all fields for client credentials flow (token-url, client-id, client-secret, scopes)")
	ErrInvalidClientIAPConfig    = errors.New("invalid Identity-Aware-Proxy configuration: must define all fields for Google Identity-Aware-Proxy programmatic authentication (audience)")

	defaultConfig = Config{
		Insecure:       false,
		IgnoreRedirect: false,
		Timeout:        defaultTimeout,
		Network:        "ip",
	}
)

// GetDefaultConfig returns a copy of the default configuration
func GetDefaultConfig() *Config {
	cfg := defaultConfig
	return &cfg
}

// Config is the configuration for clients
type Config struct {
	// ProxyURL is the URL of the proxy to use for the client
	ProxyURL string `yaml:"proxy-url,omitempty"`

	// Insecure determines whether to skip verifying the server's certificate chain and host name
	Insecure bool `yaml:"insecure,omitempty"`

	// IgnoreRedirect determines whether to ignore redirects (true) or follow them (false, default)
	IgnoreRedirect bool `yaml:"ignore-redirect,omitempty"`

	// Timeout for the client
	Timeout time.Duration `yaml:"timeout"`

	// DNSResolver override for the HTTP client
	// Expected format is {protocol}://{host}:{port}, e.g. tcp://8.8.8.8:53
	DNSResolver string `yaml:"dns-resolver,omitempty"`

	// OAuth2Config is the OAuth2 configuration used for the client.
	//
	// If non-nil, the http.Client returned by getHTTPClient will automatically retrieve a token if necessary.
	// See configureOAuth2 for more details.
	OAuth2Config *OAuth2Config `yaml:"oauth2,omitempty"`

	// IAPConfig is the Google Cloud Identity-Aware-Proxy configuration used for the client. (e.g. audience)
	IAPConfig *IAPConfig `yaml:"identity-aware-proxy,omitempty"`

	httpClient *http.Client

	// Network (ip, ip4 or ip6) for the ICMP client
	Network string `yaml:"network"`
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

// IAPConfig is the configuration for the Google Cloud Identity-Aware-Proxy
type IAPConfig struct {
	Audience string `yaml:"audience"` // e.g. "toto.apps.googleusercontent.com"
}

// ValidateAndSetDefaults validates the client configuration and sets the default values if necessary
func (c *Config) ValidateAndSetDefaults() error {
	if c.Timeout < time.Millisecond {
		c.Timeout = 10 * time.Second
	}
	if c.HasCustomDNSResolver() {
		// Validate the DNS resolver now to make sure it will not return an error later.
		if _, err := c.parseDNSResolver(); err != nil {
			return err
		}
	}
	if c.HasOAuth2Config() && !c.OAuth2Config.isValid() {
		return ErrInvalidClientOAuth2Config
	}
	if c.HasIAPConfig() && !c.IAPConfig.isValid() {
		return ErrInvalidClientIAPConfig
	}
	return nil
}

// HasCustomDNSResolver returns whether a custom DNSResolver is configured
func (c *Config) HasCustomDNSResolver() bool {
	return len(c.DNSResolver) > 0
}

// parseDNSResolver parses the DNS resolver into the DNSResolverConfig struct
func (c *Config) parseDNSResolver() (*DNSResolverConfig, error) {
	re := regexp.MustCompile(`^(?P<proto>(.*))://(?P<host>[A-Za-z0-9\-\.]+):(?P<port>[0-9]+)?(.*)$`)
	matches := re.FindStringSubmatch(c.DNSResolver)
	if len(matches) == 0 {
		return nil, ErrInvalidDNSResolver
	}
	r := make(map[string]string)
	for i, k := range re.SubexpNames() {
		if i != 0 && k != "" {
			r[k] = matches[i]
		}
	}
	port, err := strconv.Atoi(r["port"])
	if err != nil {
		return nil, err
	}
	if port < 1 || port > 65535 {
		return nil, ErrInvalidDNSResolverPort
	}
	return &DNSResolverConfig{
		Protocol: r["proto"],
		Host:     r["host"],
		Port:     r["port"],
	}, nil
}

// HasOAuth2Config returns true if the client has OAuth2 configuration parameters
func (c *Config) HasOAuth2Config() bool {
	return c.OAuth2Config != nil
}

// HasIAPConfig returns true if the client has IAP configuration parameters
func (c *Config) HasIAPConfig() bool {
	return c.IAPConfig != nil
}

// isValid() returns true if the IAP configuration is valid
func (c *IAPConfig) isValid() bool {
	return len(c.Audience) > 0
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
		if c.ProxyURL != "" {
			proxyURL, err := url.Parse(c.ProxyURL)
			if err != nil {
				log.Println("[client][getHTTPClient] THIS SHOULD NOT HAPPEN. Silently ignoring custom proxy due to error:", err.Error())
			} else {
				c.httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyURL)
			}
		}
		if c.HasCustomDNSResolver() {
			dnsResolver, err := c.parseDNSResolver()
			if err != nil {
				// We're ignoring the error, because it should have been validated on startup ValidateAndSetDefaults.
				// It shouldn't happen, but if it does, we'll log it... Better safe than sorry ;)
				log.Println("[client][getHTTPClient] THIS SHOULD NOT HAPPEN. Silently ignoring invalid DNS resolver due to error:", err.Error())
			} else {
				dialer := &net.Dialer{
					Resolver: &net.Resolver{
						PreferGo: true,
						Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
							d := net.Dialer{}
							return d.DialContext(ctx, dnsResolver.Protocol, dnsResolver.Host+":"+dnsResolver.Port)
						},
					},
				}
				c.httpClient.Transport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
					return dialer.DialContext(ctx, network, addr)
				}
			}
		}
		if c.HasOAuth2Config() && c.HasIAPConfig() {
			log.Println("[client][getHTTPClient] Error: Both Identity-Aware-Proxy and Oauth2 configuration are present.")
		} else if c.HasOAuth2Config() {
			c.httpClient = configureOAuth2(c.httpClient, *c.OAuth2Config)
		} else if c.HasIAPConfig() {
			c.httpClient = configureIAP(c.httpClient, *c.IAPConfig)
		}
	}
	return c.httpClient
}

// validateIAPToken returns a boolean that will define if the google identity-aware-proxy token can be fetch
// and if is it valid.
func validateIAPToken(ctx context.Context, c IAPConfig) bool {
	ts, err := idtoken.NewTokenSource(ctx, c.Audience)
	if err != nil {
		log.Println("[client][ValidateIAPToken] Claiming Identity token failed. error:", err.Error())
		return false
	}
	tok, err := ts.Token()
	if err != nil {
		log.Println("[client][ValidateIAPToken] Get Identity-Aware-Proxy token failed. error:", err.Error())
		return false
	}
	payload, err := idtoken.Validate(ctx, tok.AccessToken, c.Audience)
	_ = payload
	if err != nil {
		log.Println("[client][ValidateIAPToken] Token Validation failed. error:", err.Error())
		return false
	}
	return true
}

// configureIAP returns an HTTP client that will obtain and refresh Identity-Aware-Proxy tokens as necessary.
// The returned Client and its Transport should not be modified.
func configureIAP(httpClient *http.Client, c IAPConfig) *http.Client {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, httpClient)
	if validateIAPToken(ctx, c) {
		ts, err := idtoken.NewTokenSource(ctx, c.Audience)
		if err != nil {
			log.Println("[client][ConfigureIAP] Claiming Token Source failed. error:", err.Error())
			return httpClient
		}
		client := oauth2.NewClient(ctx, ts)
		client.Timeout = httpClient.Timeout
		return client
	}
	return httpClient
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
	client := oauth2cfg.Client(ctx)
	client.Timeout = httpClient.Timeout
	return client
}

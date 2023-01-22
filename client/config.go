package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defaultHTTPTimeout = 10 * time.Second
	defaultGRPCTimeout = 10 * time.Second
)

var (
	ErrInvalidDNSResolver        = errors.New("invalid DNS resolver specified. Required format is {proto}://{ip}:{port}")
	ErrInvalidDNSResolverPort    = errors.New("invalid DNS resolver port")
	ErrInvalidClientOAuth2Config = errors.New("invalid oauth2 configuration: must define all fields for client credentials flow (token-url, client-id, client-secret, scopes)")
  ErrFileNotExist              = errors.New("file not exists: ")
	ErrFailedToLoadCert          = errors.New("failed to load a server certificate in the configuration")
  ErrFailedToParseCert         = errors.New("Failed to parse the server certificate")
	ErrFailedToCreateConnection  = errors.New("Failed to create a client connection")
 
  // default client configuration for non-grpc
	defaultConfig = Config{
		Insecure:       false,
		IgnoreRedirect: false,
		Timeout:        defaultHTTPTimeout,
	}
  // default client configuration for grpc. gRPC client does not handle 302 redirects.
  // a gRPC client is not meant to be a full HTTP client. For example, cookies and redirects are not part of gRPC.
	defaultGrpcConfig = Config {
		Insecure:       false,
		Timeout:        defaultGRPCTimeout,
	}
)

// GetDefaultConfig returns a copy of the default client configuration for non-grpc.
func GetDefaultConfig() *Config {
	cfg := defaultConfig
	return &cfg
}

// GetGrpcDefaultConfig returns a copy of the default client configuration for grpc.
func GetGrpcDefaultConfig() *Config {
  cfg := defaultGrpcConfig
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

	// DNSResolver override for the HTTP client
	// Expected format is {protocol}://{host}:{port}, e.g. tcp://8.8.8.8:53
	DNSResolver string `yaml:"dns-resolver,omitempty"`

	// OAuth2Config is the OAuth2 configuration used for the client.
	//
	// If non-nil, the http.Client returned by getHTTPClient will automatically retrieve a token if necessary.
	// See configureOAuth2 for more details.
	OAuth2Config *OAuth2Config `yaml:"oauth2,omitempty"`

	// Cert is a file path where a server certifcate is at when the client connection requires the certificate.
	// If CertPath is passed, the file will be loaded into the cert.
	CertPath string `yaml:"certpath,omitempty"`

	// text representation of the server certificate. If CertPath is passed, the file will be loaded into the cert.
	cert string

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
		// Validate the DNS resolver now to make sure it will not return an error later.
		if _, err := c.parseDNSResolver(); err != nil {
			return err
		}
	}
	if c.HasOAuth2Config() && !c.OAuth2Config.isValid() {
		return ErrInvalidClientOAuth2Config
	}
  if len(c.CertPath) != 0 {
    _, err := os.Stat(c.CertPath)
    if os.IsNotExist(err) {
      return fmt.Errorf("%v: %s", ErrFileNotExist, c.CertPath)
    }

    // load a server certificate from the local disk 
		serverCA, err := os.ReadFile(c.CertPath)
		if err != nil {
			return fmt.Errorf("%v: %w", ErrFailedToLoadCert, err)
		}
		c.cert = string(serverCA)
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

func (c *Config) getGRPCClientConnection(hostPort string) (*grpc.ClientConn, error) {
	// initial tls configuration
	tlsConfig := &tls.Config {
		InsecureSkipVerify: c.Insecure,
	}

	// handle the server certificate if given
	if len(c.cert) != 0 {
		// parse and append a server certificate
		serverCA := []byte(c.cert)
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(serverCA) {
			return nil, fmt.Errorf("%v: %v", ErrFailedToParseCert, c.cert)
		}

		// update the tls configuration with the server cert.
		tlsConfig.RootCAs = certPool
	}

  // create the connection
	// (TODO) currently mTLS is not supported. If a gRPC server requires a mTLS, the following
	// has to be updated to set the healthcheck client certificate. 
	// (Currently no HybridCKO gRPC servers on a test env requires mTLS) 
  creds := credentials.NewTLS(tlsConfig)
  conn, err := grpc.Dial(hostPort, grpc.WithTransportCredentials(creds))
  if err != nil {
		return nil, fmt.Errorf("%v: %w", ErrFailedToCreateConnection, err)
  }
  
	return conn, nil
}

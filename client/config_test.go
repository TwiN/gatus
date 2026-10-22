package client

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestConfig_getHTTPClient(t *testing.T) {
	insecureConfig := &Config{Insecure: true}
	insecureConfig.ValidateAndSetDefaults()
	insecureClient := insecureConfig.getHTTPClient()
	if !(insecureClient.Transport).(*http.Transport).TLSClientConfig.InsecureSkipVerify {
		t.Error("expected Config.Insecure set to true to cause the HTTP client to skip certificate verification")
	}
	if insecureClient.Timeout != defaultTimeout {
		t.Error("expected Config.Timeout to default the HTTP client to a timeout of 10s")
	}
	if insecureClient.Jar != nil {
		t.Error("expected Config.StoreCookies to default the HTTP client to not store cookies")
	}
	request, _ := http.NewRequest("GET", "", nil)
	if err := insecureClient.CheckRedirect(request, nil); err != nil {
		t.Error("expected Config.IgnoreRedirect set to false to cause the HTTP client's CheckRedirect to return nil")
	}

	secureConfig := &Config{IgnoreRedirect: true, Timeout: 5 * time.Second}
	secureConfig.ValidateAndSetDefaults()
	secureClient := secureConfig.getHTTPClient()
	if (secureClient.Transport).(*http.Transport).TLSClientConfig.InsecureSkipVerify {
		t.Error("expected Config.Insecure set to false to cause the HTTP client to not skip certificate verification")
	}
	if secureClient.Timeout != 5*time.Second {
		t.Error("expected Config.Timeout to cause the HTTP client to have a timeout of 5s")
	}
	if insecureClient.Jar != nil {
		t.Error("expected Config.StoreCookies to default the HTTP client to not store cookies")
	}
	request, _ = http.NewRequest("GET", "", nil)
	if err := secureClient.CheckRedirect(request, nil); err != http.ErrUseLastResponse {
		t.Error("expected Config.IgnoreRedirect set to true to cause the HTTP client's CheckRedirect to return http.ErrUseLastResponse")
	}
}

func TestConfig_getHTTPClient_withCookieJar(t *testing.T) {
	config := &Config{StoreCookies: true}
	config.ValidateAndSetDefaults()
	client := config.getHTTPClient()
	if client.Jar == nil {
		t.Error("expected Config.StoreCookies to cause the HTTP client to have a CookieJar")
	}
}

func TestConfig_getHTTPClient_withNetwork(t *testing.T) {
	tests := []struct {
		network        string
		expectedDialer bool
	}{
		{network: "ip", expectedDialer: false},
		{network: "ip4", expectedDialer: true},
		{network: "ip6", expectedDialer: true},
	}
	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			cfg := &Config{Network: tt.network}
			if err := cfg.ValidateAndSetDefaults(); err != nil {
				t.Fatal(err)
			}
			client := cfg.getHTTPClient()
			transport := client.Transport.(*http.Transport)
			if (transport.DialContext != nil) != tt.expectedDialer {
				t.Errorf("expected DialContext != nil to be %v for network %s, got %v", tt.expectedDialer, tt.network, transport.DialContext != nil)
			}
		})
	}
}

func TestConfig_getHTTPClient_withNetwork_connection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// An IPv4-configured client should successfully connect to the local IPv4 test server
	ipv4Cfg := &Config{Network: "ip4", Timeout: 2 * time.Second}
	if err := ipv4Cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatal(err)
	}
	ipv4Client := ipv4Cfg.getHTTPClient()
	resp, err := ipv4Client.Get(server.URL)
	if err != nil {
		t.Fatalf("expected IPv4 client to connect to %s, got error: %v", server.URL, err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	// An IPv6-only client attempting to dial an IPv4-only address should fail
	ipv6Cfg := &Config{Network: "ip6", Timeout: 500 * time.Millisecond}
	if err := ipv6Cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatal(err)
	}
	ipv6Client := ipv6Cfg.getHTTPClient()
	_, err = ipv6Client.Get(server.URL)
	if err == nil {
		t.Errorf("expected IPv6-only client to fail connecting to IPv4 server %s, but request succeeded", server.URL)
	}
}

func TestConfig_getHTTPClient_withNetworkAndCustomDNSResolver(t *testing.T) {
	cfg := &Config{
		Network:     "ip4",
		DNSResolver: "udp://1.1.1.1:53",
		Timeout:     5 * time.Second,
	}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatal(err)
	}
	client := cfg.getHTTPClient()
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport == nil {
		t.Fatal("expected Transport to be *http.Transport")
	}
	if transport.DialContext == nil {
		t.Error("expected DialContext to be set when combining Network and DNSResolver")
	}
}

func TestConfig_ValidateAndSetDefaults_network(t *testing.T) {
	tests := []struct {
		network string
		wantErr bool
	}{
		{network: "", wantErr: false},
		{network: "ip", wantErr: false},
		{network: "ip4", wantErr: false},
		{network: "ip6", wantErr: false},
		{network: "tcp4", wantErr: true},
		{network: "tcp6", wantErr: true},
		{network: "invalid", wantErr: true},
		{network: "ip44", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.network, func(t *testing.T) {
			cfg := &Config{Network: tt.network}
			err := cfg.ValidateAndSetDefaults()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndSetDefaults() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(tt.network) == 0 && cfg.Network != "ip" {
				t.Errorf("expected empty network to default to 'ip', got %s", cfg.Network)
			}
		})
	}
}

func TestConfig_ValidateAndSetDefaults_withCustomDNSResolver(t *testing.T) {
	type args struct {
		dnsResolver string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "with-valid-resolver",
			args: args{
				dnsResolver: "tcp://1.1.1.1:53",
			},
			wantErr: false,
		},
		{
			name: "with-invalid-resolver-port",
			args: args{
				dnsResolver: "tcp://127.0.0.1:99999",
			},
			wantErr: true,
		},
		{
			name: "with-invalid-resolver-format",
			args: args{
				dnsResolver: "foobar",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				DNSResolver: tt.args.dnsResolver,
			}
			err := cfg.ValidateAndSetDefaults()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndSetDefaults() error=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_getHTTPClient_withCustomProxyURL(t *testing.T) {
	proxyURL := "http://proxy.example.com:8080"
	cfg := &Config{
		ProxyURL: proxyURL,
	}
	cfg.ValidateAndSetDefaults()
	client := cfg.getHTTPClient()
	transport := client.Transport.(*http.Transport)
	if transport.Proxy == nil {
		t.Errorf("expected Config.ProxyURL to set the HTTP client's proxy to %s", proxyURL)
	}
	req := &http.Request{
		URL: &url.URL{
			Scheme: "http",
			Host:   "www.example.com",
		},
	}
	expectProxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Errorf("can't proxy the request %s", proxyURL)
	}
	if proxyURL != expectProxyURL.String() {
		t.Errorf("expected Config.ProxyURL to set the HTTP client's proxy to %s", proxyURL)
	}
}

func TestConfig_TlsIsValid(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *Config
		expectedErr bool
	}{
		{
			name:        "good-tls-config",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "../testdata/cert.key"}},
			expectedErr: false,
		},
		{
			name:        "missing-certificate-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "doesnotexist", PrivateKeyFile: "../testdata/cert.key"}},
			expectedErr: true,
		},
		{
			name:        "bad-certificate-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../testdata/badcert.pem", PrivateKeyFile: "../testdata/cert.key"}},
			expectedErr: true,
		},
		{
			name:        "no-certificate-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "", PrivateKeyFile: "../testdata/cert.key"}},
			expectedErr: true,
		},
		{
			name:        "missing-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "doesnotexist"}},
			expectedErr: true,
		},
		{
			name:        "no-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../testdata/cert.pem", PrivateKeyFile: ""}},
			expectedErr: true,
		},
		{
			name:        "bad-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "../testdata/badcert.key"}},
			expectedErr: true,
		},
		{
			name:        "bad-certificate-and-private-key-file",
			cfg:         &Config{TLS: &TLSConfig{CertificateFile: "../testdata/badcert.pem", PrivateKeyFile: "../testdata/badcert.key"}},
			expectedErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cfg.TLS.isValid()
			if (err != nil) != test.expectedErr {
				t.Errorf("expected the existence of an error to be %v, got %v", test.expectedErr, err)
				return
			}
			if !test.expectedErr {
				if test.cfg.TLS.isValid() != nil {
					t.Error("cfg.TLS.isValid() returned an error even though no error was expected")
				}
			}
		})
	}
}

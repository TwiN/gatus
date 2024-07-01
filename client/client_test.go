package client

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint/dns"
	"github.com/TwiN/gatus/v5/pattern"
	"github.com/TwiN/gatus/v5/test"
)

func TestGetHTTPClient(t *testing.T) {
	cfg := &Config{
		Insecure:       false,
		IgnoreRedirect: false,
		Timeout:        0,
		DNSResolver:    "tcp://1.1.1.1:53",
		OAuth2Config: &OAuth2Config{
			ClientID:     "00000000-0000-0000-0000-000000000000",
			ClientSecret: "secretsauce",
			TokenURL:     "https://token-server.local/token",
			Scopes:       []string{"https://application.local/.default"},
		},
	}
	err := cfg.ValidateAndSetDefaults()
	if err != nil {
		t.Errorf("expected error to be nil, but got: `%s`", err)
	}
	if GetHTTPClient(cfg) == nil {
		t.Error("expected client to not be nil")
	}
	if GetHTTPClient(nil) == nil {
		t.Error("expected client to not be nil")
	}
}

func TestGetDomainExpiration(t *testing.T) {
	t.Parallel()
	if domainExpiration, err := GetDomainExpiration("example.com"); err != nil {
		t.Fatalf("expected error to be nil, but got: `%s`", err)
	} else if domainExpiration <= 0 {
		t.Error("expected domain expiration to be higher than 0")
	}
	if domainExpiration, err := GetDomainExpiration("example.com"); err != nil {
		t.Errorf("expected error to be nil, but got: `%s`", err)
	} else if domainExpiration <= 0 {
		t.Error("expected domain expiration to be higher than 0")
	}
	// Hack to pretend like the domain is expiring in 1 hour, which should trigger a refresh
	whoisExpirationDateCache.SetWithTTL("example.com", time.Now().Add(time.Hour), 25*time.Hour)
	if domainExpiration, err := GetDomainExpiration("example.com"); err != nil {
		t.Errorf("expected error to be nil, but got: `%s`", err)
	} else if domainExpiration <= 0 {
		t.Error("expected domain expiration to be higher than 0")
	}
	// Make sure the refresh works when the ttl is <24 hours
	whoisExpirationDateCache.SetWithTTL("example.com", time.Now().Add(35*time.Hour), 23*time.Hour)
	if domainExpiration, err := GetDomainExpiration("example.com"); err != nil {
		t.Errorf("expected error to be nil, but got: `%s`", err)
	} else if domainExpiration <= 0 {
		t.Error("expected domain expiration to be higher than 0")
	}
}

func TestPing(t *testing.T) {
	t.Parallel()
	if success, rtt := Ping("127.0.0.1", &Config{Timeout: 500 * time.Millisecond}); !success {
		t.Error("expected true")
		if rtt == 0 {
			t.Error("Round-trip time returned on success should've higher than 0")
		}
	}
	if success, rtt := Ping("256.256.256.256", &Config{Timeout: 500 * time.Millisecond}); success {
		t.Error("expected false, because the IP is invalid")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	if success, rtt := Ping("192.168.152.153", &Config{Timeout: 500 * time.Millisecond}); success {
		t.Error("expected false, because the IP is valid but the host should be unreachable")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	// Can't perform integration tests (e.g. pinging public targets by single-stacked hostname) here,
	// because ICMP is blocked in the network of GitHub-hosted runners.
	if success, rtt := Ping("127.0.0.1", &Config{Timeout: 500 * time.Millisecond, Network: "ip"}); !success {
		t.Error("expected true")
		if rtt == 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	if success, rtt := Ping("::1", &Config{Timeout: 500 * time.Millisecond, Network: "ip"}); !success {
		t.Error("expected true")
		if rtt == 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	if success, rtt := Ping("::1", &Config{Timeout: 500 * time.Millisecond, Network: "ip4"}); success {
		t.Error("expected false, because the IP isn't an IPv4 address")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
	if success, rtt := Ping("127.0.0.1", &Config{Timeout: 500 * time.Millisecond, Network: "ip6"}); success {
		t.Error("expected false, because the IP isn't an IPv6 address")
		if rtt != 0 {
			t.Error("Round-trip time returned on failure should've been 0")
		}
	}
}

func TestCanPerformStartTLS(t *testing.T) {
	type args struct {
		address  string
		insecure bool
	}
	tests := []struct {
		name          string
		args          args
		wantConnected bool
		wantErr       bool
	}{
		{
			name: "invalid address",
			args: args{
				address: "test",
			},
			wantConnected: false,
			wantErr:       true,
		},
		{
			name: "error dial",
			args: args{
				address: "test:1234",
			},
			wantConnected: false,
			wantErr:       true,
		},
		{
			name: "valid starttls",
			args: args{
				address: "smtp.gmail.com:587",
			},
			wantConnected: true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			connected, _, err := CanPerformStartTLS(tt.args.address, &Config{Insecure: tt.args.insecure, Timeout: 5 * time.Second})
			if (err != nil) != tt.wantErr {
				t.Errorf("CanPerformStartTLS() err=%v, wantErr=%v", err, tt.wantErr)
				return
			}
			if connected != tt.wantConnected {
				t.Errorf("CanPerformStartTLS() connected=%v, wantConnected=%v", connected, tt.wantConnected)
			}
		})
	}
}

func TestCanPerformTLS(t *testing.T) {
	type args struct {
		address  string
		insecure bool
	}
	tests := []struct {
		name          string
		args          args
		wantConnected bool
		wantErr       bool
	}{
		{
			name: "invalid address",
			args: args{
				address: "test",
			},
			wantConnected: false,
			wantErr:       true,
		},
		{
			name: "error dial",
			args: args{
				address: "test:1234",
			},
			wantConnected: false,
			wantErr:       true,
		},
		{
			name: "valid tls",
			args: args{
				address: "smtp.gmail.com:465",
			},
			wantConnected: true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			connected, _, err := CanPerformTLS(tt.args.address, &Config{Insecure: tt.args.insecure, Timeout: 5 * time.Second})
			if (err != nil) != tt.wantErr {
				t.Errorf("CanPerformTLS() err=%v, wantErr=%v", err, tt.wantErr)
				return
			}
			if connected != tt.wantConnected {
				t.Errorf("CanPerformTLS() connected=%v, wantConnected=%v", connected, tt.wantConnected)
			}
		})
	}
}

func TestCanCreateTCPConnection(t *testing.T) {
	if CanCreateTCPConnection("127.0.0.1", &Config{Timeout: 5 * time.Second}) {
		t.Error("should've failed, because there's no port in the address")
	}
	if !CanCreateTCPConnection("1.1.1.1:53", &Config{Timeout: 5 * time.Second}) {
		t.Error("should've succeeded, because that IP should always™ be up")
	}
}

// This test checks if a HTTP client configured with `configureOAuth2()` automatically
// performs a Client Credentials OAuth2 flow and adds the obtained token as a `Authorization`
// header to all outgoing HTTP calls.
func TestHttpClientProvidesOAuth2BearerToken(t *testing.T) {
	defer InjectHTTPClient(nil)
	oAuth2Config := &OAuth2Config{
		ClientID:     "00000000-0000-0000-0000-000000000000",
		ClientSecret: "secretsauce",
		TokenURL:     "https://token-server.local/token",
		Scopes:       []string{"https://application.local/.default"},
	}
	mockHttpClient := &http.Client{
		Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
			// if the mock HTTP client tries to get a token from the `token-server`
			// we provide the expected token response
			if r.Host == "token-server.local" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(bytes.NewReader(
						[]byte(
							`{"token_type":"Bearer","expires_in":3599,"ext_expires_in":3599,"access_token":"secret-token"}`,
						),
					)),
				}
			}
			// to verify the headers were sent as expected, we echo them back in the
			// `X-Org-Authorization` header and check if the token value matches our
			// mocked `token-server` response
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: map[string][]string{
					"X-Org-Authorization": {r.Header.Get("Authorization")},
				},
				Body: http.NoBody,
			}
		}),
	}
	mockHttpClientWithOAuth := configureOAuth2(mockHttpClient, *oAuth2Config)
	InjectHTTPClient(mockHttpClientWithOAuth)
	request, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:8282", http.NoBody)
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	response, err := mockHttpClientWithOAuth.Do(request)
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if response.Header == nil {
		t.Error("expected response headers, but got nil")
	}
	// the mock response echos the Authorization header used in the request back
	// to us as `X-Org-Authorization` header, we check here if the value matches
	// our expected token `secret-token`
	if response.Header.Get("X-Org-Authorization") != "Bearer secret-token" {
		t.Error("expected `secret-token` as Bearer token in the mocked response header `X-Org-Authorization`, but got", response.Header.Get("X-Org-Authorization"))
	}
}

func TestQueryWebSocket(t *testing.T) {
	_, _, err := QueryWebSocket("", "body", &Config{Timeout: 2 * time.Second})
	if err == nil {
		t.Error("expected an error due to the address being invalid")
	}
	_, _, err = QueryWebSocket("ws://example.org", "body", &Config{Timeout: 2 * time.Second})
	if err == nil {
		t.Error("expected an error due to the target not being websocket-friendly")
	}
}

func TestTlsRenegotiation(t *testing.T) {
	tests := []struct {
		name           string
		cfg            TLSConfig
		expectedConfig tls.RenegotiationSupport
	}{
		{
			name:           "default",
			cfg:            TLSConfig{CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "../testdata/cert.key"},
			expectedConfig: tls.RenegotiateNever,
		},
		{
			name:           "never",
			cfg:            TLSConfig{RenegotiationSupport: "never", CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "../testdata/cert.key"},
			expectedConfig: tls.RenegotiateNever,
		},
		{
			name:           "once",
			cfg:            TLSConfig{RenegotiationSupport: "once", CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "../testdata/cert.key"},
			expectedConfig: tls.RenegotiateOnceAsClient,
		},
		{
			name:           "freely",
			cfg:            TLSConfig{RenegotiationSupport: "freely", CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "../testdata/cert.key"},
			expectedConfig: tls.RenegotiateFreelyAsClient,
		},
		{
			name:           "not-valid-and-broken",
			cfg:            TLSConfig{RenegotiationSupport: "invalid", CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "../testdata/cert.key"},
			expectedConfig: tls.RenegotiateNever,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tls := &tls.Config{}
			tlsConfig := configureTLS(tls, test.cfg)
			if tlsConfig.Renegotiation != test.expectedConfig {
				t.Errorf("expected tls renegotiation to be %v, but got %v", test.expectedConfig, tls.Renegotiation)
			}
		})
	}
}

func TestQueryDNS(t *testing.T) {
	tests := []struct {
		name            string
		inputDNS        dns.Config
		inputURL        string
		expectedDNSCode string
		expectedBody    string
		isErrExpected   bool
	}{
		{
			name: "test Config with type A",
			inputDNS: dns.Config{
				QueryType: "A",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "93.184.215.14",
		},
		{
			name: "test Config with type AAAA",
			inputDNS: dns.Config{
				QueryType: "AAAA",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "2606:2800:21f:cb07:6820:80da:af6b:8b2c",
		},
		{
			name: "test Config with type CNAME",
			inputDNS: dns.Config{
				QueryType: "CNAME",
				QueryName: "en.wikipedia.org.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "dyna.wikimedia.org.",
		},
		{
			name: "test Config with type MX",
			inputDNS: dns.Config{
				QueryType: "MX",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    ".",
		},
		{
			name: "test Config with type NS",
			inputDNS: dns.Config{
				QueryType: "NS",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "*.iana-servers.net.",
		},
		{
			name: "test Config with fake type and retrieve error",
			inputDNS: dns.Config{
				QueryType: "B",
				QueryName: "example",
			},
			inputURL:      "8.8.8.8",
			isErrExpected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, dnsRCode, body, err := QueryDNS(test.inputDNS.QueryType, test.inputDNS.QueryName, test.inputURL)
			if test.isErrExpected && err == nil {
				t.Errorf("there should be an error")
			}
			if dnsRCode != test.expectedDNSCode {
				t.Errorf("expected DNSRCode to be %s, got %s", test.expectedDNSCode, dnsRCode)
			}
			if test.inputDNS.QueryType == "NS" {
				// Because there are often multiple nameservers backing a single domain, we'll only look at the suffix
				if !pattern.Match(test.expectedBody, string(body)) {
					t.Errorf("got %s, expected result %s,", string(body), test.expectedBody)
				}
			} else {
				if string(body) != test.expectedBody {
					t.Errorf("got %s, expected result %s,", string(body), test.expectedBody)
				}
			}
		})
		time.Sleep(5 * time.Millisecond)
	}
}

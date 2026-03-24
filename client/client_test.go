package client

import (
	"bytes"
	"crypto/tls"
	"io"
	"net/http"
	"net/netip"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint/dns"
	"github.com/TwiN/gatus/v5/pattern"
	"github.com/TwiN/gatus/v5/test"
)

func TestGetHTTPClient(t *testing.T) {
	t.Parallel()
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

func TestRdapQuery(t *testing.T) {
	t.Parallel()
	if _, err := rdapQuery("1.1.1.1"); err == nil {
		t.Error("expected an error due to the invalid domain type")
	}
	if _, err := rdapQuery("eurid.eu"); err == nil {
		t.Error("expected an error as there is no RDAP support currently in .eu")
	}
	if response, err := rdapQuery("example.com"); err != nil {
		t.Fatal("expected no error, got", err.Error())
	} else if response.ExpirationDate.Unix() <= 0 {
		t.Error("expected to have a valid expiry date, got", response.ExpirationDate.Unix())
	}
}

func TestGetDomainExpiration(t *testing.T) {
	t.Parallel()
	if domainExpiration, err := GetDomainExpiration("gatus.io"); err != nil {
		t.Fatalf("expected error to be nil, but got: `%s`", err)
	} else if domainExpiration <= 0 {
		t.Error("expected domain expiration to be higher than 0")
	}
	if domainExpiration, err := GetDomainExpiration("gatus.io"); err != nil {
		t.Errorf("expected error to be nil, but got: `%s`", err)
	} else if domainExpiration <= 0 {
		t.Error("expected domain expiration to be higher than 0")
	}
	// Hack to pretend like the domain is expiring in 1 hour, which should trigger a refresh
	whoisExpirationDateCache.SetWithTTL("gatus.io", time.Now().Add(time.Hour), 25*time.Hour)
	if domainExpiration, err := GetDomainExpiration("gatus.io"); err != nil {
		t.Errorf("expected error to be nil, but got: `%s`", err)
	} else if domainExpiration <= 0 {
		t.Error("expected domain expiration to be higher than 0")
	}
	// Make sure the refresh works when the ttl is <24 hours
	whoisExpirationDateCache.SetWithTTL("gatus.io", time.Now().Add(35*time.Hour), 23*time.Hour)
	if domainExpiration, err := GetDomainExpiration("gatus.io"); err != nil {
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

func TestShouldRunPingerAsPrivileged(t *testing.T) {
	// Don't run in parallel since we're testing system-dependent behavior
	if runtime.GOOS == "windows" {
		result := ShouldRunPingerAsPrivileged()
		if !result {
			t.Error("On Windows, ShouldRunPingerAsPrivileged() should return true")
		}
		return
	}

	// Non-Windows tests
	result := ShouldRunPingerAsPrivileged()
	isRoot := os.Geteuid() == 0

	// Test cases based on current environment
	if isRoot {
		if !result {
			t.Error("When running as root, ShouldRunPingerAsPrivileged() should return true")
		}
	} else {
		// When not root, the result depends on raw socket creation
		// We can at least verify the function runs without panic
		t.Logf("Non-root privileged result: %v", result)
	}
}

func TestCanPerformStartTLS(t *testing.T) {
	type args struct {
		address     string
		insecure    bool
		dnsresolver string
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
		{
			name: "dns resolver",
			args: args{
				address:     "smtp.gmail.com:587",
				dnsresolver: "tcp://1.1.1.1:53",
			},
			wantConnected: true,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			connected, _, err := CanPerformStartTLS(tt.args.address, &Config{Insecure: tt.args.insecure, Timeout: 5 * time.Second, DNSResolver: tt.args.dnsresolver})
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
		{
			name: "bad cert with insecure true",
			args: args{
				address:  "expired.badssl.com:443",
				insecure: true,
			},
			wantConnected: true,
			wantErr:       false,
		},
		{
			name: "bad cert with insecure false",
			args: args{
				address:  "expired.badssl.com:443",
				insecure: false,
			},
			wantConnected: false,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			connected, _, _, err := CanPerformTLS(tt.args.address, "", &Config{Insecure: tt.args.insecure, Timeout: 5 * time.Second})
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

func TestCanCreateConnection(t *testing.T) {
	t.Parallel()
	connected, _ := CanCreateNetworkConnection("tcp", "127.0.0.1", "", &Config{Timeout: 5 * time.Second})
	if connected {
		t.Error("should've failed, because there's no port in the address")
	}
	connected, _ = CanCreateNetworkConnection("tcp", "1.1.1.1:53", "", &Config{Timeout: 5 * time.Second})
	if !connected {
		t.Error("should've succeeded, because that IP should always™ be up")
	}
}

// This test checks if a HTTP client configured with `configureOAuth2()` automatically
// performs a Client Credentials OAuth2 flow and adds the obtained token as a `Authorization`
// header to all outgoing HTTP calls.
func TestHttpClientProvidesOAuth2BearerToken(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	_, _, err := QueryWebSocket("", "body", nil, &Config{Timeout: 2 * time.Second})
	if err == nil {
		t.Error("expected an error due to the address being invalid")
	}
	_, _, err = QueryWebSocket("ws://example.org", "body", nil, &Config{Timeout: 2 * time.Second})
	if err == nil {
		t.Error("expected an error due to the target not being websocket-friendly")
	}
}

func TestTlsRenegotiation(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
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
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			tls := &tls.Config{}
			tlsConfig := configureTLS(tls, scenario.cfg)
			if tlsConfig.Renegotiation != scenario.expectedConfig {
				t.Errorf("expected tls renegotiation to be %v, but got %v", scenario.expectedConfig, tls.Renegotiation)
			}
		})
	}
}

func TestQueryDNS(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
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
			expectedBody:    "__IPV4__",
		},
		{
			name: "test Config with type AAAA",
			inputDNS: dns.Config{
				QueryType: "AAAA",
				QueryName: "example.com.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "__IPV6__",
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
			expectedBody:    "*.ns.cloudflare.com.",
		},
		{
			name: "test Config with type PTR",
			inputDNS: dns.Config{
				QueryType: "PTR",
				QueryName: "8.8.8.8.in-addr.arpa.",
			},
			inputURL:        "8.8.8.8",
			expectedDNSCode: "NOERROR",
			expectedBody:    "dns.google.",
		},
		{
			name: "test Config with type PTR and forward IP / no in-addr",
			inputDNS: dns.Config{
				QueryType: "PTR",
				QueryName: "1.0.0.1",
			},
			inputURL:        "1.1.1.1",
			expectedDNSCode: "NOERROR",
			expectedBody:    "one.one.one.one.",
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
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			_, dnsRCode, body, err := QueryDNS(scenario.inputDNS.QueryType, scenario.inputDNS.QueryName, scenario.inputURL)
			if scenario.isErrExpected && err == nil {
				t.Errorf("there should be an error")
			}
			if dnsRCode != scenario.expectedDNSCode {
				t.Errorf("expected DNSRCode to be %s, got %s", scenario.expectedDNSCode, dnsRCode)
			}
			if scenario.inputDNS.QueryType == "NS" {
				// Because there are often multiple nameservers backing a single domain, we'll only look at the suffix
				if !pattern.Match(scenario.expectedBody, string(body)) {
					t.Errorf("got %s, expected result %s,", string(body), scenario.expectedBody)
				}
			} else {
				if string(body) != scenario.expectedBody {
					// little hack to validate arbitrary ipv4/ipv6
					switch scenario.expectedBody {
					case "__IPV4__":
						if addr, err := netip.ParseAddr(string(body)); err != nil {
							t.Errorf("got %s, expected result %s", string(body), scenario.expectedBody)
						} else if !addr.Is4() {
							t.Errorf("got %s, expected valid IPv4", string(body))
						}
					case "__IPV6__":
						if addr, err := netip.ParseAddr(string(body)); err != nil {
							t.Errorf("got %s, expected result %s", string(body), scenario.expectedBody)
						} else if !addr.Is6() {
							t.Errorf("got %s, expected valid IPv6", string(body))
						}
					default:
						t.Errorf("got %s, expected result %s", string(body), scenario.expectedBody)
					}
				}
			}
		})
		time.Sleep(10 * time.Millisecond)
	}
}

func TestCheckSSHBanner(t *testing.T) {
	t.Parallel()
	cfg := &Config{Timeout: 3}
	t.Run("no-auth-ssh", func(t *testing.T) {
		connected, status, err := CheckSSHBanner("tty.sdf.org", cfg)
		if err != nil {
			t.Errorf("Expected: error != nil, got: %v ", err)
		}
		if connected == false {
			t.Errorf("Expected: connected == true, got: %v", connected)
		}
		if status != 0 {
			t.Errorf("Expected: 0, got: %v", status)
		}
	})
	t.Run("invalid-address", func(t *testing.T) {
		connected, status, err := CheckSSHBanner("idontplaytheodds.com", cfg)
		if err == nil {
			t.Errorf("Expected: error, got: %v ", err)
		}
		if connected != false {
			t.Errorf("Expected: connected == false, got: %v", connected)
		}
		if status != 1 {
			t.Errorf("Expected: 1, got: %v", status)
		}
	})
}

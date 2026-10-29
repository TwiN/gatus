package client

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/miekg/dns"
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
			name: "with-valid-ipv6-resolver",
			args: args{
				dnsResolver: "udp://[2001:4860:4860::8888]:53",
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

func TestConfig_getHTTPClient_withCustomDNSResolverOverIPv6(t *testing.T) {
	// DNS server on [::1] that resolves every name to ::1
	packetConn, err := net.ListenPacket("udp", "[::1]:0")
	if err != nil {
		t.Skip("no IPv6 loopback:", err)
	}
	dnsServer := &dns.Server{PacketConn: packetConn, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		if r.Question[0].Qtype == dns.TypeAAAA {
			m.Answer = append(m.Answer, &dns.AAAA{
				Hdr:  dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: 60},
				AAAA: net.IPv6loopback,
			})
		}
		w.WriteMsg(m)
	})}
	go dnsServer.ActivateAndServe()
	defer dnsServer.Shutdown()
	// HTTP server on [::1]
	listener, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skip("no IPv6 loopback:", err)
	}
	httpServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	httpServer.Listener.Close()
	httpServer.Listener = listener
	httpServer.Start()
	defer httpServer.Close()

	cfg := &Config{DNSResolver: "udp://" + packetConn.LocalAddr().String()} // e.g. udp://[::1]:12345
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("ValidateAndSetDefaults() error=%v", err)
	}
	response, err := cfg.getHTTPClient().Get(fmt.Sprintf("http://gatus.test:%d/", listener.Addr().(*net.TCPAddr).Port))
	if err != nil {
		t.Fatalf("expected the request to resolve gatus.test through %s, got error=%v", cfg.DNSResolver, err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, response.StatusCode)
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

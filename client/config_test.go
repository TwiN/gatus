package client

import (
	"errors"
	"net/http"
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
	request, _ = http.NewRequest("GET", "", nil)
	if err := secureClient.CheckRedirect(request, nil); err != http.ErrUseLastResponse {
		t.Error("expected Config.IgnoreRedirect set to true to cause the HTTP client's CheckRedirect to return http.ErrUseLastResponse")
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

func TestConfig_ValidateAndSetDefaults_Kerberos(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name: "valid kerberos config",
			config: &Config{
				KerberosConfig: &KerberosConfig{
					Krb5ConfigFile: "../testdata/krb5.conf",
					KeytabFile:     "../testdata/gatus.keytab",
					Principal:      "gatus@EXAMPLE.COM",
					SPN:            "HTTP/test.example.com",
				},
			},
			wantErr: nil,
		},
		{
			name: "valid kerberos config without krb5 config file",
			config: &Config{
				KerberosConfig: &KerberosConfig{
					KeytabFile: "../testdata/gatus.keytab",
					Principal:  "gatus@EXAMPLE.COM",
				},
			},
			wantErr: nil,
		},
		{
			name: "invalid kerberos config without principal",
			config: &Config{
				KerberosConfig: &KerberosConfig{
					KeytabFile: "../testdata/gatus.keytab",
				},
			},
			wantErr: ErrInvalidClientKerberosConfig,
		},
		{
			name: "invalid kerberos config without keytab file",
			config: &Config{
				KerberosConfig: &KerberosConfig{
					Principal: "gatus@EXAMPLE.COM",
				},
			},
			wantErr: ErrInvalidClientKerberosConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateAndSetDefaults()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestLoadKerberosConfig(t *testing.T) {
	cfg, err := loadKerberosConfig(KerberosConfig{
		Krb5ConfigFile: "../testdata/krb5.conf",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected kerberos config to be loaded")
	}
}

func TestNewKerberosClientWithKeytab(t *testing.T) {
	cfg, err := loadKerberosConfig(KerberosConfig{
		Krb5ConfigFile: "../testdata/krb5.conf",
	})
	if err != nil {
		t.Fatal(err)
	}

	client, err := newKerberosClient(KerberosConfig{
		Krb5ConfigFile: "../testdata/krb5.conf",
		KeytabFile:     "../testdata/gatus.keytab",
		Principal:      "gatus@EXAMPLE.COM",
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if client == nil {
		t.Fatal("expected kerberos client to be created")
	}
}

func TestSplitPrincipal(t *testing.T) {
	tests := []struct {
		name         string
		principal    string
		wantUsername string
		wantRealm    string
		wantErr      bool
	}{
		{
			name:         "valid principal",
			principal:    "gatus@EXAMPLE.COM",
			wantUsername: "gatus",
			wantRealm:    "EXAMPLE.COM",
			wantErr:      false,
		},
		{
			name:      "missing realm",
			principal: "gatus",
			wantErr:   true,
		},
		{
			name:      "missing username",
			principal: "@EXAMPLE.COM",
			wantErr:   true,
		},
		{
			name:      "missing realm value",
			principal: "gatus@",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, realm, err := splitPrincipal(tt.principal)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if username != tt.wantUsername {
				t.Fatalf("expected username %q, got %q", tt.wantUsername, username)
			}

			if realm != tt.wantRealm {
				t.Fatalf("expected realm %q, got %q", tt.wantRealm, realm)
			}
		})
	}
}

func TestDefaultHTTPSpn(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://test.example.com/path", nil)
	if err != nil {
		t.Fatal(err)
	}

	got := defaultHTTPSpn(req)
	want := "HTTP/test.example.com"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestDefaultHTTPSpnWithPort(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "https://test.example.com:8443/path", nil)
	if err != nil {
		t.Fatal(err)
	}

	got := defaultHTTPSpn(req)
	want := "HTTP/test.example.com"

	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestConfigureKerberosWrapsTransport(t *testing.T) {
	httpClient := &http.Client{
		Transport: http.DefaultTransport,
	}

	configuredClient := configureKerberos(httpClient, KerberosConfig{
		KeytabFile: "../testdata/gatus.keytab",
		Principal:  "gatus@EXAMPLE.COM",
	})

	if configuredClient.Transport == nil {
		t.Fatal("expected transport to be configured")
	}

	if _, ok := configuredClient.Transport.(*kerberosTransport); !ok {
		t.Fatalf("expected kerberosTransport, got %T", configuredClient.Transport)
	}
}

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TwiN/gatus/v5/config/web"
)

func TestHealthCheckURL(t *testing.T) {
	scenarios := []struct {
		name        string
		cfg         *web.Config
		expectedURL string
	}{
		{
			name:        "default",
			cfg:         web.GetDefaultConfig(),
			expectedURL: "http://127.0.0.1:8080/health",
		},
		{
			name:        "empty-address",
			cfg:         &web.Config{Port: 8080},
			expectedURL: "http://127.0.0.1:8080/health",
		},
		{
			name:        "ipv6-unspecified-address",
			cfg:         &web.Config{Address: "::", Port: 8080},
			expectedURL: "http://127.0.0.1:8080/health",
		},
		{
			name:        "custom-address-and-port",
			cfg:         &web.Config{Address: "10.0.0.5", Port: 9090},
			expectedURL: "http://10.0.0.5:9090/health",
		},
		{
			name:        "ipv6-address",
			cfg:         &web.Config{Address: "::1", Port: 8080},
			expectedURL: "http://[::1]:8080/health",
		},
		{
			name:        "tls",
			cfg:         &web.Config{Address: "0.0.0.0", Port: 8443, TLS: &web.TLSConfig{CertificateFile: "cert.pem", PrivateKeyFile: "key.pem"}},
			expectedURL: "https://127.0.0.1:8443/health",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if url := healthCheckURL(scenario.cfg); url != scenario.expectedURL {
				t.Errorf("expected %s, got %s", scenario.expectedURL, url)
			}
		})
	}
}

func TestCheckHealth(t *testing.T) {
	scenarios := []struct {
		name        string
		statusCode  int
		tls         bool
		expectedErr bool
	}{
		{name: "healthy", statusCode: http.StatusOK},
		{name: "healthy-tls", statusCode: http.StatusOK, tls: true},
		{name: "unhealthy", statusCode: http.StatusInternalServerError, expectedErr: true},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/health" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(scenario.statusCode)
			})
			var server *httptest.Server
			if scenario.tls {
				server = httptest.NewTLSServer(handler)
			} else {
				server = httptest.NewServer(handler)
			}
			defer server.Close()
			err := checkHealth(server.URL + "/health")
			if scenario.expectedErr && err == nil {
				t.Error("expected an error, got none")
			} else if !scenario.expectedErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestCheckHealthWhenServerIsDown(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	url := server.URL + "/health"
	server.Close()
	if err := checkHealth(url); err == nil {
		t.Error("expected an error when the server is down, got none")
	}
}

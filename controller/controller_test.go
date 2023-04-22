package controller

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/web"
	"github.com/TwiN/gatus/v5/core"
)

func TestHandle(t *testing.T) {
	cfg := &config.Config{
		Web: &web.Config{
			Address: "0.0.0.0",
			Port:    rand.Intn(65534),
		},
		Endpoints: []*core.Endpoint{
			{
				Name:  "frontend",
				Group: "core",
			},
			{
				Name:  "backend",
				Group: "core",
			},
		},
	}
	_ = os.Setenv("ROUTER_TEST", "true")
	_ = os.Setenv("ENVIRONMENT", "dev")
	defer os.Clearenv()
	Handle(cfg)
	defer Shutdown()
	request, _ := http.NewRequest("GET", "/health", http.NoBody)
	responseRecorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusOK {
		t.Error("expected GET /health to return status code 200")
	}
	if server == nil {
		t.Fatal("server should've been set (but because we set ROUTER_TEST, it shouldn't have been started)")
	}
}

func TestHandleTLS(t *testing.T) {
	scenarios := []struct {
		name               string
		tls                *web.TLSConfig
		expectedStatusCode int
	}{
		{
			name:               "good-tls-config",
			tls:                &web.TLSConfig{CertificateFile: "../testdata/cert.pem", PrivateKeyFile: "../testdata/cert.key"},
			expectedStatusCode: 200,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			cfg := &config.Config{
				Web: &web.Config{Address: "0.0.0.0", Port: rand.Intn(65534), TLS: scenario.tls},
				Endpoints: []*core.Endpoint{
					{Name: "frontend", Group: "core"},
					{Name: "backend", Group: "core"},
				},
			}
			if err := cfg.Web.ValidateAndSetDefaults(); err != nil {
				t.Error("expected no error from web (TLS) validation, got", err)
			}
			_ = os.Setenv("ROUTER_TEST", "true")
			_ = os.Setenv("ENVIRONMENT", "dev")
			defer os.Clearenv()
			Handle(cfg)
			defer Shutdown()
			request, _ := http.NewRequest("GET", "/health", http.NoBody)
			responseRecorder := httptest.NewRecorder()
			server.Handler.ServeHTTP(responseRecorder, request)
			if responseRecorder.Code != scenario.expectedStatusCode {
				t.Errorf("expected GET /health to return status code %d, got %d", scenario.expectedStatusCode, responseRecorder.Code)
			}
			if server == nil {
				t.Fatal("server should've been set (but because we set ROUTER_TEST, it shouldn't have been started)")
			}
		})
	}
}

func TestShutdown(t *testing.T) {
	// Pretend that we called controller.Handle(), which initializes the server variable
	server = &http.Server{}
	Shutdown()
	if server != nil {
		t.Error("server should've been shut down")
	}
}

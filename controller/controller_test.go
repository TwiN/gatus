package controller

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/web"
	"github.com/gofiber/fiber/v2"
)

func TestHandle(t *testing.T) {
	cfg := &config.Config{
		Web: &web.Config{
			Address: "0.0.0.0",
			Port:    rand.Intn(65534),
		},
		Endpoints: []*endpoint.Endpoint{
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
	request := httptest.NewRequest("GET", "/health", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 {
		t.Error("expected GET /health to return status code 200")
	}
	if app == nil {
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
				Endpoints: []*endpoint.Endpoint{
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
			request := httptest.NewRequest("GET", "/health", http.NoBody)
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != scenario.expectedStatusCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.expectedStatusCode, response.StatusCode)
			}
			if app == nil {
				t.Fatal("server should've been set (but because we set ROUTER_TEST, it shouldn't have been started)")
			}
		})
	}
}

func TestShutdown(t *testing.T) {
	// Pretend that we called controller.Handle(), which initializes the server variable
	app = fiber.New()
	Shutdown()
	if app != nil {
		t.Error("server should've been shut down")
	}
}

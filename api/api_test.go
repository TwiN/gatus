package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/ui"
	"github.com/TwiN/gatus/v5/config/web"
	"github.com/TwiN/gatus/v5/security"
	"github.com/gofiber/fiber/v2"
)

func TestNew(t *testing.T) {
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		Gzip         bool
		WithSecurity bool
	}
	scenarios := []Scenario{
		{
			Name:         "health",
			Path:         "/health",
			ExpectedCode: fiber.StatusOK,
		},
		{
			Name:         "custom.css",
			Path:         "/css/custom.css",
			ExpectedCode: fiber.StatusOK,
		},
		{
			Name:         "custom.css-gzipped",
			Path:         "/css/custom.css",
			ExpectedCode: fiber.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "metrics",
			Path:         "/metrics",
			ExpectedCode: fiber.StatusOK,
		},
		{
			Name:         "favicon.ico",
			Path:         "/favicon.ico",
			ExpectedCode: fiber.StatusOK,
		},
		{
			Name:         "app.js",
			Path:         "/js/app.js",
			ExpectedCode: fiber.StatusOK,
		},
		{
			Name:         "app.js-gzipped",
			Path:         "/js/app.js",
			ExpectedCode: fiber.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "chunk-vendors.js",
			Path:         "/js/chunk-vendors.js",
			ExpectedCode: fiber.StatusOK,
		},
		{
			Name:         "chunk-vendors.js-gzipped",
			Path:         "/js/chunk-vendors.js",
			ExpectedCode: fiber.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "index",
			Path:         "/",
			ExpectedCode: fiber.StatusOK,
		},
		{
			Name:         "index-html-redirect",
			Path:         "/index.html",
			ExpectedCode: fiber.StatusMovedPermanently,
		},
		{
			Name:         "index-should-return-200-even-if-not-authenticated",
			Path:         "/",
			ExpectedCode: fiber.StatusOK,
			WithSecurity: true,
		},
		{
			Name:         "endpoints-should-return-401-if-not-authenticated",
			Path:         "/api/v1/endpoints/statuses",
			ExpectedCode: fiber.StatusUnauthorized,
			WithSecurity: true,
		},
		{
			Name:         "config-should-return-200-even-if-not-authenticated",
			Path:         "/api/v1/config",
			ExpectedCode: fiber.StatusOK,
			WithSecurity: true,
		},
		{
			Name:         "config-should-always-return-200",
			Path:         "/api/v1/config",
			ExpectedCode: fiber.StatusOK,
			WithSecurity: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg := &config.Config{Metrics: true, UI: &ui.Config{}}
			if scenario.WithSecurity {
				cfg.Security = &security.Config{
					Basic: &security.BasicConfig{
						Username:                        "john.doe",
						PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
					},
				}
			}
			api := New(cfg)
			router := api.Router()
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			if scenario.Gzip {
				request.Header.Set("Accept-Encoding", "gzip")
			}
			response, err := router.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
			}
		})
	}
}

func TestCORS(t *testing.T) {
	t.Run("cors-disabled-by-default", func(t *testing.T) {
		cfg := &config.Config{
			Web: &web.Config{},
		}
		if err := cfg.Web.ValidateAndSetDefaults(); err != nil {
			t.Fatal(err)
		}
		api := New(cfg)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", http.NoBody)
		req.Header.Set("Origin", "https://dashboard.example.com")
		resp, err := api.Router().Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if val := resp.Header.Get("Access-Control-Allow-Origin"); val != "" {
			t.Errorf("expected no Access-Control-Allow-Origin header, got %s", val)
		}
	})

	t.Run("cors-enabled-preflight", func(t *testing.T) {
		cfg := &config.Config{
			Web: &web.Config{
				CORS: &web.CORSConfig{
					AllowOrigins:     "https://dashboard.example.com",
					AllowCredentials: true,
				},
			},
		}
		if err := cfg.Web.ValidateAndSetDefaults(); err != nil {
			t.Fatal(err)
		}
		api := New(cfg)

		req := httptest.NewRequest(http.MethodOptions, "/api/v1/config", http.NoBody)
		req.Header.Set("Origin", "https://dashboard.example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")
		resp, err := api.Router().Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if val := resp.Header.Get("Access-Control-Allow-Origin"); val != "https://dashboard.example.com" {
			t.Errorf("expected Access-Control-Allow-Origin to be https://dashboard.example.com, got %s", val)
		}
		if val := resp.Header.Get("Access-Control-Allow-Credentials"); val != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials to be true, got %s", val)
		}
	})

	t.Run("cors-enabled-get", func(t *testing.T) {
		cfg := &config.Config{
			Web: &web.Config{
				CORS: &web.CORSConfig{
					AllowOrigins: "https://dashboard.example.com",
				},
			},
		}
		if err := cfg.Web.ValidateAndSetDefaults(); err != nil {
			t.Fatal(err)
		}
		api := New(cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", http.NoBody)
		req.Header.Set("Origin", "https://dashboard.example.com")
		resp, err := api.Router().Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if val := resp.Header.Get("Access-Control-Allow-Origin"); val != "https://dashboard.example.com" {
			t.Errorf("expected Access-Control-Allow-Origin to be https://dashboard.example.com, got %s", val)
		}
	})

	t.Run("cors-disallowed-origin", func(t *testing.T) {
		cfg := &config.Config{
			Web: &web.Config{
				CORS: &web.CORSConfig{
					AllowOrigins: "https://dashboard.example.com",
				},
			},
		}
		if err := cfg.Web.ValidateAndSetDefaults(); err != nil {
			t.Fatal(err)
		}
		api := New(cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", http.NoBody)
		req.Header.Set("Origin", "https://malicious.example.com")
		resp, err := api.Router().Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if val := resp.Header.Get("Access-Control-Allow-Origin"); val != "" {
			t.Errorf("expected no Access-Control-Allow-Origin header for disallowed origin, got %s", val)
		}
	})

	t.Run("cors-multiple-comma-separated-origins", func(t *testing.T) {
		cfg := &config.Config{
			Web: &web.Config{
				CORS: &web.CORSConfig{
					AllowOrigins: "https://dashboard.example.com, https://other.example.com",
				},
			},
		}
		if err := cfg.Web.ValidateAndSetDefaults(); err != nil {
			t.Fatal(err)
		}
		api := New(cfg)

		// First origin in comma-separated list
		req1 := httptest.NewRequest(http.MethodGet, "/api/v1/config", http.NoBody)
		req1.Header.Set("Origin", "https://dashboard.example.com")
		resp1, err := api.Router().Test(req1)
		if err != nil {
			t.Fatal(err)
		}
		if val := resp1.Header.Get("Access-Control-Allow-Origin"); val != "https://dashboard.example.com" {
			t.Errorf("expected Access-Control-Allow-Origin to match first origin, got %s", val)
		}

		// Second origin in comma-separated list
		req2 := httptest.NewRequest(http.MethodGet, "/api/v1/config", http.NoBody)
		req2.Header.Set("Origin", "https://other.example.com")
		resp2, err := api.Router().Test(req2)
		if err != nil {
			t.Fatal(err)
		}
		if val := resp2.Header.Get("Access-Control-Allow-Origin"); val != "https://other.example.com" {
			t.Errorf("expected Access-Control-Allow-Origin to match second origin, got %s", val)
		}

		// Origin not in comma-separated list
		req3 := httptest.NewRequest(http.MethodGet, "/api/v1/config", http.NoBody)
		req3.Header.Set("Origin", "https://notallowed.example.com")
		resp3, err := api.Router().Test(req3)
		if err != nil {
			t.Fatal(err)
		}
		if val := resp3.Header.Get("Access-Control-Allow-Origin"); val != "" {
			t.Errorf("expected no Access-Control-Allow-Origin header for disallowed origin, got %s", val)
		}
	})

	t.Run("cors-dev-environment-fallback", func(t *testing.T) {
		t.Setenv("ENVIRONMENT", "dev")
		cfg := &config.Config{
			Web: &web.Config{},
		}
		if err := cfg.Web.ValidateAndSetDefaults(); err != nil {
			t.Fatal(err)
		}
		api := New(cfg)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", http.NoBody)
		req.Header.Set("Origin", "http://localhost:8081")
		resp, err := api.Router().Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if val := resp.Header.Get("Access-Control-Allow-Origin"); val != "http://localhost:8081" {
			t.Errorf("expected Access-Control-Allow-Origin to be http://localhost:8081 in dev mode, got %s", val)
		}
		if val := resp.Header.Get("Access-Control-Allow-Credentials"); val != "true" {
			t.Errorf("expected Access-Control-Allow-Credentials to be true in dev mode, got %s", val)
		}
	})
}


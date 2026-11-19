package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/ui"
	"github.com/TwiN/gatus/v5/security"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/gofiber/fiber/v2"
)

func TestNew(t *testing.T) {
	type Scenario struct {
		Name                string
		Path                string
		ExpectedCode        int
		Gzip                bool
		WithSecurity        bool
		WithMetricsSecurity bool
		Authorization       string
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
		{
			Name:         "metrics-should-return-200-if-not-protected",
			Path:         "/metrics",
			ExpectedCode: fiber.StatusOK,
			WithSecurity: true,
		},
		{
			Name:                "metrics-should-return-401-if-protected-and-not-authenticated",
			Path:                "/metrics",
			ExpectedCode:        fiber.StatusUnauthorized,
			WithSecurity:        true,
			WithMetricsSecurity: true,
		},
		{
			Name:                "metrics-should-return-200-if-protected-and-authenticated-with-metrics-token",
			Path:                "/metrics",
			ExpectedCode:        fiber.StatusOK,
			WithSecurity:        true,
			WithMetricsSecurity: true,
			Authorization:       "Bearer metrics-token",
		},
		{
			Name:                "metrics-should-return-401-if-protected-and-authenticated-with-api-token",
			Path:                "/metrics",
			ExpectedCode:        fiber.StatusUnauthorized,
			WithSecurity:        true,
			WithMetricsSecurity: true,
			Authorization:       "Bearer api-token",
		},
		{
			Name:                "health-should-return-200-if-metrics-protected",
			Path:                "/health",
			ExpectedCode:        fiber.StatusOK,
			WithSecurity:        true,
			WithMetricsSecurity: true,
		},
		{
			Name:                "config-should-return-200-if-metrics-protected",
			Path:                "/api/v1/config",
			ExpectedCode:        fiber.StatusOK,
			WithSecurity:        true,
			WithMetricsSecurity: true,
		},
		{
			Name:          "endpoints-should-return-200-if-authenticated-with-api-token",
			Path:          "/api/v1/endpoints/statuses",
			ExpectedCode:  fiber.StatusOK,
			WithSecurity:  true,
			Authorization: "Bearer api-token",
		},
		{
			Name:                "endpoints-should-return-401-if-authenticated-with-metrics-token",
			Path:                "/api/v1/endpoints/statuses",
			ExpectedCode:        fiber.StatusUnauthorized,
			WithSecurity:        true,
			WithMetricsSecurity: true,
			Authorization:       "Bearer metrics-token",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg := &config.Config{Metrics: true, UI: &ui.Config{}, Storage: &storage.Config{MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults}}
			if scenario.WithSecurity {
				cfg.Security = &security.Config{
					Basic: &security.BasicConfig{
						Username:                        "john.doe",
						PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
						APITokens:                       []string{"api-token"},
					},
				}
				if scenario.WithMetricsSecurity {
					cfg.Security.Metrics = &security.MetricsConfig{Tokens: []string{"metrics-token"}}
				}
			}
			api := New(cfg)
			router := api.Router()
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			if scenario.Gzip {
				request.Header.Set("Accept-Encoding", "gzip")
			}
			if len(scenario.Authorization) > 0 {
				request.Header.Set("Authorization", scenario.Authorization)
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

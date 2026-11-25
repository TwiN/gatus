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

// TestNew_WithBasePath verifies that when web.base-path is set, Gatus serves the UI and API
// natively under that subpath (no reverse-proxy prefix-stripping required), while /health and
// /metrics remain at the root and the root path redirects to the base path.
func TestNew_WithBasePath(t *testing.T) {
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		WithSecurity bool
	}
	scenarios := []Scenario{
		// UI and static assets are served under the base path
		{Name: "index-under-base-path", Path: "/gatus/", ExpectedCode: fiber.StatusOK},
		{Name: "index-bare-base-path", Path: "/gatus", ExpectedCode: fiber.StatusOK}, // StrictRouting is off
		{Name: "spa-deeplink-under-base-path", Path: "/gatus/endpoints/_", ExpectedCode: fiber.StatusOK},
		{Name: "suite-deeplink-under-base-path", Path: "/gatus/suites/_", ExpectedCode: fiber.StatusOK},
		{Name: "custom-css-under-base-path", Path: "/gatus/css/custom.css", ExpectedCode: fiber.StatusOK},
		{Name: "app.js-under-base-path", Path: "/gatus/js/app.js", ExpectedCode: fiber.StatusOK},
		{Name: "favicon-under-base-path", Path: "/gatus/favicon.ico", ExpectedCode: fiber.StatusOK},
		{Name: "api-under-base-path", Path: "/gatus/api/v1/config", ExpectedCode: fiber.StatusOK},
		{Name: "index-html-redirect-under-base-path", Path: "/gatus/index.html", ExpectedCode: fiber.StatusMovedPermanently},
		// Health and metrics are served under the base path too, like everything else
		{Name: "health-under-base-path", Path: "/gatus/health", ExpectedCode: fiber.StatusOK},
		{Name: "metrics-under-base-path", Path: "/gatus/metrics", ExpectedCode: fiber.StatusOK},
		// Root redirects to the base path, and nothing else is served at the root anymore
		{Name: "root-redirects-to-base-path", Path: "/", ExpectedCode: fiber.StatusFound},
		{Name: "api-not-at-root", Path: "/api/v1/config", ExpectedCode: fiber.StatusNotFound},
		{Name: "spa-not-at-root", Path: "/endpoints/_", ExpectedCode: fiber.StatusNotFound},
		{Name: "health-not-at-root", Path: "/health", ExpectedCode: fiber.StatusNotFound},
		{Name: "metrics-not-at-root", Path: "/metrics", ExpectedCode: fiber.StatusNotFound},
		// Security still applies under the base path
		{Name: "protected-api-under-base-path-requires-auth", Path: "/gatus/api/v1/endpoints/statuses", ExpectedCode: fiber.StatusUnauthorized, WithSecurity: true},
		{Name: "unprotected-config-under-base-path", Path: "/gatus/api/v1/config", ExpectedCode: fiber.StatusOK, WithSecurity: true},
		{Name: "index-under-base-path-even-if-not-authenticated", Path: "/gatus/", ExpectedCode: fiber.StatusOK, WithSecurity: true},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg := &config.Config{
				Metrics: true,
				Web:     &web.Config{BasePath: "/gatus/", ReadBufferSize: web.DefaultReadBufferSize},
				// UI.BasePath mirrors the cross-config propagation done in config.ValidateAndSetDefaults
				UI: &ui.Config{BasePath: "/gatus/"},
			}
			if scenario.WithSecurity {
				cfg.Security = &security.Config{
					Basic: &security.BasicConfig{
						Username:                        "john.doe",
						PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
					},
				}
			}
			api := New(cfg)
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			response, err := api.Router().Test(request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
			}
			if scenario.Path == "/" && response.StatusCode == fiber.StatusFound {
				if location := response.Header.Get("Location"); location != "/gatus/" {
					t.Errorf("root should redirect to /gatus/, but redirected to %q", location)
				}
			}
		})
	}
}

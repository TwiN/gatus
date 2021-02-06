package controller

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/watchdog"
)

func TestCreateRouter(t *testing.T) {
	staticFolder = "../web/static"
	cfg := &config.Config{
		Metrics: true,
		Services: []*core.Service{
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
	watchdog.UpdateServiceStatuses(cfg.Services[0], &core.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now()})
	watchdog.UpdateServiceStatuses(cfg.Services[1], &core.Result{Success: false, Duration: time.Second, Timestamp: time.Now()})
	router := CreateRouter(cfg)
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		Gzip         bool
	}
	scenarios := []Scenario{
		{
			Name:         "health",
			Path:         "/health",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "metrics",
			Path:         "/metrics",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badges-1h",
			Path:         "/api/v1/badges/uptime/1h/core_frontend.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badges-24h",
			Path:         "/api/v1/badges/uptime/24h/core_backend.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badges-7d",
			Path:         "/api/v1/badges/uptime/7d/core_frontend.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badges-with-invalid-duration",
			Path:         "/api/v1/badges/uptime/3d/core_backend.svg",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "badges-for-invalid-key",
			Path:         "/api/v1/badges/uptime/7d/invalid_key.svg",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "service-statuses",
			Path:         "/api/v1/statuses",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "service-statuses-gzip",
			Path:         "/api/v1/statuses",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "service-status",
			Path:         "/api/v1/statuses/core_frontend",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "service-status-gzip",
			Path:         "/api/v1/statuses/core_frontend",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "service-status-for-invalid-key",
			Path:         "/api/v1/statuses/invalid_key",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "favicon",
			Path:         "/favicon.ico",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "frontend-home",
			Path:         "/",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "frontend-assets",
			Path:         "/js/app.js",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "frontend-service",
			Path:         "/services/core_frontend",
			ExpectedCode: http.StatusOK,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request, _ := http.NewRequest("GET", scenario.Path, nil)
			if scenario.Gzip {
				request.Header.Set("Accept-Encoding", "gzip")
			}
			responseRecorder := httptest.NewRecorder()
			router.ServeHTTP(responseRecorder, request)
			if responseRecorder.Code != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, responseRecorder.Code)
			}
		})
	}
}

func TestHandle(t *testing.T) {
	cfg := &config.Config{
		Web: &config.WebConfig{
			Address: "0.0.0.0",
			Port:    rand.Intn(65534),
		},
		Services: []*core.Service{
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
	config.Set(cfg)
	_ = os.Setenv("ROUTER_TEST", "true")
	_ = os.Setenv("ENVIRONMENT", "dev")
	defer os.Clearenv()
	Handle()
	request, _ := http.NewRequest("GET", "/health", nil)
	responseRecorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusOK {
		t.Error("expected GET /health to return status code 200")
	}
	if server == nil {
		t.Fatal("server should've been set (but because we set ROUTER_TEST, it shouldn't have been started)")
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

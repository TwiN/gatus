package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

func TestResponseTimeChart(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Metrics: true,
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
	watchdog.UpdateEndpointStatus(cfg.Endpoints[0], &endpoint.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now()})
	watchdog.UpdateEndpointStatus(cfg.Endpoints[1], &endpoint.Result{Success: false, Duration: time.Second, Timestamp: time.Now()})
	api := New(cfg)
	router := api.Router()
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		Gzip         bool
	}
	scenarios := []Scenario{
		{
			Name:         "chart-response-time-24h",
			Path:         "/api/v1/endpoints/core_backend/response-times/24h/chart.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "chart-response-time-7d",
			Path:         "/api/v1/endpoints/core_frontend/response-times/7d/chart.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "chart-response-time-30d",
			Path:         "/api/v1/endpoints/core_frontend/response-times/30d/chart.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "chart-response-time-with-invalid-duration",
			Path:         "/api/v1/endpoints/core_backend/response-times/3d/chart.svg",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "chart-response-time-for-invalid-key",
			Path:         "/api/v1/endpoints/invalid_key/response-times/7d/chart.svg",
			ExpectedCode: http.StatusNotFound,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			if scenario.Gzip {
				request.Header.Set("Accept-Encoding", "gzip")
			}
			response, err := router.Test(request)
			if err != nil {
				return
			}
			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
			}
		})
	}
}

func TestResponseTimeHistory(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Metrics: true,
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
	watchdog.UpdateEndpointStatus(cfg.Endpoints[0], &endpoint.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now()})
	watchdog.UpdateEndpointStatus(cfg.Endpoints[1], &endpoint.Result{Success: false, Duration: time.Second, Timestamp: time.Now()})
	api := New(cfg)
	router := api.Router()
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
	}
	scenarios := []Scenario{
		{
			Name:         "history-response-time-24h",
			Path:         "/api/v1/endpoints/core_backend/response-times/24h/history",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "history-response-time-7d",
			Path:         "/api/v1/endpoints/core_frontend/response-times/7d/history",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "history-response-time-30d",
			Path:         "/api/v1/endpoints/core_frontend/response-times/30d/history",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "history-response-time-with-invalid-duration",
			Path:         "/api/v1/endpoints/core_backend/response-times/3d/history",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "history-response-time-for-invalid-key",
			Path:         "/api/v1/endpoints/invalid_key/response-times/7d/history",
			ExpectedCode: http.StatusNotFound,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
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

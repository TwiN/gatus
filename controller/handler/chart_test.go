package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

func TestResponseTimeChart(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Metrics: true,
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
	watchdog.UpdateEndpointStatuses(cfg.Endpoints[0], &core.Result{Success: true, Duration: time.Millisecond, Timestamp: time.Now()})
	watchdog.UpdateEndpointStatuses(cfg.Endpoints[1], &core.Result{Success: false, Duration: time.Second, Timestamp: time.Now()})
	router := CreateRouter(cfg)
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
			request, _ := http.NewRequest("GET", scenario.Path, http.NoBody)
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

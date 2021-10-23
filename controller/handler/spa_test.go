package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TwiN/gatus/v3/config"
	"github.com/TwiN/gatus/v3/core"
	"github.com/TwiN/gatus/v3/storage"
	"github.com/TwiN/gatus/v3/watchdog"
)

func TestSinglePageApplication(t *testing.T) {
	defer storage.Get().Clear()
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
	router := CreateRouter("../../web/static", cfg.Security, nil, cfg.Metrics)
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		Gzip         bool
	}
	scenarios := []Scenario{
		{
			Name:         "frontend-home",
			Path:         "/",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "frontend-endpoint",
			Path:         "/endpoints/core_frontend",
			ExpectedCode: http.StatusOK,
		},
		{ // XXX: Remove this in v4.0.0
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

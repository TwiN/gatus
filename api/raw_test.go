package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/ui"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

func TestRawDataEndpoint(t *testing.T) {
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

	cfg.Endpoints[0].UIConfig = ui.GetDefaultConfig()
	cfg.Endpoints[1].UIConfig = ui.GetDefaultConfig()

	watchdog.UpdateEndpointStatus(cfg.Endpoints[0], &endpoint.Result{Success: true, Connected: true, Duration: time.Millisecond, Timestamp: time.Now()})
	watchdog.UpdateEndpointStatus(cfg.Endpoints[1], &endpoint.Result{Success: false, Connected: false, Duration: time.Second, Timestamp: time.Now()})
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
			Name:         "raw-uptime-1h",
			Path:         "/api/v1/endpoints/core_frontend/uptimes/1h",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "raw-uptime-24h",
			Path:         "/api/v1/endpoints/core_backend/uptimes/24h",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "raw-uptime-7d",
			Path:         "/api/v1/endpoints/core_frontend/uptimes/7d",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "raw-uptime-30d",
			Path:         "/api/v1/endpoints/core_frontend/uptimes/30d",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "raw-uptime-with-invalid-duration",
			Path:         "/api/v1/endpoints/core_backend/uptimes/3d",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "raw-uptime-for-invalid-key",
			Path:         "/api/v1/endpoints/invalid_key/uptimes/7d",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "raw-response-times-1h",
			Path:         "/api/v1/endpoints/core_frontend/response-times/1h",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "raw-response-times-24h",
			Path:         "/api/v1/endpoints/core_backend/response-times/24h",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "raw-response-times-7d",
			Path:         "/api/v1/endpoints/core_frontend/response-times/7d",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "raw-response-times-30d",
			Path:         "/api/v1/endpoints/core_frontend/response-times/30d",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "raw-response-times-with-invalid-duration",
			Path:         "/api/v1/endpoints/core_backend/response-times/3d",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "raw-response-times-for-invalid-key",
			Path:         "/api/v1/endpoints/invalid_key/response-times/7d",
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

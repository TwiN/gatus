package api

import (
	"encoding/json"
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

func TestPeriodStatuses(t *testing.T) {
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
	}

	scenarios := []Scenario{
		{
			Name:         "period-statuses-24h-10",
			Path:         "/api/v1/endpoints/core_frontend/period-statuses/24h/10",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "period-statuses-7d-20",
			Path:         "/api/v1/endpoints/core_backend/period-statuses/7d/20",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "period-statuses-30d-30",
			Path:         "/api/v1/endpoints/core_frontend/period-statuses/30d/30",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "period-statuses-90d-50",
			Path:         "/api/v1/endpoints/core_frontend/period-statuses/90d/50",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "period-statuses-custom-14d-14",
			Path:         "/api/v1/endpoints/core_frontend/period-statuses/14d/14",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "period-statuses-1h-1",
			Path:         "/api/v1/endpoints/core_frontend/period-statuses/1h/1",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "period-statuses-invalid-duration",
			Path:         "/api/v1/endpoints/core_backend/period-statuses/3d/10",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "period-statuses-invalid-parts-zero",
			Path:         "/api/v1/endpoints/core_backend/period-statuses/24h/0",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "period-statuses-invalid-parts-negative",
			Path:         "/api/v1/endpoints/core_backend/period-statuses/24h/-1",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "period-statuses-invalid-parts-over-100",
			Path:         "/api/v1/endpoints/core_backend/period-statuses/24h/101",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "period-statuses-invalid-key",
			Path:         "/api/v1/endpoints/invalid_key/period-statuses/24h/10",
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

func TestPeriodStatusesResponseStructure(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Metrics: true,
		Endpoints: []*endpoint.Endpoint{
			{
				Name:  "frontend",
				Group: "core",
			},
		},
	}
	cfg.Endpoints[0].UIConfig = ui.GetDefaultConfig()
	watchdog.UpdateEndpointStatus(cfg.Endpoints[0], &endpoint.Result{Success: true, Connected: true, Duration: 150 * time.Millisecond, Timestamp: time.Now()})
	api := New(cfg)
	router := api.Router()

	request := httptest.NewRequest("GET", "/api/v1/endpoints/core_frontend/period-statuses/24h/5", http.NoBody)
	response, err := router.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}

	var result PeriodStatusResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result.Duration != "24h" {
		t.Errorf("expected duration '24h', got '%s'", result.Duration)
	}
	if result.Parts != 5 {
		t.Errorf("expected parts 5, got %d", result.Parts)
	}
	if result.Uptime < 0 || result.Uptime > 1 {
		t.Errorf("uptime must be in [0, 1], got %f", result.Uptime)
	}
	if len(result.Results) != 5 {
		t.Fatalf("expected 5 results, got %d", len(result.Results))
	}
	for i, r := range result.Results {
		if r.Timestamp.IsZero() {
			t.Errorf("result %d: timestamp must not be zero", i)
		}
		if !r.Missing {
			// Non-missing results should have valid data
			if r.Duration < 0 {
				t.Errorf("result %d: duration must be non-negative, got %v", i, r.Duration)
			}
		}
	}
	// Verify timestamps are in ascending order
	for i := 1; i < len(result.Results); i++ {
		if !result.Results[i].Timestamp.After(result.Results[i-1].Timestamp) {
			t.Errorf("result %d timestamp should be after result %d timestamp", i, i-1)
		}
	}
}

func TestPeriodStatusesSinglePart(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	cfg := &config.Config{
		Metrics: true,
		Endpoints: []*endpoint.Endpoint{
			{
				Name:  "frontend",
				Group: "core",
			},
		},
	}
	cfg.Endpoints[0].UIConfig = ui.GetDefaultConfig()
	watchdog.UpdateEndpointStatus(cfg.Endpoints[0], &endpoint.Result{Success: true, Connected: true, Duration: 100 * time.Millisecond, Timestamp: time.Now()})
	api := New(cfg)
	router := api.Router()

	request := httptest.NewRequest("GET", "/api/v1/endpoints/core_frontend/period-statuses/24h/1", http.NoBody)
	response, err := router.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}

	var result PeriodStatusResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(result.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(result.Results))
	}
	if result.Uptime < 0 || result.Uptime > 1 {
		t.Errorf("uptime must be in [0, 1], got %f", result.Uptime)
	}
}

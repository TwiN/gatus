package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/storage"
	"github.com/TwinProduction/gatus/watchdog"
)

func TestUptimeBadge(t *testing.T) {
	defer storage.Get().Clear()
	defer cache.Clear()
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
	router := CreateRouter("../../web/static", cfg.Security, nil, cfg.Metrics)
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		Gzip         bool
	}
	scenarios := []Scenario{
		{
			Name:         "badge-uptime-1h",
			Path:         "/api/v1/services/core_frontend/uptimes/1h/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-uptime-24h",
			Path:         "/api/v1/services/core_backend/uptimes/24h/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-uptime-7d",
			Path:         "/api/v1/services/core_frontend/uptimes/7d/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-uptime-with-invalid-duration",
			Path:         "/api/v1/services/core_backend/uptimes/3d/badge.svg",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "badge-uptime-for-invalid-key",
			Path:         "/api/v1/services/invalid_key/uptimes/7d/badge.svg",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "badge-response-time-1h",
			Path:         "/api/v1/services/core_frontend/response-times/1h/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-response-time-24h",
			Path:         "/api/v1/services/core_backend/response-times/24h/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-response-time-7d",
			Path:         "/api/v1/services/core_frontend/response-times/7d/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-response-time-with-invalid-duration",
			Path:         "/api/v1/services/core_backend/response-times/3d/badge.svg",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "badge-response-time-for-invalid-key",
			Path:         "/api/v1/services/invalid_key/response-times/7d/badge.svg",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "chart-response-time-24h",
			Path:         "/api/v1/services/core_backend/response-times/24h/chart.svg",
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

func TestGetBadgeColorFromUptime(t *testing.T) {
	scenarios := []struct {
		Uptime        float64
		ExpectedColor string
	}{
		{
			Uptime:        1,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			Uptime:        0.99,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			Uptime:        0.97,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			Uptime:        0.95,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			Uptime:        0.93,
			ExpectedColor: badgeColorHexGood,
		},
		{
			Uptime:        0.9,
			ExpectedColor: badgeColorHexGood,
		},
		{
			Uptime:        0.85,
			ExpectedColor: badgeColorHexPassable,
		},
		{
			Uptime:        0.7,
			ExpectedColor: badgeColorHexBad,
		},
		{
			Uptime:        0.65,
			ExpectedColor: badgeColorHexBad,
		},
		{
			Uptime:        0.6,
			ExpectedColor: badgeColorHexVeryBad,
		},
	}
	for _, scenario := range scenarios {
		t.Run("uptime-"+strconv.Itoa(int(scenario.Uptime*100)), func(t *testing.T) {
			if getBadgeColorFromUptime(scenario.Uptime) != scenario.ExpectedColor {
				t.Errorf("expected %s from %f, got %v", scenario.ExpectedColor, scenario.Uptime, getBadgeColorFromUptime(scenario.Uptime))
			}
		})
	}
}

func TestGetBadgeColorFromResponseTime(t *testing.T) {
	scenarios := []struct {
		ResponseTime  int
		ExpectedColor string
	}{
		{
			ResponseTime:  10,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			ResponseTime:  50,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			ResponseTime:  75,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			ResponseTime:  150,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			ResponseTime:  201,
			ExpectedColor: badgeColorHexGood,
		},
		{
			ResponseTime:  300,
			ExpectedColor: badgeColorHexGood,
		},
		{
			ResponseTime:  301,
			ExpectedColor: badgeColorHexPassable,
		},
		{
			ResponseTime:  450,
			ExpectedColor: badgeColorHexPassable,
		},
		{
			ResponseTime:  700,
			ExpectedColor: badgeColorHexBad,
		},
		{
			ResponseTime:  1500,
			ExpectedColor: badgeColorHexVeryBad,
		},
	}
	for _, scenario := range scenarios {
		t.Run("response-time-"+strconv.Itoa(scenario.ResponseTime), func(t *testing.T) {
			if getBadgeColorFromResponseTime(scenario.ResponseTime) != scenario.ExpectedColor {
				t.Errorf("expected %s from %d, got %v", scenario.ExpectedColor, scenario.ResponseTime, getBadgeColorFromResponseTime(scenario.ResponseTime))
			}
		})
	}
}

package handler

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/core/ui"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

func TestBadge(t *testing.T) {
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

	testSuccessfulResult = core.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                nil,
		Connected:             true,
		Success:               true,
		Timestamp:             timestamp,
		Duration:              150 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*core.ConditionResult{
			{
				Condition: "[STATUS] == 200",
				Success:   true,
			},
			{
				Condition: "[RESPONSE_TIME] < 500",
				Success:   true,
			},
			{
				Condition: "[CERTIFICATE_EXPIRATION] < 72h",
				Success:   true,
			},
		},
	}

	cfg.Endpoints[0].UIConfig = ui.GetDefaultConfig()
	cfg.Endpoints[1].UIConfig = ui.GetDefaultConfig()

	watchdog.UpdateEndpointStatuses(cfg.Endpoints[0], &core.Result{Success: true, Connected: true, Duration: time.Millisecond, Timestamp: time.Now()})
	watchdog.UpdateEndpointStatuses(cfg.Endpoints[1], &core.Result{Success: false, Connected: false, Duration: time.Second, Timestamp: time.Now()})
	router := CreateRouter(cfg)
	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		Gzip         bool
	}
	scenarios := []Scenario{
		{
			Name:         "badge-uptime-1h",
			Path:         "/api/v1/endpoints/core_frontend/uptimes/1h/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-uptime-24h",
			Path:         "/api/v1/endpoints/core_backend/uptimes/24h/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-uptime-7d",
			Path:         "/api/v1/endpoints/core_frontend/uptimes/7d/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-uptime-with-invalid-duration",
			Path:         "/api/v1/endpoints/core_backend/uptimes/3d/badge.svg",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "badge-uptime-for-invalid-key",
			Path:         "/api/v1/endpoints/invalid_key/uptimes/7d/badge.svg",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "badge-response-time-1h",
			Path:         "/api/v1/endpoints/core_frontend/response-times/1h/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-response-time-24h",
			Path:         "/api/v1/endpoints/core_backend/response-times/24h/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-response-time-7d",
			Path:         "/api/v1/endpoints/core_frontend/response-times/7d/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-response-time-with-invalid-duration",
			Path:         "/api/v1/endpoints/core_backend/response-times/3d/badge.svg",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "badge-response-time-for-invalid-key",
			Path:         "/api/v1/endpoints/invalid_key/response-times/7d/badge.svg",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "badge-health-up",
			Path:         "/api/v1/endpoints/core_frontend/health/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-health-down",
			Path:         "/api/v1/endpoints/core_backend/health/badge.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "badge-health-for-invalid-key",
			Path:         "/api/v1/endpoints/invalid_key/health/badge.svg",
			ExpectedCode: http.StatusNotFound,
		},
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
	defer store.Get().Clear()
	defer cache.Clear()

	var (
		firstCondition  = core.Condition("[STATUS] == 200")
		secondCondition = core.Condition("[RESPONSE_TIME] < 500")
		thirdCondition  = core.Condition("[CERTIFICATE_EXPIRATION] < 72h")
	)

	firstTestEndpoint := core.Endpoint{
		Name:                    "a",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []core.Condition{firstCondition, secondCondition, thirdCondition},
		Alerts:                  nil,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
		UIConfig:                ui.GetDefaultConfig(),
	}
	secondTestEndpoint := core.Endpoint{
		Name:                    "b",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []core.Condition{firstCondition, secondCondition, thirdCondition},
		Alerts:                  nil,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
		UIConfig: &ui.Config{
			Badge: &ui.Badge{
				ResponseTime: &ui.ResponseTime{
					Thresholds: []int{100, 500, 1000, 2000, 3000},
				},
			},
		},
	}
	cfg := &config.Config{
		Metrics:   true,
		Endpoints: []*core.Endpoint{&firstTestEndpoint, &secondTestEndpoint},
	}

	store.Get().Insert(&firstTestEndpoint, &testSuccessfulResult)
	store.Get().Insert(&secondTestEndpoint, &testSuccessfulResult)

	scenarios := []struct {
		Key           string
		ResponseTime  int
		ExpectedColor string
	}{
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  10,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  50,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  75,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  150,
			ExpectedColor: badgeColorHexGreat,
		},
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  201,
			ExpectedColor: badgeColorHexGood,
		},
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  300,
			ExpectedColor: badgeColorHexGood,
		},
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  301,
			ExpectedColor: badgeColorHexPassable,
		},
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  450,
			ExpectedColor: badgeColorHexPassable,
		},
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  700,
			ExpectedColor: badgeColorHexBad,
		},
		{
			Key:           firstTestEndpoint.Key(),
			ResponseTime:  1500,
			ExpectedColor: badgeColorHexVeryBad,
		},
		{
			Key:           secondTestEndpoint.Key(),
			ResponseTime:  50,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			Key:           secondTestEndpoint.Key(),
			ResponseTime:  1500,
			ExpectedColor: badgeColorHexPassable,
		},
		{
			Key:           secondTestEndpoint.Key(),
			ResponseTime:  2222,
			ExpectedColor: badgeColorHexBad,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Key+"-response-time-"+strconv.Itoa(scenario.ResponseTime), func(t *testing.T) {
			if getBadgeColorFromResponseTime(scenario.ResponseTime, scenario.Key, cfg) != scenario.ExpectedColor {
				t.Errorf("expected %s from %d, got %v", scenario.ExpectedColor, scenario.ResponseTime, getBadgeColorFromResponseTime(scenario.ResponseTime, scenario.Key, cfg))
			}
		})
	}
}

func TestGetBadgeColorFromHealth(t *testing.T) {
	scenarios := []struct {
		HealthStatus  string
		ExpectedColor string
	}{
		{
			HealthStatus:  HealthStatusUp,
			ExpectedColor: badgeColorHexAwesome,
		},
		{
			HealthStatus:  HealthStatusDown,
			ExpectedColor: badgeColorHexVeryBad,
		},
		{
			HealthStatus:  HealthStatusUnknown,
			ExpectedColor: badgeColorHexPassable,
		},
	}
	for _, scenario := range scenarios {
		t.Run("health-"+scenario.HealthStatus, func(t *testing.T) {
			if getBadgeColorFromHealth(scenario.HealthStatus) != scenario.ExpectedColor {
				t.Errorf("expected %s from %s, got %v", scenario.ExpectedColor, scenario.HealthStatus, getBadgeColorFromHealth(scenario.HealthStatus))
			}
		})
	}
}

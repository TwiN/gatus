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
	"github.com/TwinProduction/gatus/storage"
	"github.com/TwinProduction/gatus/watchdog"
)

var (
	firstCondition  = core.Condition("[STATUS] == 200")
	secondCondition = core.Condition("[RESPONSE_TIME] < 500")
	thirdCondition  = core.Condition("[CERTIFICATE_EXPIRATION] < 72h")

	timestamp = time.Now()

	testService = core.Service{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []*core.Condition{&firstCondition, &secondCondition, &thirdCondition},
		Alerts:                  nil,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
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
	testUnsuccessfulResult = core.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                []string{"error-1", "error-2"},
		Connected:             true,
		Success:               false,
		Timestamp:             timestamp,
		Duration:              750 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*core.ConditionResult{
			{
				Condition: "[STATUS] == 200",
				Success:   true,
			},
			{
				Condition: "[RESPONSE_TIME] < 500",
				Success:   false,
			},
			{
				Condition: "[CERTIFICATE_EXPIRATION] < 72h",
				Success:   false,
			},
		},
	}
)

func TestCreateRouter(t *testing.T) {
	defer storage.Get().Clear()
	defer cache.Clear()
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
	router := CreateRouter(cfg.Security, cfg.Metrics)
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
			Name:         "old-badge-1h",
			Path:         "/api/v1/badges/uptime/1h/core_frontend.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "old-badge-24h",
			Path:         "/api/v1/badges/uptime/24h/core_backend.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "old-badge-7d",
			Path:         "/api/v1/badges/uptime/7d/core_frontend.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "old-badge-with-invalid-duration",
			Path:         "/api/v1/badges/uptime/3d/core_backend.svg",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "old-badge-for-invalid-key",
			Path:         "/api/v1/badges/uptime/7d/invalid_key.svg",
			ExpectedCode: http.StatusNotFound,
		},
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
		{
			Name:         "chart-response-time-7d",
			Path:         "/api/v1/services/core_frontend/response-times/7d/chart.svg",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "chart-response-time-with-invalid-duration",
			Path:         "/api/v1/services/core_backend/response-times/3d/chart.svg",
			ExpectedCode: http.StatusBadRequest,
		},
		{
			Name:         "chart-response-time-for-invalid-key",
			Path:         "/api/v1/services/invalid_key/response-times/7d/chart.svg",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "old-service-statuses",
			Path:         "/api/v1/statuses",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "old-service-statuses-gzip",
			Path:         "/api/v1/statuses",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "old-service-statuses-pagination",
			Path:         "/api/v1/statuses?page=1&pageSize=20",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "old-service-status",
			Path:         "/api/v1/statuses/core_frontend",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "old-service-status-gzip",
			Path:         "/api/v1/statuses/core_frontend",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "old-service-status-for-invalid-key",
			Path:         "/api/v1/statuses/invalid_key",
			ExpectedCode: http.StatusNotFound,
		},
		{
			Name:         "service-statuses",
			Path:         "/api/v1/services/statuses",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "service-statuses-gzip",
			Path:         "/api/v1/services/statuses",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "service-statuses-pagination",
			Path:         "/api/v1/services/statuses?page=1&pageSize=20",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "service-status",
			Path:         "/api/v1/services/core_frontend/statuses",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "service-status-gzip",
			Path:         "/api/v1/services/core_frontend/statuses",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "service-status-pagination",
			Path:         "/api/v1/services/core_frontend/statuses?page=1&pageSize=20",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "service-status-for-invalid-key",
			Path:         "/api/v1/services/invalid_key/statuses",
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
	defer storage.Get().Clear()
	defer cache.Clear()
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
	_ = os.Setenv("ROUTER_TEST", "true")
	_ = os.Setenv("ENVIRONMENT", "dev")
	defer os.Clearenv()
	Handle(cfg.Security, cfg.Web, cfg.Metrics)
	defer Shutdown()
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

func TestServiceStatusesHandler(t *testing.T) {
	defer storage.Get().Clear()
	defer cache.Clear()
	staticFolder = "../web/static"
	firstResult := &testSuccessfulResult
	secondResult := &testUnsuccessfulResult
	storage.Get().Insert(&testService, firstResult)
	storage.Get().Insert(&testService, secondResult)
	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
	firstResult.Timestamp = time.Time{}
	secondResult.Timestamp = time.Time{}
	router := CreateRouter(nil, false)

	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		ExpectedBody string
	}
	scenarios := []Scenario{
		{
			Name:         "no-pagination",
			Path:         "/api/v1/statuses",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `{"group_name":{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"errors":null,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}}`,
		},
		{
			Name:         "pagination-first-result",
			Path:         "/api/v1/statuses?page=1&pageSize=1",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `{"group_name":{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}}`,
		},
		{
			Name:         "pagination-second-result",
			Path:         "/api/v1/statuses?page=2&pageSize=1",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `{"group_name":{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"errors":null,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"}]}}`,
		},
		{
			Name:         "pagination-no-results",
			Path:         "/api/v1/statuses?page=5&pageSize=20",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `{"group_name":{"name":"name","group":"group","key":"group_name","results":[]}}`,
		},
		{
			Name:         "invalid-pagination-should-fall-back-to-default",
			Path:         "/api/v1/statuses?page=INVALID&pageSize=INVALID",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `{"group_name":{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"errors":null,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}}`,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request, _ := http.NewRequest("GET", scenario.Path, nil)
			responseRecorder := httptest.NewRecorder()
			router.ServeHTTP(responseRecorder, request)
			if responseRecorder.Code != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, responseRecorder.Code)
			}
			output := responseRecorder.Body.String()
			if output != scenario.ExpectedBody {
				t.Errorf("expected:\n %s\n\ngot:\n %s", scenario.ExpectedBody, output)
			}
		})
	}
}

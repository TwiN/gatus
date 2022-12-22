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

var (
	timestamp = time.Now()

	testEndpoint = core.Endpoint{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []core.Condition{core.Condition("[STATUS] == 200"), core.Condition("[RESPONSE_TIME] < 500"), core.Condition("[CERTIFICATE_EXPIRATION] < 72h")},
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

func TestEndpointStatus(t *testing.T) {
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
			Name:         "endpoint-status",
			Path:         "/api/v1/endpoints/core_frontend/statuses",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "endpoint-status-gzip",
			Path:         "/api/v1/endpoints/core_frontend/statuses",
			ExpectedCode: http.StatusOK,
			Gzip:         true,
		},
		{
			Name:         "endpoint-status-pagination",
			Path:         "/api/v1/endpoints/core_frontend/statuses?page=1&pageSize=20",
			ExpectedCode: http.StatusOK,
		},
		{
			Name:         "endpoint-status-for-invalid-key",
			Path:         "/api/v1/endpoints/invalid_key/statuses",
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

func TestEndpointStatuses(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	firstResult := &testSuccessfulResult
	secondResult := &testUnsuccessfulResult
	store.Get().Insert(&testEndpoint, firstResult)
	store.Get().Insert(&testEndpoint, secondResult)
	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
	firstResult.Timestamp = time.Time{}
	secondResult.Timestamp = time.Time{}
	router := CreateRouter(&config.Config{Metrics: true})

	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		ExpectedBody string
	}
	scenarios := []Scenario{
		{
			Name:         "no-pagination",
			Path:         "/api/v1/endpoints/statuses",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`,
		},
		{
			Name:         "pagination-first-result",
			Path:         "/api/v1/endpoints/statuses?page=1&pageSize=1",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`,
		},
		{
			Name:         "pagination-second-result",
			Path:         "/api/v1/endpoints/statuses?page=2&pageSize=1",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"}]}]`,
		},
		{
			Name:         "pagination-no-results",
			Path:         "/api/v1/endpoints/statuses?page=5&pageSize=20",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"name","group":"group","key":"group_name","results":[]}]`,
		},
		{
			Name:         "invalid-pagination-should-fall-back-to-default",
			Path:         "/api/v1/endpoints/statuses?page=INVALID&pageSize=INVALID",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request, _ := http.NewRequest("GET", scenario.Path, http.NoBody)
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

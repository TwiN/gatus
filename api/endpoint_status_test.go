package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/ui"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

var (
	timestamp = time.Now()

	testEndpoint = endpoint.Endpoint{
		Name:                    "name",
		Group:                   "group",
		URL:                     "https://example.org/what/ever",
		Method:                  "GET",
		Body:                    "body",
		Interval:                30 * time.Second,
		Conditions:              []endpoint.Condition{endpoint.Condition("[STATUS] == 200"), endpoint.Condition("[RESPONSE_TIME] < 500"), endpoint.Condition("[CERTIFICATE_EXPIRATION] < 72h")},
		Alerts:                  nil,
		NumberOfFailuresInARow:  0,
		NumberOfSuccessesInARow: 0,
	}
	testSuccessfulResult = endpoint.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                nil,
		Connected:             true,
		Success:               true,
		Timestamp:             timestamp,
		Duration:              150 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*endpoint.ConditionResult{
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
	testUnsuccessfulResult = endpoint.Result{
		Hostname:              "example.org",
		IP:                    "127.0.0.1",
		HTTPStatus:            200,
		Errors:                []string{"error-1", "error-2"},
		Connected:             true,
		Success:               false,
		Timestamp:             timestamp,
		Duration:              750 * time.Millisecond,
		CertificateExpiration: 10 * time.Hour,
		ConditionResults: []*endpoint.ConditionResult{
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
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
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

func TestEndpointStatuses(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	firstResult := &testSuccessfulResult
	secondResult := &testUnsuccessfulResult
	store.Get().InsertEndpointResult(&testEndpoint, firstResult)
	store.Get().InsertEndpointResult(&testEndpoint, secondResult)
	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
	firstResult.Timestamp = time.Time{}
	secondResult.Timestamp = time.Time{}
	api := New(&config.Config{
		Metrics: true,
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	})
	router := api.Router()
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
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			response, err := router.Test(request)
			if err != nil {
				return
			}
			defer response.Body.Close()
			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
			}
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Error("expected err to be nil, but was", err)
			}
			if string(body) != scenario.ExpectedBody {
				t.Errorf("expected:\n %s\n\ngot:\n %s", scenario.ExpectedBody, string(body))
			}
		})
	}
}

func TestEndpointStatusesWithDashboardHistory(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	endpointWithHistory := testEndpoint
	endpointWithHistory.Name = "with-history"
	endpointWithHistory.UIConfig = &ui.Config{DashboardHistory: time.Hour}
	endpointWithoutHistory := testEndpoint
	endpointWithoutHistory.Name = "without-history"
	endpointWithoutHistory.UIConfig = &ui.Config{}
	now := time.Now()
	for _, ep := range []*endpoint.Endpoint{&endpointWithHistory, &endpointWithoutHistory} {
		olderResult := testUnsuccessfulResult
		olderResult.Timestamp = now.Add(-30 * time.Minute)
		newerResult := testSuccessfulResult
		newerResult.Timestamp = now.Add(-time.Minute)
		store.Get().InsertEndpointResult(ep, &olderResult)
		store.Get().InsertEndpointResult(ep, &newerResult)
	}
	api := New(&config.Config{
		Endpoints: []*endpoint.Endpoint{&endpointWithHistory, &endpointWithoutHistory},
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	})
	router := api.Router()
	response, err := router.Test(httptest.NewRequest("GET", "/api/v1/endpoints/statuses", http.NoBody))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, response.StatusCode)
	}
	var endpointStatuses []*endpoint.Status
	if err := json.NewDecoder(response.Body).Decode(&endpointStatuses); err != nil {
		t.Fatal("failed to decode response:", err)
	}
	if len(endpointStatuses) != 2 {
		t.Fatalf("expected 2 endpoint statuses, got %d", len(endpointStatuses))
	}
	for _, endpointStatus := range endpointStatuses {
		switch endpointStatus.Key {
		case endpointWithHistory.Key():
			if endpointStatus.History == nil {
				t.Fatal("expected endpoint with dashboard-history to have a history")
			}
			if endpointStatus.History.Window != time.Hour {
				t.Errorf("expected history window of 1h, got %s", endpointStatus.History.Window)
			}
			if len(endpointStatus.History.Bins) != endpoint.NumberOfHistoryBins {
				t.Fatalf("expected %d bins, got %d", endpoint.NumberOfHistoryBins, len(endpointStatus.History.Bins))
			}
			var checks, successfulChecks int
			for _, bin := range endpointStatus.History.Bins {
				checks += bin.Checks
				successfulChecks += bin.SuccessfulChecks
			}
			if checks != 2 || successfulChecks != 1 {
				t.Errorf("expected 2 checks with 1 successful check across all bins, got %d checks with %d successful", checks, successfulChecks)
			}
			if len(endpointStatus.Results) != 2 {
				t.Errorf("expected results to still be returned alongside the history, got %d results", len(endpointStatus.Results))
			}
		case endpointWithoutHistory.Key():
			if endpointStatus.History != nil {
				t.Error("expected endpoint without dashboard-history to have no history")
			}
		default:
			t.Errorf("unexpected endpoint status with key %s", endpointStatus.Key)
		}
	}
}

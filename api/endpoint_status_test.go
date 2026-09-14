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

// TestEndpointStatus_SuspendedEndpointWithoutHistory makes sure that requesting the status of a suspended endpoint
// that has never been checked returns a placeholder built from the configuration (marked suspended) instead of a
// 404, so that it's still reachable from the UI.
func TestEndpointStatus_SuspendedEndpointWithoutHistory(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	suspended := &endpoint.Endpoint{Name: "suspended", Group: "core", Suspended: boolPtr(true)}
	cfg := &config.Config{
		Metrics:   true,
		Endpoints: []*endpoint.Endpoint{suspended},
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	}
	api := New(cfg)
	router := api.Router()
	request := httptest.NewRequest("GET", "/api/v1/endpoints/"+suspended.Key()+"/statuses", http.NoBody)
	response, err := router.Test(request)
	if err != nil {
		t.Fatal("expected err to be nil, but was", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, response.StatusCode)
	}
	var status endpoint.Status
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatal("failed to decode response body:", err)
	}
	if !status.Suspended {
		t.Error("expected the placeholder status to be marked suspended")
	}
	if status.Key != suspended.Key() {
		t.Errorf("expected key=%s, got %s", suspended.Key(), status.Key)
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

// TestEndpointStatuses_SuspendedEndpoints makes sure that suspended endpoints are annotated with suspended=true in
// the response whether or not they already have persisted results, and that endpoints which aren't suspended are
// unaffected.
func TestEndpointStatuses_SuspendedEndpoints(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	activeEndpoint := &endpoint.Endpoint{Name: "active", Group: "group"}
	suspendedWithHistory := &endpoint.Endpoint{Name: "suspended-with-history", Group: "group", Suspended: boolPtr(true)}
	suspendedWithoutHistory := &endpoint.Endpoint{Name: "suspended-without-history", Group: "group", Suspended: boolPtr(true)}
	watchdog.UpdateEndpointStatus(activeEndpoint, &endpoint.Result{Success: true, Timestamp: time.Now()})
	watchdog.UpdateEndpointStatus(suspendedWithHistory, &endpoint.Result{Success: false, Timestamp: time.Now()})
	cfg := &config.Config{
		Metrics:   true,
		Endpoints: []*endpoint.Endpoint{activeEndpoint, suspendedWithHistory, suspendedWithoutHistory},
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	}
	api := New(cfg)
	router := api.Router()
	request := httptest.NewRequest("GET", "/api/v1/endpoints/statuses", http.NoBody)
	response, err := router.Test(request)
	if err != nil {
		t.Fatal("expected err to be nil, but was", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, response.StatusCode)
	}
	var statuses []*endpoint.Status
	if err := json.NewDecoder(response.Body).Decode(&statuses); err != nil {
		t.Fatal("failed to decode response body:", err)
	}
	if len(statuses) != 3 {
		t.Fatalf("expected 3 endpoint statuses (including the suspended one with no history), got %d", len(statuses))
	}
	byKey := make(map[string]*endpoint.Status, len(statuses))
	for _, status := range statuses {
		byKey[status.Key] = status
	}
	if status, ok := byKey[activeEndpoint.Key()]; !ok || status.Suspended {
		t.Errorf("expected active endpoint to not be suspended, got %+v", status)
	}
	if status, ok := byKey[suspendedWithHistory.Key()]; !ok || !status.Suspended || len(status.Results) != 1 {
		t.Errorf("expected suspended endpoint with history to be marked suspended and keep its results, got %+v", status)
	}
	if status, ok := byKey[suspendedWithoutHistory.Key()]; !ok || !status.Suspended || len(status.Results) != 0 {
		t.Errorf("expected suspended endpoint without history to be marked suspended with no results, got %+v", status)
	}
}

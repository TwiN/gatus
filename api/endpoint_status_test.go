package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/test"
	"github.com/TwiN/gatus/v5/watchdog"
	"github.com/TwiN/logr"
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

	apiAURL = "https://a.example.com/api/v1/endpoints/statuses"
	apiBURL = "https://b.example.com/api/v1/endpoints/statuses"
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
				t.Error("Request failed or timed out", err)
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
				t.Error("Request failed or timed out", err)
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

// Here we test that a gatus instance respects the `remote` query param
// in the /api/v1/endpoints/statuses URL. We do not test the include remote.
func TestEndpointStatusesRespectsIncludeRemoteQuery(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	firstResult := &testSuccessfulResult
	secondResult := &testUnsuccessfulResult
	store.Get().InsertEndpointResult(&testEndpoint, firstResult)
	store.Get().InsertEndpointResult(&testEndpoint, secondResult)
	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
	firstResult.Timestamp = time.Time{}
	secondResult.Timestamp = time.Time{}

	cfg := &config.Config{
		Metrics: true,
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	}
	api := New(cfg)
	router := api.Router()

	remoteApi := New(&config.Config{
		Metrics: true,
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	})
	remoteRouter := remoteApi.Router()

	mockRoundTripper := test.MockRoundTripper(func(r *http.Request) *http.Response {
		cache.Clear()
		defer cache.Clear()
		if r.Host == "a.example.com" {
			logr.Infof("Mocking remote request to %s", r.URL)
			response, err := remoteRouter.Test(r)
			if err != nil {
				panic("mocked request should not fail")
			}
			return response
		} else {
			panic("should only mock the remote endpoint")
		}
	})
	client.InjectHTTPClient(&http.Client{Transport: mockRoundTripper})

	// We use the same endpoint answers in both the local and remote instances
	expectedBody := `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`
	expectedBodyWithRemote := `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"remote-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`

	type Scenario struct {
		Name         string
		Path         string
		ExpectedCode int
		ExpectedBody string
	}
	scenarios := []Scenario{
		{
			Name:         "remote-query-default",
			Path:         "/api/v1/endpoints/statuses",
			ExpectedCode: http.StatusOK,
			ExpectedBody: expectedBodyWithRemote,
		},
		{
			Name:         "remote-query-true",
			Path:         "/api/v1/endpoints/statuses?remote=true",
			ExpectedCode: http.StatusOK,
			ExpectedBody: expectedBodyWithRemote,
		},
		{
			Name:         "remote-query-false",
			Path:         "/api/v1/endpoints/statuses?remote=false",
			ExpectedCode: http.StatusOK,
			ExpectedBody: expectedBody,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			falseVal := false
			cfg.Remote = &remote.Config{
				Instances: []*remote.Instance{&remote.Instance{
					EndpointPrefix: "remote-",
					URL:            apiAURL,
					IncludeRemote:  &falseVal,
				}},
				ClientConfig: client.GetDefaultConfig(),
			}
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			response, err := router.Test(request)
			if err != nil {
				t.Error("Request failed or timed out", err)
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

// Here we test that the `include-remote` setting is respected
// when building remote API queries. This does not check that an actual
// remote instance behaves properly.
func TestEndpointStatusesRespectsIncludeRemoteConfig(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	firstResult := &testSuccessfulResult
	secondResult := &testUnsuccessfulResult
	store.Get().InsertEndpointResult(&testEndpoint, firstResult)
	store.Get().InsertEndpointResult(&testEndpoint, secondResult)
	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
	firstResult.Timestamp = time.Time{}
	secondResult.Timestamp = time.Time{}

	instanceAcfg := &config.Config{
		// Used to disambiguate instances
		Metrics: true,
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	}
	instanceAapi := New(instanceAcfg)
	instanceArouter := instanceAapi.Router()

	expectedBodyB := `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`
	expectedBodyBWithRemoteFromA := `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"from-a-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`

	mockRoundTripper := test.MockRoundTripper(func(r *http.Request) *http.Response {
		cache.Clear()
		defer cache.Clear()
		if r.Host == "b.example.com" {
			logr.Infof("Mocking remote instance B request to %s", r.URL)
			if r.URL.Query().Get("remote") == "true" {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(expectedBodyBWithRemoteFromA))}
			} else {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(expectedBodyB))}
			}
		} else {
			panic("should only mock the remote endpoint")
		}
	})
	client.InjectHTTPClient(&http.Client{Transport: mockRoundTripper})

	// We use the same endpoint answers in both the local and remote instances
	expectedBodyAWithRemoteFromB := `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"from-b-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`
	expectedBodyAWithRemoteFromBAndA := `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"from-b-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"from-b-from-a-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`

	type Scenario struct {
		Name          string
		Path          string
		IncludeRemote bool
		ExpectedCode  int
		ExpectedBody  string
	}
	scenarios := []Scenario{
		{
			Name:          "remote-include-remote-true",
			Path:          "/api/v1/endpoints/statuses",
			IncludeRemote: true,
			ExpectedCode:  http.StatusOK,
			ExpectedBody:  expectedBodyAWithRemoteFromBAndA,
		},
		{
			Name:          "remote-include-remote-false",
			Path:          "/api/v1/endpoints/statuses",
			IncludeRemote: false,
			ExpectedCode:  http.StatusOK,
			ExpectedBody:  expectedBodyAWithRemoteFromB,
		},
	}

	for _, scenario := range scenarios {
		// The cache does not account for config changes for instance.IncludeRemote
		// so the second scenario always fails if we don't clear the cache because the API
		// cache defaults to 10s.
		cache.Clear()
		t.Run(scenario.Name, func(t *testing.T) {
			logr.Infof("Starting scenario %s", scenario.Name)
			instanceAcfg.Remote = &remote.Config{
				Instances: []*remote.Instance{&remote.Instance{
					EndpointPrefix: "from-b-",
					URL:            apiBURL,
					IncludeRemote:  &scenario.IncludeRemote,
				}},
				ClientConfig: client.GetDefaultConfig(),
			}
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			response, err := instanceArouter.Test(request)
			if err != nil {
				t.Error("Request failed or timed out", err)
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

// Here we test that in remote instance config, `include-remote` setting
// is respected. This test will fail if TestEndpointStatusesRespectsIncludeRemoteConfig
// fails because this is an integration test, not a unit test.
func TestEndpointStatusesRespectsIncludeRemoteIntegration(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	firstResult := &testSuccessfulResult
	secondResult := &testUnsuccessfulResult
	store.Get().InsertEndpointResult(&testEndpoint, firstResult)
	store.Get().InsertEndpointResult(&testEndpoint, secondResult)
	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
	firstResult.Timestamp = time.Time{}
	secondResult.Timestamp = time.Time{}

	instanceAcfg := &config.Config{
		// Used to disambiguate instances
		Metrics: true,
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
	}
	instanceAapi := New(instanceAcfg)
	instanceArouter := instanceAapi.Router()

	falseVal := false
	instanceBcfg := &config.Config{
		// Used to disambiguate instances
		Metrics: true,
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			Instances: []*remote.Instance{&remote.Instance{
				EndpointPrefix: "from-a-",
				URL:            apiAURL,
				IncludeRemote:  &falseVal,
			}},
			ClientConfig: client.GetDefaultConfig(),
		},
	}
	instanceBapi := New(instanceBcfg)
	instanceBrouter := instanceBapi.Router()

	mockRoundTripper := test.MockRoundTripper(func(r *http.Request) *http.Response {
		cache.Clear()
		defer cache.Clear()
		if r.Host == "a.example.com" {
			logr.Infof("Mocking remote instance A request to %s", r.URL)
			response, err := instanceArouter.Test(r)
			if err != nil {
				panic("mocked request should not fail")
			}
			return response
		} else if r.Host == "b.example.com" {
			logr.Infof("Mocking remote instance B request to %s", r.URL)
			response, err := instanceBrouter.Test(r)
			if err != nil {
				panic("mocked request should not fail")
			}
			return response
		} else {
			panic("should only mock the remote endpoint")
		}
	})
	client.InjectHTTPClient(&http.Client{Transport: mockRoundTripper})

	// We use the same endpoint answers in both the local and remote instances
	expectedBodyWithRemoteFromB := `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"from-b-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`
	expectedBodyWithRemoteFromBAndA := `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"from-b-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"from-b-from-a-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`

	type Scenario struct {
		Name          string
		Path          string
		IncludeRemote bool
		ExpectedCode  int
		ExpectedBody  string
	}
	scenarios := []Scenario{
		{
			Name:          "remote-include-remote-true",
			Path:          "/api/v1/endpoints/statuses",
			IncludeRemote: true,
			ExpectedCode:  http.StatusOK,
			ExpectedBody:  expectedBodyWithRemoteFromBAndA,
		},
		{
			Name:          "remote-include-remote-false",
			Path:          "/api/v1/endpoints/statuses",
			IncludeRemote: false,
			ExpectedCode:  http.StatusOK,
			ExpectedBody:  expectedBodyWithRemoteFromB,
		},
	}

	for _, scenario := range scenarios {
		// The cache does not account for config changes for instance.IncludeRemote
		// so the second scenario always fails if we don't clear the cache because the API
		// cache defaults to 10s.
		cache.Clear()
		t.Run(scenario.Name, func(t *testing.T) {
			logr.Infof("Starting scenario %s", scenario.Name)
			instanceAcfg.Remote = &remote.Config{
				Instances: []*remote.Instance{&remote.Instance{
					EndpointPrefix: "from-b-",
					URL:            apiBURL,
					IncludeRemote:  &scenario.IncludeRemote,
				}},
				ClientConfig: client.GetDefaultConfig(),
			}
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			response, err := instanceArouter.Test(request)
			if err != nil {
				t.Error("Request failed or timed out", err)
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

// Here we test that remote endpoint formatting depending on `instance.include-remote`
// adds the query params properly and respects any existing param.
func TestEndpointStatusesFormatQueryParams(t *testing.T) {
	type Scenario struct {
		Name          string
		InstanceURL   string
		ExpectedURL   string
		ExpectedError bool
		IncludeRemote bool
	}
	scenarios := []Scenario{
		{
			Name:          "remote-include-remote-true-noparams",
			InstanceURL:   "https://a.example.com/api/v1/endpoints/statuses",
			ExpectedURL:   "https://a.example.com/api/v1/endpoints/statuses?remote=true",
			ExpectedError: false,
			IncludeRemote: true,
		},
		{
			Name:          "remote-include-remote-false-noparams",
			InstanceURL:   "https://a.example.com/api/v1/endpoints/statuses",
			ExpectedURL:   "https://a.example.com/api/v1/endpoints/statuses?remote=false",
			ExpectedError: false,
			IncludeRemote: false,
		},
		{
			Name:          "remote-include-remote-true-someparams",
			InstanceURL:   "https://a.example.com/api/v1/endpoints/statuses?action=foo",
			ExpectedURL:   "https://a.example.com/api/v1/endpoints/statuses?action=foo&remote=true",
			ExpectedError: false,
			IncludeRemote: true,
		},
		{
			Name:          "remote-include-remote-false-someparams",
			InstanceURL:   "https://a.example.com/api/v1/endpoints/statuses?action=foo",
			ExpectedURL:   "https://a.example.com/api/v1/endpoints/statuses?action=foo&remote=false",
			ExpectedError: false,
			IncludeRemote: false,
		},
		// Turns out i don't know how to produce a URL that fails url.Parse validation.
		// Leaivng the `ExpectedError` here in case it ever becomes handy.
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			foundURL, err := formatRemoteInstanceQueryParams(scenario.InstanceURL, scenario.IncludeRemote)
			if err != nil {
				if scenario.ExpectedError {
					return
				}

				t.Errorf("scenario %s should not error: %s", scenario.Name, err.Error())
			}
			if foundURL != scenario.ExpectedURL {
				t.Errorf("scenario %s expected:\n %s\n\ngot:\n %s", scenario.Name, scenario.ExpectedURL, foundURL)
			}
		})
	}
}

// // This test will probably fail because at no point do we set remote=false anywhere
// // so it should timeout after exhausting a lot of resources. Why does it work?
// func TestEndpointStatusesInfiniteRecursion(t *testing.T) {
// 	defer store.Get().Clear()
// 	defer cache.Clear()
// 	firstResult := &testSuccessfulResult
// 	secondResult := &testUnsuccessfulResult
// 	store.Get().InsertEndpointResult(&testEndpoint, firstResult)
// 	store.Get().InsertEndpointResult(&testEndpoint, secondResult)
// 	// Can't be bothered dealing with timezone issues on the worker that runs the automated tests
// 	firstResult.Timestamp = time.Time{}
// 	secondResult.Timestamp = time.Time{}

// 	instanceAcfg := &config.Config{
// 		// Used to disambiguate instances
// 		Metrics: true,
// 		Storage: &storage.Config{
// 			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
// 			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
// 		},
// 	}
// 	instanceAapi := New(instanceAcfg)
// 	instanceArouter := instanceAapi.Router()

// 	trueVal := true
// 	instanceBcfg := &config.Config{
// 		// Used to disambiguate instances
// 		Metrics: true,
// 		Storage: &storage.Config{
// 			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
// 			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
// 		},
// 		Remote: &remote.Config{
// 			Instances: []*remote.Instance{&remote.Instance{
// 				EndpointPrefix: "from-a-",
// 				URL:            apiAURL,
// 				IncludeRemote:  &trueVal,
// 			}},
// 			ClientConfig: client.GetDefaultConfig(),
// 		},
// 	}
// 	instanceBapi := New(instanceBcfg)
// 	instanceBrouter := instanceBapi.Router()

// 	mockRoundTripper := test.MockRoundTripper(func(r *http.Request) *http.Response {
// 		cache.Clear()
// 		defer cache.Clear()
// 		if r.Host == "a.example.com" {
// 			logr.Infof("Mocking remote instance A request to %s", r.URL)
// 			response, err := instanceArouter.Test(r)
// 			if err != nil {
// 				panic("mocked request should not fail")
// 			}
// 			return response
// 		} else if r.Host == "b.example.com" {
// 			logr.Infof("Mocking remote instance B request to %s", r.URL)
// 			response, err := instanceBrouter.Test(r)
// 			if err != nil {
// 				panic("mocked request should not fail")
// 			}
// 			return response
// 		} else {
// 			panic("should only mock the remote endpoint")
// 		}
// 	})
// 	client.InjectHTTPClient(&http.Client{Transport: mockRoundTripper})

// 	// We use the same endpoint answers in both the local and remote instances
// 	expectedBodyWithRemoteFromBAndA := `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"from-b-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]},{"name":"from-b-from-a-name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] \u003c 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] \u003c 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`

// 	type Scenario struct {
// 		Name          string
// 		Path          string
// 		IncludeRemote bool
// 		ExpectedCode  int
// 		ExpectedBody  string
// 	}
// 	scenarios := []Scenario{
// 		{
// 			Name:          "remote-include-remote-true",
// 			Path:          "/api/v1/endpoints/statuses",
// 			IncludeRemote: true,
// 			ExpectedCode:  http.StatusOK,
// 			ExpectedBody:  expectedBodyWithRemoteFromBAndA,
// 		},
// 	}

// 	for _, scenario := range scenarios {
// 		// The cache does not account for config changes for instance.IncludeRemote
// 		// so the second scenario always fails if we don't clear the cache because the API
// 		// cache defaults to 10s.
// 		cache.Clear()
// 		t.Run(scenario.Name, func(t *testing.T) {
// 			logr.Infof("Starting scenario %s", scenario.Name)
// 			instanceAcfg.Remote = &remote.Config{
// 				Instances: []*remote.Instance{&remote.Instance{
// 					EndpointPrefix: "from-b-",
// 					URL:            apiBURL,
// 					IncludeRemote:  &scenario.IncludeRemote,
// 				}},
// 				ClientConfig: client.GetDefaultConfig(),
// 			}
// 			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
// 			response, err := instanceArouter.Test(request)
// 			if err != nil {
// 				t.Error("Request failed or timed out", err)
// 			}
// 			defer response.Body.Close()
// 			if response.StatusCode != scenario.ExpectedCode {
// 				t.Errorf("%s %s should have returned %d, but returned %d instead", request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
// 			}
// 			body, err := io.ReadAll(response.Body)
// 			if err != nil {
// 				t.Error("expected err to be nil, but was", err)
// 			}
// 			if string(body) != scenario.ExpectedBody {
// 				t.Errorf("expected:\n %s\n\ngot:\n %s", scenario.ExpectedBody, string(body))
// 			}
// 		})
// 	}
// }

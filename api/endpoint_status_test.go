package api


import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"fmt"
	"encoding/json"
	"reflect"

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
		Errors:                []string{},
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

func TestEndpointStatuses(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()

	firstResult := &testSuccessfulResult
	secondResult := &testUnsuccessfulResult

	// ✅ Use watchdog to insert
	watchdog.UpdateEndpointStatuses(&testEndpoint, firstResult)
	watchdog.UpdateEndpointStatuses(&testEndpoint, secondResult)

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
		Name               string
		Path               string
		ExpectedCode       int
		ExpectedBody       string
		ExpectedTotalCount int
	}


	scenarios := []Scenario{
		{
			Name: "no-pagination",
			Path: "/api/v1/endpoints/statuses",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] < 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] < 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] < 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] < 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`,
			ExpectedTotalCount: 1,
		},
		{
			Name: "pagination-first-page",
			Path: "/api/v1/endpoints/statuses?page=1&pageSize=1",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] < 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] < 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"}]}]`,
			ExpectedTotalCount: 1,
		},
		{
			Name: "pagination-second-page-no-endpoints",
			Path: "/api/v1/endpoints/statuses?page=2&pageSize=1",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[]`,
			ExpectedTotalCount: 1,
		},
		{
			Name: "pagination-no-results",
			Path: "/api/v1/endpoints/statuses?page=5&pageSize=20",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[]`,
			ExpectedTotalCount: 1,
		},
		{
			Name: "invalid-pagination-should-fall-back-to-default",
			Path: "/api/v1/endpoints/statuses?page=INVALID&pageSize=INVALID",
			ExpectedCode: http.StatusOK,
			ExpectedBody: `[{"name":"name","group":"group","key":"group_name","results":[{"status":200,"hostname":"example.org","duration":150000000,"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] < 500","success":true},{"condition":"[CERTIFICATE_EXPIRATION] < 72h","success":true}],"success":true,"timestamp":"0001-01-01T00:00:00Z"},{"status":200,"hostname":"example.org","duration":750000000,"errors":["error-1","error-2"],"conditionResults":[{"condition":"[STATUS] == 200","success":true},{"condition":"[RESPONSE_TIME] < 500","success":false},{"condition":"[CERTIFICATE_EXPIRATION] < 72h","success":false}],"success":false,"timestamp":"0001-01-01T00:00:00Z"}]}]`,
			ExpectedTotalCount: 1,
		},
	}


	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			request := httptest.NewRequest("GET", scenario.Path, http.NoBody)
			response, err := router.Test(request)
			if err != nil {
				t.Fatalf("unexpected error executing test request: %v", err)
			}
			defer response.Body.Close()

			if response.StatusCode != scenario.ExpectedCode {
				t.Errorf("%s %s should have returned %d, but returned %d instead",
					request.Method, request.URL, scenario.ExpectedCode, response.StatusCode)
			}

			totalCountHeader := response.Header.Get("X-Total-Count")
			if totalCountHeader == "" {
				t.Errorf("%s %s should have returned X-Total-Count header, but it was missing",
					request.Method, request.URL)
			} else {
				if totalCountHeader != fmt.Sprintf("%d", scenario.ExpectedTotalCount) {
					t.Errorf("%s %s should have returned X-Total-Count=%d, but got %s",
						request.Method, request.URL, scenario.ExpectedTotalCount, totalCountHeader)
				}
			}

			// ✅ check body
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatalf("failed reading response body: %v", err)
			}

			bodyStr := string(body)

			var actual, expected any
			if err := json.Unmarshal([]byte(bodyStr), &actual); err != nil {
				t.Fatalf("failed to unmarshal actual body: %v\nbody:\n%s", err, bodyStr)
			}
			if err := json.Unmarshal([]byte(scenario.ExpectedBody), &expected); err != nil {
				t.Fatalf("failed to unmarshal expected body: %v\nexpected:\n%s", err, scenario.ExpectedBody)
			}

			if !reflect.DeepEqual(actual, expected) {
				t.Errorf("expected JSON:\n%s\n\ngot JSON:\n%s", scenario.ExpectedBody, bodyStr)
			}
		})
	}
}

package ntfy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestAlertDefaultProvider_IsValid(t *testing.T) {
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected bool
	}{
		{
			name:     "valid",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
			expected: true,
		},
		{
			name:     "no-url-should-use-default-value",
			provider: AlertProvider{Topic: "example", Priority: 1},
			expected: true,
		},
		{
			name:     "valid-with-token",
			provider: AlertProvider{Topic: "example", Priority: 1, Token: "tk_faketoken"},
			expected: true,
		},
		{
			name:     "invalid-token",
			provider: AlertProvider{Topic: "example", Priority: 1, Token: "xx_faketoken"},
			expected: false,
		},
		{
			name:     "invalid-topic",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "", Priority: 1},
			expected: false,
		},
		{
			name:     "invalid-priority-too-high",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 6},
			expected: false,
		},
		{
			name:     "invalid-priority-too-low",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: -1},
			expected: false,
		},
		{
			name:     "no-priority-should-use-default-value",
			provider: AlertProvider{URL: "https://ntfy.sh", Topic: "example"},
			expected: true,
		},
		{
			name:     "invalid-override-token",
			provider: AlertProvider{Topic: "example", Overrides: []Override{Override{Group: "g", Token: "xx_faketoken"}}},
			expected: false,
		},
		{
			name:     "invalid-override-priority",
			provider: AlertProvider{Topic: "example", Overrides: []Override{Override{Group: "g", Priority: 8}}},
			expected: false,
		},
		{
			name:     "no-override-group-name",
			provider: AlertProvider{Topic: "example", Overrides: []Override{Override{}}},
			expected: false,
		},
		{
			name:     "duplicate-override-group-names",
			provider: AlertProvider{Topic: "example", Overrides: []Override{Override{Group: "g"}, Override{Group: "g"}}},
			expected: false,
		},
		{
			name:     "valid-override",
			provider: AlertProvider{Topic: "example", Overrides: []Override{Override{Group: "g1", Priority: 4, Click: "https://example.com"}, Override{Group: "g2", Topic: "Example", Token: "tk_faketoken"}}},
			expected: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if scenario.provider.IsValid() != scenario.expected {
				t.Errorf("expected %t, got %t", scenario.expected, scenario.provider.IsValid())
			}
		})
	}
}

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Alert        alert.Alert
		Resolved     bool
		Override     *Override
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Override:     nil,
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1}`,
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 2},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			Override:     nil,
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\n🟢 [CONNECTED] == true\n🟢 [STATUS] == 200","tags":["white_check_mark"],"priority":2}`,
		},
		{
			Name:         "triggered-email",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Override:     nil,
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1,"email":"test@example.com","click":"example.com"}`,
		},
		{
			Name:         "resolved-email",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 2, Email: "test@example.com", Click: "example.com"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			Override:     nil,
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\n🟢 [CONNECTED] == true\n🟢 [STATUS] == 200","tags":["white_check_mark"],"priority":2,"email":"test@example.com","click":"example.com"}`,
		},
		{
			Name:         "override",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 5, Email: "test@example.com", Click: "example.com"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Override:     &Override{Group: "g", Topic: "override-topic", Priority: 4, Email: "override@test.com", Click: "test.com"},
			ExpectedBody: `{"topic":"override-topic","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":4,"email":"override@test.com","click":"test.com"}`,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
				scenario.Override,
			)
			if string(body) != scenario.ExpectedBody {
				t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
			}
			out := make(map[string]interface{})
			if err := json.Unmarshal(body, &out); err != nil {
				t.Error("expected body to be valid JSON, got error:", err.Error())
			}
		})
	}
}

func TestAlertProvider_Send(t *testing.T) {
	description := "description-1"
	scenarios := []struct {
		Name            string
		Provider        AlertProvider
		Alert           alert.Alert
		Resolved        bool
		Group           string
		ExpectedBody    string
		ExpectedHeaders map[string]string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com"},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Group:        "",
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1,"email":"test@example.com","click":"example.com"}`,
			ExpectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name:         "token",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", Token: "tk_mytoken"},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Group:        "",
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1,"email":"test@example.com","click":"example.com"}`,
			ExpectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer tk_mytoken",
			},
		},
		{
			Name:         "no firebase",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", DisableFirebase: true},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Group:        "",
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1,"email":"test@example.com","click":"example.com"}`,
			ExpectedHeaders: map[string]string{
				"Content-Type": "application/json",
				"Firebase":     "no",
			},
		},
		{
			Name:         "no cache",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", DisableCache: true},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Group:        "",
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1,"email":"test@example.com","click":"example.com"}`,
			ExpectedHeaders: map[string]string{
				"Content-Type": "application/json",
				"Cache":        "no",
			},
		},
		{
			Name:         "neither firebase & cache",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", DisableFirebase: true, DisableCache: true},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Group:        "",
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1,"email":"test@example.com","click":"example.com"}`,
			ExpectedHeaders: map[string]string{
				"Content-Type": "application/json",
				"Firebase":     "no",
				"Cache":        "no",
			},
		},
		{
			Name:         "overrides",
			Provider:     AlertProvider{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", Token: "tk_mytoken", Overrides: []Override{Override{Group: "other-group", URL: "https://example.com", Token: "tk_othertoken"}, Override{Group: "test-group", Token: "tk_test_token"}}},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Group:        "test-group",
			ExpectedBody: `{"topic":"example","title":"Gatus: test-group/endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1,"email":"test@example.com","click":"example.com"}`,
			ExpectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer tk_test_token",
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			// Start a local HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				// Test request parameters
				for header, value := range scenario.ExpectedHeaders {
					if value != req.Header.Get(header) {
						t.Errorf("expected: %s, got: %s", value, req.Header.Get(header))
					}
				}
				body, _ := io.ReadAll(req.Body)
				if string(body) != scenario.ExpectedBody {
					t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
				}
				// Send response to be tested
				rw.Write([]byte(`OK`))
			}))
			// Close the server when test finishes
			defer server.Close()

			scenario.Provider.URL = server.URL
			err := scenario.Provider.Send(
				&endpoint.Endpoint{Name: "endpoint-name", Group: scenario.Group},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			if err != nil {
				t.Error("Encountered an error on Send: ", err)
			}

		})
	}

}

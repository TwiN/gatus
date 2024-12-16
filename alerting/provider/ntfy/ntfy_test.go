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

func TestAlertProvider_Validate(t *testing.T) {
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected bool
	}{
		{
			name:     "valid",
			provider: AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1}},
			expected: true,
		},
		{
			name:     "no-url-should-use-default-value",
			provider: AlertProvider{DefaultConfig: Config{Topic: "example", Priority: 1}},
			expected: true,
		},
		{
			name:     "valid-with-token",
			provider: AlertProvider{DefaultConfig: Config{Topic: "example", Priority: 1, Token: "tk_faketoken"}},
			expected: true,
		},
		{
			name:     "invalid-token",
			provider: AlertProvider{DefaultConfig: Config{Topic: "example", Priority: 1, Token: "xx_faketoken"}},
			expected: false,
		},
		{
			name:     "invalid-topic",
			provider: AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "", Priority: 1}},
			expected: false,
		},
		{
			name:     "invalid-priority-too-high",
			provider: AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 6}},
			expected: false,
		},
		{
			name:     "invalid-priority-too-low",
			provider: AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: -1}},
			expected: false,
		},
		{
			name:     "no-priority-should-use-default-value",
			provider: AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example"}},
			expected: true,
		},
		{
			name:     "invalid-override-token",
			provider: AlertProvider{DefaultConfig: Config{Topic: "example"}, Overrides: []Override{{Group: "g", Config: Config{Token: "xx_faketoken"}}}},
			expected: false,
		},
		{
			name:     "invalid-override-priority",
			provider: AlertProvider{DefaultConfig: Config{Topic: "example"}, Overrides: []Override{{Group: "g", Config: Config{Priority: 8}}}},
			expected: false,
		},
		{
			name:     "no-override-group-name",
			provider: AlertProvider{DefaultConfig: Config{Topic: "example"}, Overrides: []Override{{}}},
			expected: false,
		},
		{
			name:     "duplicate-override-group-names",
			provider: AlertProvider{DefaultConfig: Config{Topic: "example"}, Overrides: []Override{{Group: "g"}, {Group: "g"}}},
			expected: false,
		},
		{
			name:     "valid-override",
			provider: AlertProvider{DefaultConfig: Config{Topic: "example"}, Overrides: []Override{{Group: "g1", Config: Config{Priority: 4, Click: "https://example.com"}}, {Group: "g2", Config: Config{Topic: "Example", Token: "tk_faketoken"}}}},
			expected: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.provider.Validate()
			if scenario.expected && err != nil {
				t.Error("expected no error, got", err.Error())
			}
			if !scenario.expected && err == nil {
				t.Error("expected error, got none")
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
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1}`,
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 2}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\n🟢 [CONNECTED] == true\n🟢 [STATUS] == 200","tags":["white_check_mark"],"priority":2}`,
		},
		{
			Name:         "triggered-email",
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":1,"email":"test@example.com","click":"example.com"}`,
		},
		{
			Name:         "resolved-email",
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 2, Email: "test@example.com", Click: "example.com"}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: `{"topic":"example","title":"Gatus: endpoint-name","message":"An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\n🟢 [CONNECTED] == true\n🟢 [STATUS] == 200","tags":["white_check_mark"],"priority":2,"email":"test@example.com","click":"example.com"}`,
		},
		{
			Name:         "group-override",
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 5, Email: "test@example.com", Click: "example.com"}, Overrides: []Override{{Group: "g", Config: Config{Topic: "group-topic", Priority: 4, Email: "override@test.com", Click: "test.com"}}}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: `{"topic":"group-topic","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":4,"email":"override@test.com","click":"test.com"}`,
		},
		{
			Name:         "alert-override",
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 5, Email: "test@example.com", Click: "example.com"}, Overrides: []Override{{Group: "g", Config: Config{Topic: "group-topic", Priority: 4, Email: "override@test.com", Click: "test.com"}}}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3, ProviderOverride: map[string]any{"topic": "alert-topic"}},
			Resolved:     false,
			ExpectedBody: `{"topic":"alert-topic","title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","tags":["rotating_light"],"priority":4,"email":"override@test.com","click":"test.com"}`,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg, err := scenario.Provider.GetConfig("g", &scenario.Alert)
			if err != nil {
				t.Error("expected no error, got", err.Error())
			}
			body := scenario.Provider.buildRequestBody(
				cfg,
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
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
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com"}},
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
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", Token: "tk_mytoken"}},
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
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", DisableFirebase: true}},
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
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", DisableCache: true}},
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
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", DisableFirebase: true, DisableCache: true}},
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
			Provider:     AlertProvider{DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1, Email: "test@example.com", Click: "example.com", Token: "tk_mytoken"}, Overrides: []Override{Override{Group: "other-group", Config: Config{URL: "https://example.com", Token: "tk_othertoken"}}, Override{Group: "test-group", Config: Config{Token: "tk_test_token"}}}},
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

			scenario.Provider.DefaultConfig.URL = server.URL
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

func TestAlertProvider_GetConfig(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		InputAlert     alert.Alert
		ExpectedOutput Config
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{URL: "https://group-example.com", Topic: "group-topic", Priority: 2},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{URL: "https://group-example.com", Topic: "group-topic", Priority: 2},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://group-example.com", Topic: "group-topic", Priority: 2},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{URL: "https://group-example.com", Topic: "group-topic", Priority: 2},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"url": "http://alert-example.com", "topic": "alert-topic", "priority": 3}},
			ExpectedOutput: Config{URL: "http://alert-example.com", Topic: "alert-topic", Priority: 3},
		},
		{
			Name: "provider-with-partial-overrides",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://ntfy.sh", Topic: "example", Priority: 1},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{Topic: "group-topic"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"priority": 3}},
			ExpectedOutput: Config{URL: "https://ntfy.sh", Topic: "group-topic", Priority: 3},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.URL != scenario.ExpectedOutput.URL {
				t.Errorf("expected url %s, got %s", scenario.ExpectedOutput.URL, got.URL)
			}
			if got.Topic != scenario.ExpectedOutput.Topic {
				t.Errorf("expected topic %s, got %s", scenario.ExpectedOutput.Topic, got.Topic)
			}
			if got.Priority != scenario.ExpectedOutput.Priority {
				t.Errorf("expected priority %d, got %d", scenario.ExpectedOutput.Priority, got.Priority)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

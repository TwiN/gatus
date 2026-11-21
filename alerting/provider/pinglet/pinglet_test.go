package pinglet

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
			provider: AlertProvider{DefaultConfig: Config{URL: "https://pinglet.dev", APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "urgent"}},
			expected: true,
		},
		{
			name:     "no-url-should-use-default-value",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys"}},
			expected: true,
		},
		{
			name:     "no-priority-should-use-default-value",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys"}},
			expected: true,
		},
		{
			name:     "invalid-priority",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "critical"}},
			expected: false,
		},
		{
			name:     "missing-api-key",
			provider: AlertProvider{DefaultConfig: Config{Namespace: "acme", Topic: "deploys"}},
			expected: false,
		},
		{
			name:     "missing-namespace",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Topic: "deploys"}},
			expected: false,
		},
		{
			name:     "missing-topic",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme"}},
			expected: false,
		},
		{
			name:     "invalid-override-priority",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys"}, Overrides: []Override{{Group: "g", Config: Config{Priority: "loud"}}}},
			expected: false,
		},
		{
			name:     "no-override-group-name",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys"}, Overrides: []Override{{}}},
			expected: false,
		},
		{
			name:     "duplicate-override-group-names",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys"}, Overrides: []Override{{Group: "g"}, {Group: "g"}}},
			expected: false,
		},
		{
			name:     "valid-override",
			provider: AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys"}, Overrides: []Override{{Group: "g1", Config: Config{Priority: "silent"}}, {Group: "g2", Config: Config{Topic: "other", APIKey: "pinglet_other"}}}},
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
			Provider:     AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "urgent"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: `{"title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","priority":"urgent"}`,
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "normal"}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: `{"title":"Gatus: endpoint-name","message":"An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\n🟢 [CONNECTED] == true\n🟢 [STATUS] == 200","priority":"normal"}`,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg, err := scenario.Provider.GetConfig("", &scenario.Alert)
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
		ExpectedPath    string
		ExpectedHeaders map[string]string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "urgent"}},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Group:        "",
			ExpectedBody: `{"title":"Gatus: endpoint-name","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1\n🔴 [CONNECTED] == true\n🔴 [STATUS] == 200","priority":"urgent"}`,
			ExpectedPath: "/acme/deploys",
			ExpectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer pinglet_key",
			},
		},
		{
			Name:         "resolved-with-override",
			Provider:     AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "urgent"}, Overrides: []Override{{Group: "test-group", Config: Config{Topic: "group-topic", Priority: "normal"}}}},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			Group:        "test-group",
			ExpectedBody: `{"title":"Gatus: test-group/endpoint-name","message":"An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-1\n🟢 [CONNECTED] == true\n🟢 [STATUS] == 200","priority":"normal"}`,
			ExpectedPath: "/acme/group-topic",
			ExpectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer pinglet_key",
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
						t.Errorf("expected header %s to be %s, got %s", header, value, req.Header.Get(header))
					}
				}
				if req.URL.Path != scenario.ExpectedPath {
					t.Errorf("expected path %s, got %s", scenario.ExpectedPath, req.URL.Path)
				}
				body, _ := io.ReadAll(req.Body)
				if string(body) != scenario.ExpectedBody {
					t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
				}
				rw.Write([]byte(`{}`))
			}))
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

func TestAlertProvider_SendReturnsErrorOnFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(`{"error":"someone turned off the internet"}`))
	}))
	defer server.Close()
	description := "description-1"
	provider := AlertProvider{DefaultConfig: Config{URL: server.URL, APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys"}}
	err := provider.Send(
		&endpoint.Endpoint{Name: "endpoint-name"},
		&alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
		&endpoint.Result{ConditionResults: []*endpoint.ConditionResult{{Condition: "[STATUS] == 200", Success: false}}},
		false,
	)
	if err == nil {
		t.Error("expected an error, got none")
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
				DefaultConfig: Config{URL: "https://pinglet.dev", APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "normal"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://pinglet.dev", APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "normal"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://pinglet.dev", APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "normal"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{Topic: "group-topic", Priority: "urgent"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "https://pinglet.dev", APIKey: "pinglet_key", Namespace: "acme", Topic: "group-topic", Priority: "urgent"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "https://pinglet.dev", APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys", Priority: "normal"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{Topic: "group-topic", Priority: "urgent"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"topic": "alert-topic", "priority": "silent"}},
			ExpectedOutput: Config{URL: "https://pinglet.dev", APIKey: "pinglet_key", Namespace: "acme", Topic: "alert-topic", Priority: "silent"},
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
			if got.APIKey != scenario.ExpectedOutput.APIKey {
				t.Errorf("expected api-key %s, got %s", scenario.ExpectedOutput.APIKey, got.APIKey)
			}
			if got.Namespace != scenario.ExpectedOutput.Namespace {
				t.Errorf("expected namespace %s, got %s", scenario.ExpectedOutput.Namespace, got.Namespace)
			}
			if got.Topic != scenario.ExpectedOutput.Topic {
				t.Errorf("expected topic %s, got %s", scenario.ExpectedOutput.Topic, got.Topic)
			}
			if got.Priority != scenario.ExpectedOutput.Priority {
				t.Errorf("expected priority %s, got %s", scenario.ExpectedOutput.Priority, got.Priority)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

func TestConfig_URLTrailingSlash(t *testing.T) {
	provider := AlertProvider{DefaultConfig: Config{APIKey: "pinglet_key", Namespace: "acme", Topic: "deploys"}}
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		capturedPath = req.URL.Path
		rw.Write([]byte(`{}`))
	}))
	defer server.Close()
	provider.DefaultConfig.URL = server.URL + "/"
	if err := provider.Send(
		&endpoint.Endpoint{Name: "endpoint-name"},
		&alert.Alert{SuccessThreshold: 1, FailureThreshold: 1},
		&endpoint.Result{ConditionResults: []*endpoint.ConditionResult{{Condition: "[STATUS] == 200", Success: false}}},
		false,
	); err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if strings.Contains(capturedPath, "//") {
		t.Errorf("expected no double slash in path, got %s", capturedPath)
	}
	if capturedPath != "/acme/deploys" {
		t.Errorf("expected path /acme/deploys, got %s", capturedPath)
	}
}

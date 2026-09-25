package flowtriq

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
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus"}},
			expected: true,
		},
		{
			name:     "valid-with-api-key",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus", APIKey: "test-key"}},
			expected: true,
		},
		{
			name:     "missing-webhook-url",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: ""}},
			expected: false,
		},
		{
			name:     "no-override-group-name",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus"}, Overrides: []Override{{}}},
			expected: false,
		},
		{
			name:     "duplicate-override-group-names",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus"}, Overrides: []Override{{Group: "g"}, {Group: "g"}}},
			expected: false,
		},
		{
			name:     "valid-override",
			provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus"}, Overrides: []Override{{Group: "g1", Config: Config{WebhookURL: "https://other.flowtriq.com/webhook/gatus"}}}},
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
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: `{"endpoint_name":"endpoint-name","status":"TRIGGERED","description":"description-1","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1","conditions":[{"condition":"[CONNECTED] == true","success":false},{"condition":"[STATUS] == 200","success":false}]}`,
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus"}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: `{"endpoint_name":"endpoint-name","status":"RESOLVED","description":"description-2","message":"An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-2","conditions":[{"condition":"[CONNECTED] == true","success":true},{"condition":"[STATUS] == 200","success":true}]}`,
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
		ExpectedHeaders map[string]string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus"}},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Group:        "",
			ExpectedBody: `{"endpoint_name":"endpoint-name","status":"TRIGGERED","description":"description-1","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1","conditions":[{"condition":"[CONNECTED] == true","success":false},{"condition":"[STATUS] == 200","success":false}]}`,
			ExpectedHeaders: map[string]string{
				"Content-Type": "application/json",
			},
		},
		{
			Name:         "triggered-with-api-key",
			Provider:     AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus", APIKey: "my-api-key"}},
			Alert:        alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			Group:        "",
			ExpectedBody: `{"endpoint_name":"endpoint-name","status":"TRIGGERED","description":"description-1","message":"An alert has been triggered due to having failed 3 time(s) in a row with the following description: description-1","conditions":[{"condition":"[CONNECTED] == true","success":false},{"condition":"[STATUS] == 200","success":false}]}`,
			ExpectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer my-api-key",
			},
		},
		{
			Name:     "resolved-with-group",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus"}, Overrides: []Override{{Group: "test-group", Config: Config{APIKey: "override-key"}}}},
			Alert:    alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			Group:    "test-group",
			ExpectedBody: `{"endpoint_name":"test-group/endpoint-name","group":"test-group","status":"RESOLVED","description":"description-1","message":"An alert has been resolved after passing successfully 5 time(s) in a row with the following description: description-1","conditions":[{"condition":"[CONNECTED] == true","success":true},{"condition":"[STATUS] == 200","success":true}]}`,
			ExpectedHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer override-key",
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				for header, value := range scenario.ExpectedHeaders {
					if value != req.Header.Get(header) {
						t.Errorf("expected: %s, got: %s", value, req.Header.Get(header))
					}
				}
				body, _ := io.ReadAll(req.Body)
				if string(body) != scenario.ExpectedBody {
					t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
				}
				rw.Write([]byte(`OK`))
			}))
			defer server.Close()

			scenario.Provider.DefaultConfig.WebhookURL = server.URL
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
				DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus", APIKey: "default-key"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus", APIKey: "default-key"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus", APIKey: "default-key"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "https://other.flowtriq.com/webhook/gatus", APIKey: "group-key"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "https://other.flowtriq.com/webhook/gatus", APIKey: "group-key"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "https://app.flowtriq.com/webhook/gatus", APIKey: "default-key"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "https://group.flowtriq.com/webhook/gatus"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"webhook-url": "https://alert.flowtriq.com/webhook/gatus"}},
			ExpectedOutput: Config{WebhookURL: "https://alert.flowtriq.com/webhook/gatus", APIKey: "default-key"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.WebhookURL != scenario.ExpectedOutput.WebhookURL {
				t.Errorf("expected webhook-url %s, got %s", scenario.ExpectedOutput.WebhookURL, got.WebhookURL)
			}
			if got.APIKey != scenario.ExpectedOutput.APIKey {
				t.Errorf("expected api-key %s, got %s", scenario.ExpectedOutput.APIKey, got.APIKey)
			}
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

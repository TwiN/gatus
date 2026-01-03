package clickup

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProviderNoListID := AlertProvider{DefaultConfig: Config{ListID: "", Token: "test-token"}}
	if err := invalidProviderNoListID.Validate(); err == nil {
		t.Error("provider shouldn't have been valid without list-id")
	}
	invalidProviderNoToken := AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: ""}}
	if err := invalidProviderNoToken.Validate(); err == nil {
		t.Error("provider shouldn't have been valid without token")
	}
	invalidProviderBadPriority := AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token", Priority: "invalid"}}
	if err := invalidProviderBadPriority.Validate(); err == nil {
		t.Error("provider shouldn't have been valid with invalid priority")
	}
	validProvider := AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
	if validProvider.DefaultConfig.Priority != "normal" {
		t.Errorf("expected default priority to be 'normal', got '%s'", validProvider.DefaultConfig.Priority)
	}
	validProviderWithAPIURL := AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token", APIURL: "https://api.clickup.com/api/v2"}}
	if err := validProviderWithAPIURL.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
	validProviderWithPriority := AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token", Priority: "urgent"}}
	if err := validProviderWithPriority.Validate(); err != nil {
		t.Error("provider should've been valid with priority 'urgent'")
	}
	validProviderWithNone := AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token", Priority: "none"}}
	if err := validProviderWithNone.Validate(); err != nil {
		t.Error("provider should've been valid with priority 'none'")
	}
}

func TestAlertProvider_ValidateSetsDefaultAPIURL(t *testing.T) {
	provider := AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"}}
	if err := provider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
	if provider.DefaultConfig.APIURL != "https://api.clickup.com/api/v2" {
		t.Errorf("expected APIURL to be set to default, got %s", provider.DefaultConfig.APIURL)
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Endpoint         endpoint.Endpoint
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:     "triggered",
			Provider: AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"}},
			Endpoint: endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Method == "POST" && r.URL.Path == "/api/v2/list/test-list-id/task" {
					return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
				}
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"}},
			Endpoint: endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"}},
			Endpoint: endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Method == "GET" {
					// Mock fetch tasks response
					tasksResponse := map[string]interface{}{
						"tasks": []map[string]interface{}{
							{
								"id":   "task-123",
								"name": "Health Check: endpoint-group:endpoint-name",
							},
						},
					}
					body, _ := json.Marshal(tasksResponse)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(body)),
					}
				}
				if r.Method == "PUT" {
					// Mock update task status response
					return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
				}
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-no-matching-tasks",
			Provider: AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"}},
			Endpoint: endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Method == "GET" {
					// Mock fetch tasks response with no matching tasks
					tasksResponse := map[string]interface{}{
						"tasks": []map[string]interface{}{},
					}
					body, _ := json.Marshal(tasksResponse)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(body)),
					}
				}
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved-error-fetching-tasks",
			Provider: AlertProvider{DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"}},
			Endpoint: endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			err := scenario.Provider.Send(
				&scenario.Endpoint,
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
					Errors: []string{"error1", "error2"},
				},
				scenario.Resolved,
			)
			if scenario.ExpectedError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.ExpectedError && err != nil {
				t.Error("expected no error, got", err.Error())
			}
		})
	}
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	if (&AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
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
			Name: "provider-no-override-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{ListID: "test-list-id", Token: "test-token", Priority: "normal"},
		},
		{
			Name: "provider-with-alert-override-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"},
			},
			InputGroup: "",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{
				"list-id": "override-list-id",
				"token":   "override-token",
			}},
			ExpectedOutput: Config{ListID: "override-list-id", Token: "override-token", Priority: "normal"},
		},
		{
			Name: "provider-with-partial-alert-override-should-merge",
			Provider: AlertProvider{
				DefaultConfig: Config{ListID: "test-list-id", Token: "test-token", Status: "in progress"},
			},
			InputGroup: "",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{
				"status": "closed",
			}},
			ExpectedOutput: Config{ListID: "test-list-id", Token: "test-token", Status: "closed", Priority: "normal"},
		},
		{
			Name: "provider-with-assignees-override",
			Provider: AlertProvider{
				DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"},
			},
			InputGroup: "",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{
				"assignees": []string{"user1", "user2"},
			}},
			ExpectedOutput: Config{ListID: "test-list-id", Token: "test-token", Assignees: []string{"user1", "user2"}, Priority: "normal"},
		},
		{
			Name: "provider-with-priority-override",
			Provider: AlertProvider{
				DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"},
			},
			InputGroup: "",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{
				"priority": "urgent",
			}},
			ExpectedOutput: Config{ListID: "test-list-id", Token: "test-token", Priority: "urgent"},
		},
		{
			Name: "provider-with-none-priority",
			Provider: AlertProvider{
				DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"},
			},
			InputGroup: "",
			InputAlert: alert.Alert{ProviderOverride: map[string]any{
				"priority": "none",
			}},
			ExpectedOutput: Config{ListID: "test-list-id", Token: "test-token", Priority: "none"},
		},
		{
			Name: "provider-with-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{ListID: "test-list-id", Token: "test-token"},
				Overrides: []Override{
					{Group: "core", Config: Config{ListID: "core-list-id", Priority: "urgent"}},
				},
			},
			InputGroup:     "core",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{ListID: "core-list-id", Token: "test-token", Priority: "urgent"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.ListID != scenario.ExpectedOutput.ListID {
				t.Errorf("expected ListID to be %s, got %s", scenario.ExpectedOutput.ListID, got.ListID)
			}
			if got.Token != scenario.ExpectedOutput.Token {
				t.Errorf("expected Token to be %s, got %s", scenario.ExpectedOutput.Token, got.Token)
			}
			if got.Status != scenario.ExpectedOutput.Status {
				t.Errorf("expected Status to be %s, got %s", scenario.ExpectedOutput.Status, got.Status)
			}
			if got.Priority != scenario.ExpectedOutput.Priority {
				t.Errorf("expected Priority to be %s, got %s", scenario.ExpectedOutput.Priority, got.Priority)
			}
			if len(got.Assignees) != len(scenario.ExpectedOutput.Assignees) {
				t.Errorf("expected Assignees length to be %d, got %d", len(scenario.ExpectedOutput.Assignees), len(got.Assignees))
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

package jira

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
	invalidProviderNoBaseURL := AlertProvider{DefaultConfig: Config{Username: "user@example.com", Token: "token", ProjectKey: "OPS"}}
	if err := invalidProviderNoBaseURL.Validate(); err == nil {
		t.Error("provider shouldn't have been valid without base-url")
	}
	invalidProviderNoUsername := AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Token: "token", ProjectKey: "OPS"}}
	if err := invalidProviderNoUsername.Validate(); err == nil {
		t.Error("provider shouldn't have been valid without username")
	}
	invalidProviderNoToken := AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", ProjectKey: "OPS"}}
	if err := invalidProviderNoToken.Validate(); err == nil {
		t.Error("provider shouldn't have been valid without token")
	}
	invalidProviderNoProjectKey := AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token"}}
	if err := invalidProviderNoProjectKey.Validate(); err == nil {
		t.Error("provider shouldn't have been valid without project-key")
	}
	validProvider := AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
	if validProvider.DefaultConfig.IssueType != defaultIssueType {
		t.Errorf("expected default issue-type to be %q, got %q", defaultIssueType, validProvider.DefaultConfig.IssueType)
	}
	if validProvider.DefaultConfig.ResolveTransition != defaultResolveTransition {
		t.Errorf("expected default resolve-transition to be %q, got %q", defaultResolveTransition, validProvider.DefaultConfig.ResolveTransition)
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Resolved         bool
		Alert            alert.Alert
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:     "triggered",
			Provider: AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS"}},
			Resolved: false,
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Method == http.MethodPost && r.URL.Path == "/rest/api/2/issue" {
					return &http.Response{StatusCode: http.StatusCreated, Body: http.NoBody}
				}
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS"}},
			Resolved: false,
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS"}},
			Resolved: true,
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/rest/api/2/search":
					body, _ := json.Marshal(map[string]any{
						"issues": []map[string]any{
							{"key": "OPS-1", "fields": map[string]any{"summary": "alert(gatus): endpoint-name"}},
						},
					})
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body))}
				case r.Method == http.MethodGet && r.URL.Path == "/rest/api/2/issue/OPS-1/transitions":
					body, _ := json.Marshal(map[string]any{
						"transitions": []map[string]any{{"id": "31", "name": "Done"}},
					})
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body))}
				case r.Method == http.MethodPost && r.URL.Path == "/rest/api/2/issue/OPS-1/transitions":
					return &http.Response{StatusCode: http.StatusNoContent, Body: http.NoBody}
				}
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-no-matching-issue",
			Provider: AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS"}},
			Resolved: true,
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				if r.Method == http.MethodGet && r.URL.Path == "/rest/api/2/search" {
					body, _ := json.Marshal(map[string]any{"issues": []map[string]any{}})
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body))}
				}
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error-searching",
			Provider: AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS"}},
			Resolved: true,
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved-missing-transition",
			Provider: AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS", ResolveTransition: "Closed"}},
			Resolved: true,
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/rest/api/2/search":
					body, _ := json.Marshal(map[string]any{
						"issues": []map[string]any{
							{"key": "OPS-1", "fields": map[string]any{"summary": "alert(gatus): endpoint-name"}},
						},
					})
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body))}
				case r.Method == http.MethodGet && r.URL.Path == "/rest/api/2/issue/OPS-1/transitions":
					body, _ := json.Marshal(map[string]any{
						"transitions": []map[string]any{{"id": "31", "name": "Done"}},
					})
					return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(body))}
				}
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			err := scenario.Provider.Send(
				&endpoint.Endpoint{Name: "endpoint-name"},
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
		InputAlert     alert.Alert
		ExpectedOutput Config
	}{
		{
			Name:           "provider-no-override-should-default",
			Provider:       AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS"}},
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS", IssueType: "Task", ResolveTransition: "Done"},
		},
		{
			Name:     "provider-with-alert-override-should-override",
			Provider: AlertProvider{DefaultConfig: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "OPS"}},
			InputAlert: alert.Alert{ProviderOverride: map[string]any{
				"project-key": "SRE",
				"issue-type":  "Bug",
			}},
			ExpectedOutput: Config{BaseURL: "https://example.atlassian.net", Username: "user@example.com", Token: "token", ProjectKey: "SRE", IssueType: "Bug", ResolveTransition: "Done"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig("", &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.ProjectKey != scenario.ExpectedOutput.ProjectKey {
				t.Errorf("expected ProjectKey to be %s, got %s", scenario.ExpectedOutput.ProjectKey, got.ProjectKey)
			}
			if got.IssueType != scenario.ExpectedOutput.IssueType {
				t.Errorf("expected IssueType to be %s, got %s", scenario.ExpectedOutput.IssueType, got.IssueType)
			}
			if got.ResolveTransition != scenario.ExpectedOutput.ResolveTransition {
				t.Errorf("expected ResolveTransition to be %s, got %s", scenario.ExpectedOutput.ResolveTransition, got.ResolveTransition)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides("", &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

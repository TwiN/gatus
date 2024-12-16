package github

import (
	"net/http"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
	"github.com/google/go-github/v48/github"
)

func TestAlertProvider_Validate(t *testing.T) {
	scenarios := []struct {
		Name          string
		Provider      AlertProvider
		ExpectedError bool
	}{
		{
			Name:          "invalid",
			Provider:      AlertProvider{DefaultConfig: Config{RepositoryURL: "", Token: ""}},
			ExpectedError: true,
		},
		{
			Name:          "invalid-token",
			Provider:      AlertProvider{DefaultConfig: Config{RepositoryURL: "https://github.com/TwiN/test", Token: "12345"}},
			ExpectedError: true,
		},
		{
			Name:          "missing-repository-name",
			Provider:      AlertProvider{DefaultConfig: Config{RepositoryURL: "https://github.com/TwiN", Token: "12345"}},
			ExpectedError: true,
		},
		{
			Name:          "enterprise-client",
			Provider:      AlertProvider{DefaultConfig: Config{RepositoryURL: "https://github.example.com/TwiN/test", Token: "12345"}},
			ExpectedError: true,
		},
		{
			Name:          "invalid-url",
			Provider:      AlertProvider{DefaultConfig: Config{RepositoryURL: "github.com/TwiN/test", Token: "12345"}},
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			err := scenario.Provider.Validate()
			if scenario.ExpectedError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.ExpectedError && err != nil && !strings.Contains(err.Error(), "user does not exist") && !strings.Contains(err.Error(), "no such host") {
				t.Error("expected no error, got", err.Error())
			}
		})
	}
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:          "triggered-error",
			Provider:      AlertProvider{DefaultConfig: Config{RepositoryURL: "https://github.com/TwiN/test", Token: "12345"}},
			Alert:         alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      false,
			ExpectedError: true,
		},
		{
			Name:          "resolved-error",
			Provider:      AlertProvider{DefaultConfig: Config{RepositoryURL: "https://github.com/TwiN/test", Token: "12345"}},
			Alert:         alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      true,
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg, err := scenario.Provider.GetConfig("", &scenario.Alert)
			if err != nil && !strings.Contains(err.Error(), "failed to retrieve GitHub user") && !strings.Contains(err.Error(), "no such host") {
				t.Error("expected no error, got", err.Error())
			}
			cfg.githubClient = github.NewClient(nil)
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			err = scenario.Provider.Send(
				&endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
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

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	scenarios := []struct {
		Name         string
		Endpoint     endpoint.Endpoint
		Provider     AlertProvider
		Alert        alert.Alert
		NoConditions bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name", URL: "https://example.org"},
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, FailureThreshold: 3},
			ExpectedBody: "An alert for **endpoint-name** has been triggered due to having failed 3 time(s) in a row:\n> description-1\n\n## Condition results\n- :white_check_mark: - `[CONNECTED] == true`\n- :x: - `[STATUS] == 200`",
		},
		{
			Name:         "triggered-with-no-description",
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name", URL: "https://example.org"},
			Provider:     AlertProvider{},
			Alert:        alert.Alert{FailureThreshold: 10},
			ExpectedBody: "An alert for **endpoint-name** has been triggered due to having failed 10 time(s) in a row\n\n## Condition results\n- :white_check_mark: - `[CONNECTED] == true`\n- :x: - `[STATUS] == 200`",
		},
		{
			Name:         "triggered-with-no-conditions",
			NoConditions: true,
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name", URL: "https://example.org"},
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, FailureThreshold: 10},
			ExpectedBody: "An alert for **endpoint-name** has been triggered due to having failed 10 time(s) in a row:\n> description-1",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var conditionResults []*endpoint.ConditionResult
			if !scenario.NoConditions {
				conditionResults = []*endpoint.ConditionResult{
					{Condition: "[CONNECTED] == true", Success: true},
					{Condition: "[STATUS] == 200", Success: false},
				}
			}
			body := scenario.Provider.buildIssueBody(
				&scenario.Endpoint,
				&scenario.Alert,
				&endpoint.Result{ConditionResults: conditionResults},
			)
			if strings.TrimSpace(body) != strings.TrimSpace(scenario.ExpectedBody) {
				t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
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
			Name: "provider-no-override-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{RepositoryURL: "https://github.com/TwiN/test", Token: "12345"},
			},
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{RepositoryURL: "https://github.com/TwiN/test", Token: "12345"},
		},
		{
			Name: "provider-with-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{RepositoryURL: "https://github.com/TwiN/test", Token: "12345"},
			},
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"repository-url": "https://github.com/TwiN/alert-test", "token": "54321"}},
			ExpectedOutput: Config{RepositoryURL: "https://github.com/TwiN/alert-test", Token: "54321"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig("", &scenario.InputAlert)
			if err != nil && !strings.Contains(err.Error(), "failed to retrieve GitHub user") && !strings.Contains(err.Error(), "no such host") {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.RepositoryURL != scenario.ExpectedOutput.RepositoryURL {
				t.Errorf("expected repository URL %s, got %s", scenario.ExpectedOutput.RepositoryURL, got.RepositoryURL)
			}
			if got.Token != scenario.ExpectedOutput.Token {
				t.Errorf("expected token %s, got %s", scenario.ExpectedOutput.Token, got.Token)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides("", &scenario.InputAlert); err != nil && !strings.Contains(err.Error(), "failed to retrieve GitHub user") {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

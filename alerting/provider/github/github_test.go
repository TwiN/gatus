package github

import (
	"net/http"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/test"
	"github.com/google/go-github/v48/github"
)

func TestAlertDefaultProvider_IsValid(t *testing.T) {
	scenarios := []struct {
		Name     string
		Provider AlertProvider
		Expected bool
	}{
		{
			Name:     "invalid",
			Provider: AlertProvider{RepositoryURL: "", Token: ""},
			Expected: false,
		},
		{
			Name:     "invalid-token",
			Provider: AlertProvider{RepositoryURL: "https://github.com/TwiN/test", Token: "12345"},
			Expected: false,
		},
		{
			Name:     "missing-repository-name",
			Provider: AlertProvider{RepositoryURL: "https://github.com/TwiN", Token: "12345"},
			Expected: false,
		},
		{
			Name:     "enterprise-client",
			Provider: AlertProvider{RepositoryURL: "https://github.example.com/TwiN/test", Token: "12345"},
			Expected: false,
		},
		{
			Name:     "invalid-url",
			Provider: AlertProvider{RepositoryURL: "github.com/TwiN/test", Token: "12345"},
			Expected: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if scenario.Provider.IsValid() != scenario.Expected {
				t.Errorf("expected %t, got %t", scenario.Expected, scenario.Provider.IsValid())
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
			Provider:      AlertProvider{RepositoryURL: "https://github.com/TwiN/test", Token: "12345"},
			Alert:         alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      false,
			ExpectedError: true,
		},
		{
			Name:          "resolved-error",
			Provider:      AlertProvider{RepositoryURL: "https://github.com/TwiN/test", Token: "12345"},
			Alert:         alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:      true,
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			scenario.Provider.githubClient = github.NewClient(nil)
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			err := scenario.Provider.Send(
				&core.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
				&scenario.Alert,
				&core.Result{
					ConditionResults: []*core.ConditionResult{
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
		Endpoint     core.Endpoint
		Provider     AlertProvider
		Alert        alert.Alert
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Endpoint:     core.Endpoint{Name: "endpoint-name", URL: "https://example.org"},
			Provider:     AlertProvider{},
			Alert:        alert.Alert{Description: &firstDescription, FailureThreshold: 3},
			ExpectedBody: "An alert for **endpoint-name** has been triggered due to having failed 3 time(s) in a row:\n> description-1\n\n## Condition results\n- :white_check_mark: - `[CONNECTED] == true`\n- :x: - `[STATUS] == 200`",
		},
		{
			Name:         "no-description",
			Endpoint:     core.Endpoint{Name: "endpoint-name", URL: "https://example.org"},
			Provider:     AlertProvider{},
			Alert:        alert.Alert{FailureThreshold: 10},
			ExpectedBody: "An alert for **endpoint-name** has been triggered due to having failed 10 time(s) in a row\n\n## Condition results\n- :white_check_mark: - `[CONNECTED] == true`\n- :x: - `[STATUS] == 200`",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildIssueBody(
				&scenario.Endpoint,
				&scenario.Alert,
				&core.Result{
					ConditionResults: []*core.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: true},
						{Condition: "[STATUS] == 200", Success: false},
					},
				},
			)
			if strings.TrimSpace(body) != strings.TrimSpace(scenario.ExpectedBody) {
				t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
			}
		})
	}
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	if (AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

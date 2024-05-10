package pagerduty

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{IntegrationKey: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{IntegrationKey: "00000000000000000000000000000000"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_IsValidWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				IntegrationKey: "00000000000000000000000000000000",
				Group:          "",
			},
		},
	}
	if providerWithInvalidOverrideGroup.IsValid() {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideIntegrationKey := AlertProvider{
		Overrides: []Override{
			{
				IntegrationKey: "",
				Group:          "group",
			},
		},
	}
	if providerWithInvalidOverrideIntegrationKey.IsValid() {
		t.Error("provider integration key shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		Overrides: []Override{
			{
				IntegrationKey: "00000000000000000000000000000000",
				Group:          "group",
			},
		},
	}
	if !providerWithValidOverride.IsValid() {
		t.Error("provider should've been valid")
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
			Name:     "triggered",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{},
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
	description := "test"
	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{IntegrationKey: "00000000000000000000000000000000"},
			Alert:        alert.Alert{Description: &description},
			Resolved:     false,
			ExpectedBody: "{\"routing_key\":\"00000000000000000000000000000000\",\"dedup_key\":\"\",\"event_action\":\"trigger\",\"payload\":{\"summary\":\"TRIGGERED: endpoint-name - test\",\"source\":\"Gatus\",\"severity\":\"critical\"}}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{IntegrationKey: "00000000000000000000000000000000"},
			Alert:        alert.Alert{Description: &description, ResolveKey: "key"},
			Resolved:     true,
			ExpectedBody: "{\"routing_key\":\"00000000000000000000000000000000\",\"dedup_key\":\"key\",\"event_action\":\"resolve\",\"payload\":{\"summary\":\"RESOLVED: endpoint-name - test\",\"source\":\"Gatus\",\"severity\":\"critical\"}}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(&endpoint.Endpoint{Name: "endpoint-name"}, &scenario.Alert, &endpoint.Result{}, scenario.Resolved)
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

func TestAlertProvider_getIntegrationKeyForGroup(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		ExpectedOutput string
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				IntegrationKey: "00000000000000000000000000000001",
				Overrides:      nil,
			},
			InputGroup:     "",
			ExpectedOutput: "00000000000000000000000000000001",
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				IntegrationKey: "00000000000000000000000000000001",
				Overrides:      nil,
			},
			InputGroup:     "group",
			ExpectedOutput: "00000000000000000000000000000001",
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				IntegrationKey: "00000000000000000000000000000001",
				Overrides: []Override{
					{
						Group:          "group",
						IntegrationKey: "00000000000000000000000000000002",
					},
				},
			},
			InputGroup:     "",
			ExpectedOutput: "00000000000000000000000000000001",
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				IntegrationKey: "00000000000000000000000000000001",
				Overrides: []Override{
					{
						Group:          "group",
						IntegrationKey: "00000000000000000000000000000002",
					},
				},
			},
			InputGroup:     "group",
			ExpectedOutput: "00000000000000000000000000000002",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if output := scenario.Provider.getIntegrationKeyForGroup(scenario.InputGroup); output != scenario.ExpectedOutput {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput, output)
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

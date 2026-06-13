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

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{DefaultConfig: Config{IntegrationKey: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000000"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000001"},
		Overrides: []Override{
			{
				Config: Config{IntegrationKey: "00000000000000000000000000000002"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000001"},
		Overrides: []Override{
			{
				Config: Config{IntegrationKey: "00000000000000000000000000000002"},
				Group:  "group",
			},
		},
	}
	if err := providerWithValidOverride.Validate(); err != nil {
		t.Error("provider should've been valid, got error:", err.Error())
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
			Provider: AlertProvider{DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000000"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000000"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000000"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000000"}},
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
		Endpoint     endpoint.Endpoint
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000000"}},
			Alert:        alert.Alert{Description: &description},
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name"},
			Resolved:     false,
			ExpectedBody: "{\"routing_key\":\"00000000000000000000000000000000\",\"dedup_key\":\"\",\"event_action\":\"trigger\",\"payload\":{\"summary\":\"TRIGGERED: endpoint-name - test\",\"source\":\"Gatus\",\"severity\":\"critical\"}}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000000"}},
			Alert:        alert.Alert{Description: &description, ResolveKey: "key"},
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name"},
			Resolved:     true,
			ExpectedBody: "{\"routing_key\":\"00000000000000000000000000000000\",\"dedup_key\":\"key\",\"event_action\":\"resolve\",\"payload\":{\"summary\":\"RESOLVED: endpoint-name - test\",\"source\":\"Gatus\",\"severity\":\"critical\"}}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(&scenario.Provider.DefaultConfig, &scenario.Endpoint, &scenario.Alert, &endpoint.Result{}, scenario.Resolved)
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

func TestAlertProvider_buildRequestBodyWithExtraLabels(t *testing.T) {
	description := "test"
	ep := &endpoint.Endpoint{
		Name:        "endpoint-name",
		ExtraLabels: map[string]string{"environment": "production", "region": "es"},
	}
	provider := AlertProvider{DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000000"}}
	al := alert.Alert{Description: &description}
	body := provider.buildRequestBody(&provider.DefaultConfig, ep, &al, &endpoint.Result{}, false)
	var out Body
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("expected body to be valid JSON, got error: %s", err.Error())
	}
	if out.Payload.CustomDetails == nil {
		t.Fatal("expected custom_details to be present")
	}
	if out.Payload.CustomDetails["environment"] != "production" {
		t.Errorf("expected custom_details.environment=production, got %s", out.Payload.CustomDetails["environment"])
	}
	if out.Payload.CustomDetails["region"] != "es" {
		t.Errorf("expected custom_details.region=es, got %s", out.Payload.CustomDetails["region"])
	}
}

func TestAlertProvider_buildRequestBodyWithoutExtraLabels(t *testing.T) {
	description := "test"
	ep := &endpoint.Endpoint{Name: "endpoint-name"}
	provider := AlertProvider{DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000000"}}
	al := alert.Alert{Description: &description}
	body := provider.buildRequestBody(&provider.DefaultConfig, ep, &al, &endpoint.Result{}, false)
	var out Body
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("expected body to be valid JSON, got error: %s", err.Error())
	}
	if out.Payload.CustomDetails != nil {
		t.Error("expected custom_details to be absent when no extra-labels are defined")
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
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000001"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{IntegrationKey: "00000000000000000000000000000001"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000001"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{IntegrationKey: "00000000000000000000000000000001"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000001"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{IntegrationKey: "00000000000000000000000000000002"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{IntegrationKey: "00000000000000000000000000000001"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000001"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{IntegrationKey: "00000000000000000000000000000002"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{IntegrationKey: "00000000000000000000000000000002"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "00000000000000000000000000000001"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{IntegrationKey: "00000000000000000000000000000002"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"integration-key": "00000000000000000000000000000003"}},
			ExpectedOutput: Config{IntegrationKey: "00000000000000000000000000000003"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.IntegrationKey != scenario.ExpectedOutput.IntegrationKey {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput.IntegrationKey, got.IntegrationKey)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

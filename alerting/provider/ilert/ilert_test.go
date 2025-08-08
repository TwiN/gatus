package ilert

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
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected bool
	}{
		{
			name: "valid",
			provider: AlertProvider{
				DefaultConfig: Config{
					IntegrationKey: "some-random-key",
				},
			},
			expected: true,
		},
		{
			name: "invalid-integration-key",
			provider: AlertProvider{
				DefaultConfig: Config{
					IntegrationKey: "",
				},
			},
			expected: false,
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
	sendOnResolved := true
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name: "triggered",
			Provider: AlertProvider{DefaultConfig: Config{
				IntegrationKey: "some-integration-key",
			}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 3, FailureThreshold: 3, ResolveKey: "123", Type: "ilert", SendOnResolved: &sendOnResolved},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				var b bytes.Buffer

				reader := io.NopCloser(&b)
				return &http.Response{StatusCode: http.StatusAccepted, Body: reader}
			}),
			ExpectedError: false,
		},
		{
			Name: "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{
				IntegrationKey: "some-integration-key",
			}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 3, FailureThreshold: 3, ResolveKey: "123", Type: "ilert", SendOnResolved: &sendOnResolved},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusBadRequest, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name: "resolved",
			Provider: AlertProvider{DefaultConfig: Config{
				IntegrationKey: "some-integration-key",
			}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 3, FailureThreshold: 3, ResolveKey: "123", Type: "ilert", SendOnResolved: &sendOnResolved},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				var b bytes.Buffer
				reader := io.NopCloser(&b)
				return &http.Response{StatusCode: http.StatusAccepted, Body: reader}
			}),
			ExpectedError: false,
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

func TestAlertProvider_BuildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	sendOnResolved := true

	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{DefaultConfig: Config{IntegrationKey: "some-integration-key"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 3, FailureThreshold: 3, ResolveKey: "123", Type: "ilert", SendOnResolved: &sendOnResolved},
			Resolved:     false,
			ExpectedBody: `{"alert":{"Type":"ilert","Enabled":null,"FailureThreshold":3,"SuccessThreshold":3,"MinimumReminderInterval":0,"Description":"description-1","SendOnResolved":true,"ProviderOverride":null,"ResolveKey":"123","Triggered":false},"name":"endpoint-name","group":"","status":"firing","title":"endpoint-name","details":"description-1","condition_results":[{"condition":"[CONNECTED] == true","success":false},{"condition":"[STATUS] == 200","success":false}],"url":""}`,
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{IntegrationKey: "some-integration-key"}},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 4, FailureThreshold: 3, ResolveKey: "123", Type: "ilert", SendOnResolved: &sendOnResolved},
			Resolved:     true,
			ExpectedBody: `{"alert":{"Type":"ilert","Enabled":null,"FailureThreshold":3,"SuccessThreshold":4,"MinimumReminderInterval":0,"Description":"description-1","SendOnResolved":true,"ProviderOverride":null,"ResolveKey":"123","Triggered":false},"name":"endpoint-name","group":"","status":"resolved","title":"endpoint-name","details":"description-1","condition_results":[{"condition":"[CONNECTED] == true","success":true},{"condition":"[STATUS] == 200","success":true}],"url":""}`,
		},
		{
			Name:         "group-override",
			Provider:     AlertProvider{DefaultConfig: Config{IntegrationKey: "some-integration-key"}, Overrides: []Override{{Group: "g", Config: Config{IntegrationKey: "different-integration-key"}}}},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3, ResolveKey: "123", Type: "ilert", SendOnResolved: &sendOnResolved},
			Resolved:     false,
			ExpectedBody: `{"alert":{"Type":"ilert","Enabled":null,"FailureThreshold":3,"SuccessThreshold":5,"MinimumReminderInterval":0,"Description":"description-2","SendOnResolved":true,"ProviderOverride":null,"ResolveKey":"123","Triggered":false},"name":"endpoint-name","group":"","status":"firing","title":"endpoint-name","details":"description-2","condition_results":[{"condition":"[CONNECTED] == true","success":false},{"condition":"[STATUS] == 200","success":false}],"url":""}`,
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

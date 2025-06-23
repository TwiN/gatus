package homeassistant

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
	invalidProvider := AlertProvider{DefaultConfig: Config{URL: "", Token: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	invalidProviderNoToken := AlertProvider{DefaultConfig: Config{URL: "http://homeassistant:8123", Token: ""}}
	if err := invalidProviderNoToken.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{URL: "http://homeassistant:8123", Token: "token"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{URL: "http://homeassistant:8123", Token: "token"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{URL: "http://homeassistant:8123", Token: "token"},
		Overrides: []Override{
			{
				Config: Config{URL: "http://homeassistant:8123", Token: "token"},
				Group:  "group",
			},
		},
	}
	if err := providerWithValidOverride.Validate(); err != nil {
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
			Provider: AlertProvider{DefaultConfig: Config{URL: "http://homeassistant:8123", Token: "token"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{URL: "http://homeassistant:8123", Token: "token"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{URL: "http://homeassistant:8123", Token: "token"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
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
						{Condition: "SUCCESSFUL_CONDITION", Success: true},
						{Condition: "FAILING_CONDITION", Success: false},
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
	description := "test-description"
	provider := AlertProvider{DefaultConfig: Config{URL: "http://homeassistant:8123", Token: "token"}}
	body := provider.buildRequestBody(
		&endpoint.Endpoint{Name: "endpoint-name"},
		&alert.Alert{Description: &description, SuccessThreshold: 5, FailureThreshold: 3},
		&endpoint.Result{
			ConditionResults: []*endpoint.ConditionResult{
				{Condition: "SUCCESSFUL_CONDITION", Success: true},
				{Condition: "FAILING_CONDITION", Success: false},
			},
		},
		false,
	)
	var decodedBody Body
	if err := json.Unmarshal(body, &decodedBody); err != nil {
		t.Error("expected body to be valid JSON, got error:", err.Error())
	}
	if decodedBody.EventType != "gatus_alert" {
		t.Errorf("expected event_type to be gatus_alert, got %s", decodedBody.EventType)
	}
	if decodedBody.EventData.Status != "triggered" {
		t.Errorf("expected status to be triggered, got %s", decodedBody.EventData.Status)
	}
	if decodedBody.EventData.Description != description {
		t.Errorf("expected description to be %s, got %s", description, decodedBody.EventData.Description)
	}
	if len(decodedBody.EventData.Conditions) != 2 {
		t.Errorf("expected 2 conditions, got %d", len(decodedBody.EventData.Conditions))
	}
	if !decodedBody.EventData.Conditions[0].Success {
		t.Error("expected first condition to be successful")
	}
	if decodedBody.EventData.Conditions[1].Success {
		t.Error("expected second condition to be unsuccessful")
	}
}

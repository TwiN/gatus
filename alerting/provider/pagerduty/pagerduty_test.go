package pagerduty

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/core"
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

func TestAlertProvider_ToCustomAlertProviderWithResolvedAlert(t *testing.T) {
	provider := AlertProvider{IntegrationKey: "00000000000000000000000000000000"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Endpoint{}, &alert.Alert{}, &core.Result{}, true)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "RESOLVED") {
		t.Error("customAlertProvider.Body should've contained the substring RESOLVED")
	}
	if customAlertProvider.URL != "https://events.pagerduty.com/v2/enqueue" {
		t.Errorf("expected URL to be %s, got %s", "https://events.pagerduty.com/v2/enqueue", customAlertProvider.URL)
	}
	if customAlertProvider.Method != http.MethodPost {
		t.Errorf("expected method to be %s, got %s", http.MethodPost, customAlertProvider.Method)
	}
	body := make(map[string]interface{})
	err := json.Unmarshal([]byte(customAlertProvider.Body), &body)
	if err != nil {
		t.Error("expected body to be valid JSON, got error:", err.Error())
	}
}

func TestAlertProvider_ToCustomAlertProviderWithResolvedAlertAndOverride(t *testing.T) {
	provider := AlertProvider{
		IntegrationKey: "",
		Overrides: []Override{
			{
				IntegrationKey: "00000000000000000000000000000000",
				Group:          "group",
			},
		},
	}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Endpoint{}, &alert.Alert{}, &core.Result{}, true)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "RESOLVED") {
		t.Error("customAlertProvider.Body should've contained the substring RESOLVED")
	}
	if customAlertProvider.URL != "https://events.pagerduty.com/v2/enqueue" {
		t.Errorf("expected URL to be %s, got %s", "https://events.pagerduty.com/v2/enqueue", customAlertProvider.URL)
	}
	if customAlertProvider.Method != http.MethodPost {
		t.Errorf("expected method to be %s, got %s", http.MethodPost, customAlertProvider.Method)
	}
	body := make(map[string]interface{})
	err := json.Unmarshal([]byte(customAlertProvider.Body), &body)
	if err != nil {
		t.Error("expected body to be valid JSON, got error:", err.Error())
	}
}

func TestAlertProvider_ToCustomAlertProviderWithTriggeredAlert(t *testing.T) {
	provider := AlertProvider{IntegrationKey: "00000000000000000000000000000000"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Endpoint{}, &alert.Alert{}, &core.Result{}, false)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "TRIGGERED") {
		t.Error("customAlertProvider.Body should've contained the substring TRIGGERED")
	}
	if customAlertProvider.URL != "https://events.pagerduty.com/v2/enqueue" {
		t.Errorf("expected URL to be %s, got %s", "https://events.pagerduty.com/v2/enqueue", customAlertProvider.URL)
	}
	if customAlertProvider.Method != http.MethodPost {
		t.Errorf("expected method to be %s, got %s", http.MethodPost, customAlertProvider.Method)
	}
	body := make(map[string]interface{})
	err := json.Unmarshal([]byte(customAlertProvider.Body), &body)
	if err != nil {
		t.Error("expected body to be valid JSON, got error:", err.Error())
	}
}

func TestAlertProvider_ToCustomAlertProviderWithTriggeredAlertAndOverride(t *testing.T) {
	provider := AlertProvider{
		IntegrationKey: "",
		Overrides: []Override{
			{
				IntegrationKey: "00000000000000000000000000000000",
				Group:          "group",
			},
		},
	}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Endpoint{}, &alert.Alert{}, &core.Result{}, false)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "TRIGGERED") {
		t.Error("customAlertProvider.Body should've contained the substring TRIGGERED")
	}
	if customAlertProvider.URL != "https://events.pagerduty.com/v2/enqueue" {
		t.Errorf("expected URL to be %s, got %s", "https://events.pagerduty.com/v2/enqueue", customAlertProvider.URL)
	}
	if customAlertProvider.Method != http.MethodPost {
		t.Errorf("expected method to be %s, got %s", http.MethodPost, customAlertProvider.Method)
	}
	body := make(map[string]interface{})
	err := json.Unmarshal([]byte(customAlertProvider.Body), &body)
	if err != nil {
		t.Error("expected body to be valid JSON, got error:", err.Error())
	}
}

func TestAlertProvider_getPagerDutyIntegrationKey(t *testing.T) {
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
			if output := scenario.Provider.getPagerDutyIntegrationKeyForGroup(scenario.InputGroup); output != scenario.ExpectedOutput {
				t.Errorf("expected %s, got %s", scenario.ExpectedOutput, output)
			}
		})
	}
}

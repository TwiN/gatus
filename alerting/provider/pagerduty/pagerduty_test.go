package pagerduty

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/TwinProduction/gatus/alerting/alert"
	"github.com/TwinProduction/gatus/core"
)

func TestAlertDefaultProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{IntegrationKey: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{IntegrationKey: "00000000000000000000000000000000"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}
func TestAlertPerGroupProvider_IsValid(t *testing.T) {
	invalidGroup := Integrations{
		IntegrationKey: "00000000000000000000000000000000",
		Group:          "",
	}
	integrations := []Integrations{}
	integrations = append(integrations, invalidGroup)
	invalidProviderGroupNameError := AlertProvider{
		Integrations: integrations,
	}
	if invalidProviderGroupNameError.IsValid() {
		t.Error("provider Group shouldn't have been valid")
	}
	invalidIntegrationKey := Integrations{
		IntegrationKey: "",
		Group:          "group",
	}
	integrations = []Integrations{}
	integrations = append(integrations, invalidIntegrationKey)
	invalidProviderIntegrationKey := AlertProvider{
		Integrations: integrations,
	}
	if invalidProviderIntegrationKey.IsValid() {
		t.Error("provider integration key shouldn't have been valid")
	}
	validIntegration := Integrations{
		IntegrationKey: "00000000000000000000000000000000",
		Group:          "group",
	}
	integrations = []Integrations{}
	integrations = append(integrations, validIntegration)
	validProvider := AlertProvider{
		Integrations: integrations,
	}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ToCustomAlertProviderWithResolvedAlert(t *testing.T) {
	provider := AlertProvider{IntegrationKey: "00000000000000000000000000000000"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &alert.Alert{}, &core.Result{}, true)
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

func TestAlertPerGroupProvider_ToCustomAlertProviderWithResolvedAlert(t *testing.T) {
	validIntegration := Integrations{
		IntegrationKey: "00000000000000000000000000000000",
		Group:          "group",
	}
	integrations := []Integrations{}
	integrations = append(integrations, validIntegration)
	provider := AlertProvider{
		IntegrationKey: "",
		Integrations:   integrations,
	}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &alert.Alert{}, &core.Result{}, true)
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
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &alert.Alert{}, &core.Result{}, false)
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

func TestAlertPerGroupProvider_ToCustomAlertProviderWithTriggeredAlert(t *testing.T) {
	validIntegration := Integrations{
		IntegrationKey: "00000000000000000000000000000000",
		Group:          "group",
	}
	integrations := []Integrations{}
	integrations = append(integrations, validIntegration)
	provider := AlertProvider{
		IntegrationKey: "",
		Integrations:   integrations,
	}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &alert.Alert{}, &core.Result{}, false)
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

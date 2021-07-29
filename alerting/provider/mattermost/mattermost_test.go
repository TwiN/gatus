package mattermost

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/TwinProduction/gatus/alerting/alert"
	"github.com/TwinProduction/gatus/core"
)

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{WebhookURL: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{WebhookURL: "http://example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ToCustomAlertProviderWithResolvedAlert(t *testing.T) {
	provider := AlertProvider{WebhookURL: "http://example.org"}
	alertDescription := "test"
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{Name: "svc"}, &alert.Alert{Description: &alertDescription}, &core.Result{ConditionResults: []*core.ConditionResult{{Condition: "SUCCESSFUL_CONDITION", Success: true}}}, true)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "resolved") {
		t.Error("customAlertProvider.Body should've contained the substring resolved")
	}
	if customAlertProvider.URL != "http://example.org" {
		t.Errorf("expected URL to be %s, got %s", "http://example.org", customAlertProvider.URL)
	}
	if customAlertProvider.Method != http.MethodPost {
		t.Errorf("expected method to be %s, got %s", http.MethodPost, customAlertProvider.Method)
	}
	body := make(map[string]interface{})
	err := json.Unmarshal([]byte(customAlertProvider.Body), &body)
	if err != nil {
		t.Error("expected body to be valid JSON, got error:", err.Error())
	}
	if expected := "An alert for *svc* has been resolved after passing successfully 0 time(s) in a row:\n> test"; expected != body["attachments"].([]interface{})[0].(map[string]interface{})["text"] {
		t.Errorf("expected $.attachments[0].description to be %s, got %s", expected, body["attachments"].([]interface{})[0].(map[string]interface{})["text"])
	}
}

func TestAlertProvider_ToCustomAlertProviderWithTriggeredAlert(t *testing.T) {
	provider := AlertProvider{WebhookURL: "http://example.org"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &alert.Alert{}, &core.Result{ConditionResults: []*core.ConditionResult{{Condition: "UNSUCCESSFUL_CONDITION", Success: false}}}, false)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "triggered") {
		t.Error("customAlertProvider.Body should've contained the substring triggered")
	}
	if customAlertProvider.URL != "http://example.org" {
		t.Errorf("expected URL to be %s, got %s", "http://example.org", customAlertProvider.URL)
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

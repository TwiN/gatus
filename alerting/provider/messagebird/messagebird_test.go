package messagebird

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/Meldiron/gatus/core"
)

func TestMessagebirdAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{
		AccessKey:  "1",
		Originator: "1",
		Recipients: "1",
	}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ToCustomAlertProviderWithResolvedAlert(t *testing.T) {
	provider := AlertProvider{
		AccessKey:  "1",
		Originator: "1",
		Recipients: "1",
	}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &core.Alert{}, &core.Result{}, true)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "RESOLVED") {
		t.Error("customAlertProvider.Body should've contained the substring RESOLVED")
	}
	if customAlertProvider.URL != "https://rest.messagebird.com/messages" {
		t.Errorf("expected URL to be %s, got %s", "https://rest.messagebird.com/messages", customAlertProvider.URL)
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
	provider := AlertProvider{
		AccessKey:  "1",
		Originator: "1",
		Recipients: "1",
	}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &core.Alert{}, &core.Result{}, false)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "TRIGGERED") {
		t.Error("customAlertProvider.Body should've contained the substring TRIGGERED")
	}
	if customAlertProvider.URL != "https://rest.messagebird.com/messages" {
		t.Errorf("expected URL to be %s, got %s", "https://rest.messagebird.com/messages", customAlertProvider.URL)
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

package telegram

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/TwinProduction/gatus/core"
)

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{Token: "", ID: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ToCustomAlertProviderWithResolvedAlert(t *testing.T) {
	provider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "12345678"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &core.Alert{}, &core.Result{ConditionResults: []*core.ConditionResult{{Condition: "SUCCESSFUL_CONDITION", Success: true}}}, true)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "resolved") {
		t.Error("customAlertProvider.Body should've contained the substring resolved")
	}
	if customAlertProvider.URL != fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", provider.Token) {
		t.Errorf("expected URL to be %s, got %s", fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", provider.Token), customAlertProvider.URL)
	}
	if customAlertProvider.Method != http.MethodPost {
		t.Errorf("expected method to be %s, got %s", http.MethodPost, customAlertProvider.Method)
	}
	body := make(map[string]interface{})
	err := json.Unmarshal([]byte(customAlertProvider.Body), &body)
	//_, err := json.Marshal(customAlertProvider.Body)
	if err != nil {
		t.Error("expected body to be valid JSON, got error:", err.Error())
	}
}

func TestAlertProvider_ToCustomAlertProviderWithTriggeredAlert(t *testing.T) {
	provider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "0123456789"}
	description := "Healthcheck Successful"
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &core.Alert{Description: &description}, &core.Result{ConditionResults: []*core.ConditionResult{{Condition: "UNSUCCESSFUL_CONDITION", Success: false}}}, false)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "triggered") {
		t.Error("customAlertProvider.Body should've contained the substring triggered")
	}
	if customAlertProvider.URL != fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", provider.Token) {
		t.Errorf("expected URL to be %s, got %s", fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", provider.Token), customAlertProvider.URL)
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

func TestAlertProvider_ToCustomAlertProviderWithDescription(t *testing.T) {
	provider := AlertProvider{Token: "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", ID: "0123456789"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &core.Alert{}, &core.Result{ConditionResults: []*core.ConditionResult{{Condition: "UNSUCCESSFUL_CONDITION", Success: false}}}, false)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "triggered") {
		t.Error("customAlertProvider.Body should've contained the substring triggered")
	}
	if customAlertProvider.URL != fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", provider.Token) {
		t.Errorf("expected URL to be %s, got %s", fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", provider.Token), customAlertProvider.URL)
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

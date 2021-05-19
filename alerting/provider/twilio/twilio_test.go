package twilio

import (
	"net/http"
	"strings"
	"testing"

	"github.com/TwinProduction/gatus/alerting/alert"
	"github.com/TwinProduction/gatus/core"
)

func TestTwilioAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{
		SID:   "1",
		Token: "1",
		From:  "1",
		To:    "1",
	}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ToCustomAlertProviderWithResolvedAlert(t *testing.T) {
	provider := AlertProvider{
		SID:   "1",
		Token: "2",
		From:  "3",
		To:    "4",
	}
	description := "alert-description"
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{Name: "service-name"}, &alert.Alert{Description: &description}, &core.Result{}, true)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "RESOLVED") {
		t.Error("customAlertProvider.Body should've contained the substring RESOLVED")
	}
	if customAlertProvider.URL != "https://api.twilio.com/2010-04-01/Accounts/1/Messages.json" {
		t.Errorf("expected URL to be %s, got %s", "https://api.twilio.com/2010-04-01/Accounts/1/Messages.json", customAlertProvider.URL)
	}
	if customAlertProvider.Method != http.MethodPost {
		t.Errorf("expected method to be %s, got %s", http.MethodPost, customAlertProvider.Method)
	}
	if customAlertProvider.Body != "Body=RESOLVED%3A+service-name+-+alert-description&From=3&To=4" {
		t.Errorf("expected body to be %s, got %s", "Body=RESOLVED%3A+service-name+-+alert-description&From=3&To=4", customAlertProvider.Body)
	}
}

func TestAlertProvider_ToCustomAlertProviderWithTriggeredAlert(t *testing.T) {
	provider := AlertProvider{
		SID:   "4",
		Token: "3",
		From:  "2",
		To:    "1",
	}
	description := "alert-description"
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{Name: "service-name"}, &alert.Alert{Description: &description}, &core.Result{}, false)
	if customAlertProvider == nil {
		t.Fatal("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "TRIGGERED") {
		t.Error("customAlertProvider.Body should've contained the substring TRIGGERED")
	}
	if customAlertProvider.URL != "https://api.twilio.com/2010-04-01/Accounts/4/Messages.json" {
		t.Errorf("expected URL to be %s, got %s", "https://api.twilio.com/2010-04-01/Accounts/4/Messages.json", customAlertProvider.URL)
	}
	if customAlertProvider.Method != http.MethodPost {
		t.Errorf("expected method to be %s, got %s", http.MethodPost, customAlertProvider.Method)
	}
	if customAlertProvider.Body != "Body=TRIGGERED%3A+service-name+-+alert-description&From=2&To=1" {
		t.Errorf("expected body to be %s, got %s", "Body=TRIGGERED%3A+service-name+-+alert-description&From=2&To=1", customAlertProvider.Body)
	}
}

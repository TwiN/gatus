package mattermost

import (
	"strings"
	"testing"

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
	provider := AlertProvider{WebhookURL: "http://example.com"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &core.Alert{}, &core.Result{ConditionResults: []*core.ConditionResult{{Condition: "SUCCESSFUL_CONDITION", Success: true}}}, true)
	if customAlertProvider == nil {
		t.Error("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "resolved") {
		t.Error("customAlertProvider.Body should've contained the substring resolved")
	}
}

func TestAlertProvider_ToCustomAlertProviderWithTriggeredAlert(t *testing.T) {
	provider := AlertProvider{WebhookURL: "http://example.com"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &core.Alert{}, &core.Result{ConditionResults: []*core.ConditionResult{{Condition: "UNSUCCESSFUL_CONDITION", Success: false}}}, false)
	if customAlertProvider == nil {
		t.Error("customAlertProvider shouldn't have been nil")
	}
	if !strings.Contains(customAlertProvider.Body, "triggered") {
		t.Error("customAlertProvider.Body should've contained the substring triggered")
	}
}

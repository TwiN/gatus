package alerting

import "testing"

func TestSlackAlertProvider_IsValid(t *testing.T) {
	invalidProvider := SlackAlertProvider{WebhookUrl: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := SlackAlertProvider{WebhookUrl: "http://example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

package slack

import "testing"

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{WebhookUrl: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{WebhookUrl: "http://example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

package alerting

import "testing"

func TestTwilioAlertProvider_IsValid(t *testing.T) {
	invalidProvider := TwilioAlertProvider{}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := TwilioAlertProvider{
		SID:   "1",
		Token: "1",
		From:  "1",
		To:    "1",
	}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

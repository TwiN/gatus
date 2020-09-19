package alerting

import "testing"

func TestPagerDutyAlertProvider_IsValid(t *testing.T) {
	invalidProvider := PagerDutyAlertProvider{IntegrationKey: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := PagerDutyAlertProvider{IntegrationKey: "00000000000000000000000000000000"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

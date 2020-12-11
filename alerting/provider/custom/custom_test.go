package custom

import (
	"io/ioutil"
	"testing"

	"github.com/TwinProduction/gatus/core"
)

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{URL: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{URL: "http://example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_buildHTTPRequestWhenResolved(t *testing.T) {
	const (
		ExpectedURL  = "http://example.com/service-name?event=RESOLVED&description=alert-description"
		ExpectedBody = "service-name,alert-description,RESOLVED"
	)
	customAlertProvider := &AlertProvider{
		URL:     "http://example.com/[SERVICE_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
		Body:    "[SERVICE_NAME],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers: nil,
	}
	request := customAlertProvider.buildHTTPRequest("service-name", "alert-description", true)
	if request.URL.String() != ExpectedURL {
		t.Error("expected URL to be", ExpectedURL, "was", request.URL.String())
	}
	body, _ := ioutil.ReadAll(request.Body)
	if string(body) != ExpectedBody {
		t.Error("expected body to be", ExpectedBody, "was", string(body))
	}
}

func TestAlertProvider_buildHTTPRequestWhenTriggered(t *testing.T) {
	const (
		ExpectedURL  = "http://example.com/service-name?event=TRIGGERED&description=alert-description"
		ExpectedBody = "service-name,alert-description,TRIGGERED"
	)
	customAlertProvider := &AlertProvider{
		URL:     "http://example.com/[SERVICE_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
		Body:    "[SERVICE_NAME],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers: map[string]string{"Authorization": "Basic hunter2"},
	}
	request := customAlertProvider.buildHTTPRequest("service-name", "alert-description", false)
	if request.URL.String() != ExpectedURL {
		t.Error("expected URL to be", ExpectedURL, "was", request.URL.String())
	}
	body, _ := ioutil.ReadAll(request.Body)
	if string(body) != ExpectedBody {
		t.Error("expected body to be", ExpectedBody, "was", string(body))
	}
}

func TestAlertProvider_ToCustomAlertProvider(t *testing.T) {
	provider := AlertProvider{URL: "http://example.com"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &core.Alert{}, &core.Result{}, true)
	if customAlertProvider == nil {
		t.Error("customAlertProvider shouldn't have been nil")
	}
	if customAlertProvider != customAlertProvider {
		t.Error("customAlertProvider should've been equal to customAlertProvider")
	}
}

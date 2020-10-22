package custom

import (
	"github.com/TwinProduction/gatus/core"
	"io/ioutil"
	"testing"
)

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{Url: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{Url: "http://example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_buildRequestWhenResolved(t *testing.T) {
	const (
		ExpectedUrl  = "http://example.com/service-name"
		ExpectedBody = "service-name,alert-description,RESOLVED"
	)
	customAlertProvider := &AlertProvider{
		Url:     "http://example.com/[SERVICE_NAME]",
		Method:  "GET",
		Body:    "[SERVICE_NAME],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers: nil,
	}
	request := customAlertProvider.buildRequest("service-name", "alert-description", true)
	if request.URL.String() != ExpectedUrl {
		t.Error("expected URL to be", ExpectedUrl, "was", request.URL.String())
	}
	body, _ := ioutil.ReadAll(request.Body)
	if string(body) != ExpectedBody {
		t.Error("expected body to be", ExpectedBody, "was", string(body))
	}
}

func TestAlertProvider_buildRequestWhenTriggered(t *testing.T) {
	const (
		ExpectedUrl  = "http://example.com/service-name"
		ExpectedBody = "service-name,alert-description,TRIGGERED"
	)
	customAlertProvider := &AlertProvider{
		Url:     "http://example.com/[SERVICE_NAME]",
		Method:  "GET",
		Body:    "[SERVICE_NAME],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers: nil,
	}
	request := customAlertProvider.buildRequest("service-name", "alert-description", false)
	if request.URL.String() != ExpectedUrl {
		t.Error("expected URL to be", ExpectedUrl, "was", request.URL.String())
	}
	body, _ := ioutil.ReadAll(request.Body)
	if string(body) != ExpectedBody {
		t.Error("expected body to be", ExpectedBody, "was", string(body))
	}
}

func TestAlertProvider_ToCustomAlertProvider(t *testing.T) {
	provider := AlertProvider{Url: "http://example.com"}
	customAlertProvider := provider.ToCustomAlertProvider(&core.Service{}, &core.Alert{}, &core.Result{}, true)
	if customAlertProvider == nil {
		t.Error("customAlertProvider shouldn't have been nil")
	}
	if customAlertProvider != customAlertProvider {
		t.Error("customAlertProvider should've been equal to customAlertProvider")
	}
}

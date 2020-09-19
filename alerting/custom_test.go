package alerting

import (
	"io/ioutil"
	"testing"
)

func TestCustomAlertProvider_IsValid(t *testing.T) {
	invalidProvider := CustomAlertProvider{Url: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := CustomAlertProvider{Url: "http://example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestCustomAlertProvider_buildRequestWhenResolved(t *testing.T) {
	const (
		ExpectedUrl  = "http://example.com/service-name"
		ExpectedBody = "service-name,alert-description,RESOLVED"
	)
	customAlertProvider := &CustomAlertProvider{
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

func TestCustomAlertProvider_buildRequestWhenTriggered(t *testing.T) {
	const (
		ExpectedUrl  = "http://example.com/service-name"
		ExpectedBody = "service-name,alert-description,TRIGGERED"
	)
	customAlertProvider := &CustomAlertProvider{
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

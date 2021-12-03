package custom

import (
	"io/ioutil"
	"testing"

	"github.com/TwiN/gatus/v3/alerting/alert"
)

func TestAlertProvider_IsValid(t *testing.T) {
	invalidProvider := AlertProvider{URL: ""}
	if invalidProvider.IsValid() {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{URL: "https://example.com"}
	if !validProvider.IsValid() {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_buildHTTPRequestWhenResolved(t *testing.T) {
	const (
		ExpectedURL  = "https://example.com/endpoint-name?event=RESOLVED&description=alert-description"
		ExpectedBody = "endpoint-name,alert-description,RESOLVED"
	)
	customAlertProvider := &AlertProvider{
		URL:     "https://example.com/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
		Body:    "[ENDPOINT_NAME],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers: nil,
	}
	request := customAlertProvider.buildHTTPRequest("endpoint-name", "alert-description", true)
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
		ExpectedURL  = "https://example.com/endpoint-name?event=TRIGGERED&description=alert-description"
		ExpectedBody = "endpoint-name,alert-description,TRIGGERED"
	)
	customAlertProvider := &AlertProvider{
		URL:     "https://example.com/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
		Body:    "[ENDPOINT_NAME],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers: map[string]string{"Authorization": "Basic hunter2"},
	}
	request := customAlertProvider.buildHTTPRequest("endpoint-name", "alert-description", false)
	if request.URL.String() != ExpectedURL {
		t.Error("expected URL to be", ExpectedURL, "was", request.URL.String())
	}
	body, _ := ioutil.ReadAll(request.Body)
	if string(body) != ExpectedBody {
		t.Error("expected body to be", ExpectedBody, "was", string(body))
	}
}

func TestAlertProvider_buildHTTPRequestWithCustomPlaceholder(t *testing.T) {
	const (
		ExpectedURL  = "https://example.com/endpoint-name?event=test&description=alert-description"
		ExpectedBody = "endpoint-name,alert-description,test"
	)
	customAlertProvider := &AlertProvider{
		URL:     "https://example.com/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
		Body:    "[ENDPOINT_NAME],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers: nil,
		Placeholders: map[string]map[string]string{
			"ALERT_TRIGGERED_OR_RESOLVED": {
				"RESOLVED": "test",
			},
		},
	}
	request := customAlertProvider.buildHTTPRequest("endpoint-name", "alert-description", true)
	if request.URL.String() != ExpectedURL {
		t.Error("expected URL to be", ExpectedURL, "was", request.URL.String())
	}
	body, _ := ioutil.ReadAll(request.Body)
	if string(body) != ExpectedBody {
		t.Error("expected body to be", ExpectedBody, "was", string(body))
	}
}

func TestAlertProvider_GetAlertStatePlaceholderValueDefaults(t *testing.T) {
	customAlertProvider := &AlertProvider{
		URL:          "https://example.com/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
		Body:         "[ENDPOINT_NAME],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers:      nil,
		Placeholders: nil,
	}
	if customAlertProvider.GetAlertStatePlaceholderValue(true) != "RESOLVED" {
		t.Error("expected RESOLVED, got", customAlertProvider.GetAlertStatePlaceholderValue(true))
	}
	if customAlertProvider.GetAlertStatePlaceholderValue(false) != "TRIGGERED" {
		t.Error("expected TRIGGERED, got", customAlertProvider.GetAlertStatePlaceholderValue(false))
	}
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	if (AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

// TestAlertProvider_isBackwardCompatibleWithServiceRename checks if the custom alerting provider still supports
// service placeholders after the migration from "service" to "endpoint"
//
// XXX: Remove this in v4.0.0
func TestAlertProvider_isBackwardCompatibleWithServiceRename(t *testing.T) {
	const (
		ExpectedURL  = "https://example.com/endpoint-name?event=TRIGGERED&description=alert-description"
		ExpectedBody = "endpoint-name,alert-description,TRIGGERED"
	)
	customAlertProvider := &AlertProvider{
		URL:  "https://example.com/[SERVICE_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
		Body: "[SERVICE_NAME],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
	}
	request := customAlertProvider.buildHTTPRequest("endpoint-name", "alert-description", false)
	if request.URL.String() != ExpectedURL {
		t.Error("expected URL to be", ExpectedURL, "was", request.URL.String())
	}
	body, _ := ioutil.ReadAll(request.Body)
	if string(body) != ExpectedBody {
		t.Error("expected body to be", ExpectedBody, "was", string(body))
	}
}

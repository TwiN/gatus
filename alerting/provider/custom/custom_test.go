package custom

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_IsValid(t *testing.T) {
	t.Run("invalid-provider", func(t *testing.T) {
		invalidProvider := AlertProvider{URL: ""}
		if invalidProvider.IsValid() {
			t.Error("provider shouldn't have been valid")
		}
	})
	t.Run("valid-provider", func(t *testing.T) {
		validProvider := AlertProvider{URL: "https://example.com"}
		if validProvider.ClientConfig != nil {
			t.Error("provider client config should have been nil prior to IsValid() being executed")
		}
		if !validProvider.IsValid() {
			t.Error("provider should've been valid")
		}
		if validProvider.ClientConfig == nil {
			t.Error("provider client config should have been set after IsValid() was executed")
		}
	})
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name             string
		Provider         AlertProvider
		Alert            alert.Alert
		Resolved         bool
		MockRoundTripper test.MockRoundTripper
		ExpectedError    bool
	}{
		{
			Name:     "triggered",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: scenario.MockRoundTripper})
			err := scenario.Provider.Send(
				&endpoint.Endpoint{Name: "endpoint-name"},
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			if scenario.ExpectedError && err == nil {
				t.Error("expected error, got none")
			}
			if !scenario.ExpectedError && err != nil {
				t.Error("expected no error, got", err.Error())
			}
		})
	}
}

func TestAlertProvider_buildHTTPRequest(t *testing.T) {
	customAlertProvider := &AlertProvider{
		URL:     "https://example.com/[ENDPOINT_GROUP]/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]&url=[ENDPOINT_URL]",
		Body:    "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ENDPOINT_URL],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers: map[string]string{"Test": "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ENDPOINT_URL],[ALERT_TRIGGERED_OR_RESOLVED]"},
	}
	alertDescription := "alert-description"
	scenarios := []struct {
		AlertProvider  *AlertProvider
		Resolved       bool
		ExpectedURL    string
		ExpectedBody   string
		ExpectedHeader string
	}{
		{
			AlertProvider:  customAlertProvider,
			Resolved:       true,
			ExpectedURL:    "https://example.com/endpoint-group/endpoint-name?event=RESOLVED&description=alert-description&url=https://example.com",
			ExpectedBody:   "endpoint-name,endpoint-group,alert-description,https://example.com,RESOLVED",
			ExpectedHeader: "endpoint-name,endpoint-group,alert-description,https://example.com,RESOLVED",
		},
		{
			AlertProvider:  customAlertProvider,
			Resolved:       false,
			ExpectedURL:    "https://example.com/endpoint-group/endpoint-name?event=TRIGGERED&description=alert-description&url=https://example.com",
			ExpectedBody:   "endpoint-name,endpoint-group,alert-description,https://example.com,TRIGGERED",
			ExpectedHeader: "endpoint-name,endpoint-group,alert-description,https://example.com,TRIGGERED",
		},
	}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("resolved-%v-with-default-placeholders", scenario.Resolved), func(t *testing.T) {
			request := customAlertProvider.buildHTTPRequest(
				&endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group", URL: "https://example.com"},
				&alert.Alert{Description: &alertDescription},
				scenario.Resolved,
			)
			if request.URL.String() != scenario.ExpectedURL {
				t.Error("expected URL to be", scenario.ExpectedURL, "got", request.URL.String())
			}
			body, _ := io.ReadAll(request.Body)
			if string(body) != scenario.ExpectedBody {
				t.Error("expected body to be", scenario.ExpectedBody, "got", string(body))
			}
			header := request.Header.Get("Test")
			if header != scenario.ExpectedHeader {
				t.Error("expected header to be", scenario.ExpectedHeader, "got", header)
			}
		})
	}
}

func TestAlertProvider_buildHTTPRequestWithCustomPlaceholder(t *testing.T) {
	customAlertProvider := &AlertProvider{
		URL:     "https://example.com/[ENDPOINT_GROUP]/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
		Body:    "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		Headers: map[string]string{"Test": "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]"},
		Placeholders: map[string]map[string]string{
			"ALERT_TRIGGERED_OR_RESOLVED": {
				"RESOLVED":  "fixed",
				"TRIGGERED": "boom",
			},
		},
	}
	alertDescription := "alert-description"
	scenarios := []struct {
		AlertProvider  *AlertProvider
		Resolved       bool
		ExpectedURL    string
		ExpectedBody   string
		ExpectedHeader string
	}{
		{
			AlertProvider:  customAlertProvider,
			Resolved:       true,
			ExpectedURL:    "https://example.com/endpoint-group/endpoint-name?event=fixed&description=alert-description",
			ExpectedBody:   "endpoint-name,endpoint-group,alert-description,fixed",
			ExpectedHeader: "endpoint-name,endpoint-group,alert-description,fixed",
		},
		{
			AlertProvider:  customAlertProvider,
			Resolved:       false,
			ExpectedURL:    "https://example.com/endpoint-group/endpoint-name?event=boom&description=alert-description",
			ExpectedBody:   "endpoint-name,endpoint-group,alert-description,boom",
			ExpectedHeader: "endpoint-name,endpoint-group,alert-description,boom",
		},
	}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("resolved-%v-with-custom-placeholders", scenario.Resolved), func(t *testing.T) {
			request := customAlertProvider.buildHTTPRequest(
				&endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
				&alert.Alert{Description: &alertDescription},
				scenario.Resolved,
			)
			if request.URL.String() != scenario.ExpectedURL {
				t.Error("expected URL to be", scenario.ExpectedURL, "got", request.URL.String())
			}
			body, _ := io.ReadAll(request.Body)
			if string(body) != scenario.ExpectedBody {
				t.Error("expected body to be", scenario.ExpectedBody, "got", string(body))
			}
			header := request.Header.Get("Test")
			if header != scenario.ExpectedHeader {
				t.Error("expected header to be", scenario.ExpectedHeader, "got", header)
			}
		})
	}
}

func TestAlertProvider_GetAlertStatePlaceholderValueDefaults(t *testing.T) {
	customAlertProvider := &AlertProvider{
		URL:  "https://example.com/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
		Body: "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
	}
	if customAlertProvider.GetAlertStatePlaceholderValue(true) != "RESOLVED" {
		t.Error("expected RESOLVED, got", customAlertProvider.GetAlertStatePlaceholderValue(true))
	}
	if customAlertProvider.GetAlertStatePlaceholderValue(false) != "TRIGGERED" {
		t.Error("expected TRIGGERED, got", customAlertProvider.GetAlertStatePlaceholderValue(false))
	}
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	if (&AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

func TestAlertProvider_ReplacePlaceholder(t *testing.T) {
	placeholder := "[TEST]"
	content := "replaced"
	scenarios := []struct {
		URL            string
		Body           string
		Header         map[string]string
		ExpectedURL    string
		ExpectedBody   string
		ExpectedHeader string
	}{
		{
			URL:            "https://[TEST]/",
			Body:           "body to be [TEST].",
			Header:         map[string]string{"Test": "header to be [TEST]."},
			ExpectedURL:    "https://replaced/",
			ExpectedBody:   "body to be replaced.",
			ExpectedHeader: "header to be replaced.",
		},
		{
			URL:            "https://TEST/",
			Body:           "body to be TEST.",
			Header:         map[string]string{"Test": "header to be TEST."},
			ExpectedURL:    "https://TEST/",
			ExpectedBody:   "body to be TEST.",
			ExpectedHeader: "header to be TEST.",
		},
	}
	for _, scenario := range scenarios {
		a := &AlertProvider{}
		a.ReplacePlaceholder(placeholder, content, &scenario.Body, &scenario.URL, scenario.Header)
		if scenario.Body != scenario.ExpectedBody {
			t.Error("expected body to be", scenario.ExpectedBody, "got", scenario.Body)
		}
		if scenario.URL != scenario.ExpectedURL {
			t.Error("expected URL to be", scenario.ExpectedURL, "got", scenario.URL)
		}
		if scenario.Header["Test"] != scenario.ExpectedHeader {
			t.Error("expected header to be", scenario.ExpectedHeader, "got", scenario.Header["Test"])
		}
	}
}

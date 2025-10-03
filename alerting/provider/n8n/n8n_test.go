package n8n

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{DefaultConfig: Config{WebhookURL: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{WebhookURL: "https://example.com"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{WebhookURL: "http://example.com"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); err == nil {
		t.Error("provider Group shouldn't have been valid")
	}
	providerWithInvalidOverrideTo := AlertProvider{
		Overrides: []Override{
			{
				Config: Config{WebhookURL: ""},
				Group:  "group",
			},
		},
	}
	if err := providerWithInvalidOverrideTo.Validate(); err == nil {
		t.Error("provider webhook URL shouldn't have been valid")
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{WebhookURL: "http://example.com"},
		Overrides: []Override{
			{
				Config: Config{WebhookURL: "http://example.com"},
				Group:  "group",
			},
		},
	}
	if err := providerWithValidOverride.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
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
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
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

func TestAlertProvider_buildRequestBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Endpoint     endpoint.Endpoint
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody Body
	}{
		{
			Name:     "triggered",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Endpoint: endpoint.Endpoint{Name: "name", URL: "https://example.org"},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			ExpectedBody: Body{
				Title:            "Gatus",
				EndpointName:     "name",
				EndpointURL:      "https://example.org",
				AlertDescription: "description-1",
				Resolved:         false,
				Message:          "An alert for name has been triggered due to having failed 3 time(s) in a row",
				ConditionResults: []ConditionResult{
					{Condition: "[CONNECTED] == true", Success: false},
					{Condition: "[STATUS] == 200", Success: false},
				},
			},
		},
		{
			Name:     "triggered-with-group",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Endpoint: endpoint.Endpoint{Name: "name", Group: "group", URL: "https://example.org"},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			ExpectedBody: Body{
				Title:            "Gatus",
				EndpointName:     "name",
				EndpointGroup:    "group",
				EndpointURL:      "https://example.org",
				AlertDescription: "description-1",
				Resolved:         false,
				Message:          "An alert for group/name has been triggered due to having failed 3 time(s) in a row",
				ConditionResults: []ConditionResult{
					{Condition: "[CONNECTED] == true", Success: false},
					{Condition: "[STATUS] == 200", Success: false},
				},
			},
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com"}},
			Endpoint: endpoint.Endpoint{Name: "name", URL: "https://example.org"},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			ExpectedBody: Body{
				Title:            "Gatus",
				EndpointName:     "name",
				EndpointURL:      "https://example.org",
				AlertDescription: "description-2",
				Resolved:         true,
				Message:          "An alert for name has been resolved after passing successfully 5 time(s) in a row",
				ConditionResults: []ConditionResult{
					{Condition: "[CONNECTED] == true", Success: true},
					{Condition: "[STATUS] == 200", Success: true},
				},
			},
		},
		{
			Name:     "resolved-with-custom-title",
			Provider: AlertProvider{DefaultConfig: Config{WebhookURL: "http://example.com", Title: "Custom Title"}},
			Endpoint: endpoint.Endpoint{Name: "name", URL: "https://example.org"},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			ExpectedBody: Body{
				Title:            "Custom Title",
				EndpointName:     "name",
				EndpointURL:      "https://example.org",
				AlertDescription: "description-2",
				Resolved:         true,
				Message:          "An alert for name has been resolved after passing successfully 5 time(s) in a row",
				ConditionResults: []ConditionResult{
					{Condition: "[CONNECTED] == true", Success: true},
					{Condition: "[STATUS] == 200", Success: true},
				},
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg, err := scenario.Provider.GetConfig(scenario.Endpoint.Group, &scenario.Alert)
			if err != nil {
				t.Fatal("couldn't get config:", err.Error())
			}
			body := scenario.Provider.buildRequestBody(
				cfg,
				&scenario.Endpoint,
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			var actualBody Body
			if err := json.Unmarshal(body, &actualBody); err != nil {
				t.Error("expected body to be valid JSON, got error:", err.Error())
			}
			if actualBody.Title != scenario.ExpectedBody.Title {
				t.Errorf("expected title to be %s, got %s", scenario.ExpectedBody.Title, actualBody.Title)
			}
			if actualBody.EndpointName != scenario.ExpectedBody.EndpointName {
				t.Errorf("expected endpoint name to be %s, got %s", scenario.ExpectedBody.EndpointName, actualBody.EndpointName)
			}
			if actualBody.Resolved != scenario.ExpectedBody.Resolved {
				t.Errorf("expected resolved to be %v, got %v", scenario.ExpectedBody.Resolved, actualBody.Resolved)
			}
			if actualBody.Message != scenario.ExpectedBody.Message {
				t.Errorf("expected message to be %s, got %s", scenario.ExpectedBody.Message, actualBody.Message)
			}
		})
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

func TestAlertProvider_GetConfig(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		InputAlert     alert.Alert
		ExpectedOutput Config
	}{
		{
			Name: "provider-no-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://example01.com"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://example.com"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://group-example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{WebhookURL: "http://group-example.com"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{WebhookURL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{WebhookURL: "http://group-example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"webhook-url": "http://alert-example.com"}},
			ExpectedOutput: Config{WebhookURL: "http://alert-example.com"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.WebhookURL != scenario.ExpectedOutput.WebhookURL {
				t.Errorf("expected webhook URL to be %s, got %s", scenario.ExpectedOutput.WebhookURL, got.WebhookURL)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

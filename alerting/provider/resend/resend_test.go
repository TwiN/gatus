package resend

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	invalidProvider := AlertProvider{DefaultConfig: Config{APIKey: "", From: "", To: ""}}
	if err := invalidProvider.Validate(); err == nil {
		t.Error("provider shouldn't have been valid")
	}
	validProvider := AlertProvider{DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	providerWithInvalidOverrideGroup := AlertProvider{
		DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
		Overrides: []Override{
			{
				Config: Config{To: "to@example.com"},
				Group:  "",
			},
		},
	}
	if err := providerWithInvalidOverrideGroup.Validate(); !errors.Is(err, ErrDuplicateGroupOverride) {
		t.Errorf("provider with empty Group should return %v, got %v", ErrDuplicateGroupOverride, err)
	}
	providerWithDuplicateOverrideGroups := AlertProvider{
		DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
		Overrides: []Override{
			{
				Config: Config{To: "to1@example.com"},
				Group:  "group",
			},
			{
				Config: Config{To: "to2@example.com"},
				Group:  "group",
			},
		},
	}
	if err := providerWithDuplicateOverrideGroups.Validate(); !errors.Is(err, ErrDuplicateGroupOverride) {
		t.Errorf("provider with duplicate group overrides should return %v, got %v", ErrDuplicateGroupOverride, err)
	}
	providerWithValidOverride := AlertProvider{
		DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
		Overrides: []Override{
			{
				Config: Config{To: "group1@example.com"},
				Group:  "group1",
			},
			{
				Config: Config{To: "group2@example.com"},
				Group:  "group2",
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
			Provider: AlertProvider{DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"49a3999c-0ce1-4ea6-ab68-afcd6dc2e794"}`))}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{"statusCode":401,"name":"missing_api_key","message":"Missing API key in the authorization header"}`))}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"49a3999c-0ce1-4ea6-ab68-afcd6dc2e794"}`))}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "invalid-config",
			Provider: AlertProvider{DefaultConfig: Config{APIKey: "", From: "from@example.com", To: "to@example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				t.Error("no request should have been sent with an invalid configuration")
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
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

func TestAlertProvider_SendRequest(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	provider := AlertProvider{DefaultConfig: Config{APIKey: "re_test", From: "Gatus <alerts@example.com>", To: "to1@example.com, to2@example.com"}}
	var requestSent bool
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		requestSent = true
		if r.Method != http.MethodPost {
			t.Errorf("expected method to be %s, got %s", http.MethodPost, r.Method)
		}
		if r.URL.String() != ApiURL {
			t.Errorf("expected URL to be %s, got %s", ApiURL, r.URL.String())
		}
		if authorization := r.Header.Get("Authorization"); authorization != "Bearer re_test" {
			t.Errorf("expected Authorization header to be %s, got %s", "Bearer re_test", authorization)
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type header to be %s, got %s", "application/json", contentType)
		}
		var body Body
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %s", err)
		}
		expectedBody := Body{
			From:    "Gatus <alerts@example.com>",
			To:      []string{"to1@example.com", "to2@example.com"},
			Subject: "endpoint-name: Alert triggered",
			Text:    "An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row\n\nCondition results:\n❌ [STATUS] == 200\n",
		}
		if !reflect.DeepEqual(body, expectedBody) {
			t.Errorf("expected body to be %+v, got %+v", expectedBody, body)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"49a3999c-0ce1-4ea6-ab68-afcd6dc2e794"}`))}
	})})
	err := provider.Send(
		&endpoint.Endpoint{Name: "endpoint-name"},
		&alert.Alert{SuccessThreshold: 5, FailureThreshold: 3},
		&endpoint.Result{ConditionResults: []*endpoint.ConditionResult{{Condition: "[STATUS] == 200", Success: false}}},
		false,
	)
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if !requestSent {
		t.Error("expected a request to have been sent")
	}
}

func TestAlertProvider_buildRequestBody(t *testing.T) {
	provider := &AlertProvider{}
	cfg := &Config{From: "from@example.com", To: "to1@example.com, to2@example.com,,"}
	body := provider.buildRequestBody(cfg, "Test Subject", "Test Body\nWith new line")
	expectedBody := Body{
		From:    "from@example.com",
		To:      []string{"to1@example.com", "to2@example.com"},
		Subject: "Test Subject",
		Text:    "Test Body\nWith new line",
	}
	if !reflect.DeepEqual(body, expectedBody) {
		t.Errorf("expected body to be %+v, got %+v", expectedBody, body)
	}
}

func TestAlertProvider_buildMessageSubjectAndBody(t *testing.T) {
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name            string
		Provider        AlertProvider
		Alert           alert.Alert
		Resolved        bool
		Endpoint        *endpoint.Endpoint
		ExpectedSubject string
		ExpectedBody    string
	}{
		{
			Name:            "triggered",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        false,
			Endpoint:        &endpoint.Endpoint{Name: "endpoint-name"},
			ExpectedSubject: "endpoint-name: Alert triggered",
			ExpectedBody:    "An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row\n\nAlert description: description-1\n\nCondition results:\n❌ [CONNECTED] == true\n❌ [STATUS] == 200\n",
		},
		{
			Name:            "resolved",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        true,
			Endpoint:        &endpoint.Endpoint{Name: "endpoint-name"},
			ExpectedSubject: "endpoint-name: Alert resolved",
			ExpectedBody:    "An alert for endpoint-name has been resolved after passing successfully 5 time(s) in a row\n\nAlert description: description-2\n\nCondition results:\n✅ [CONNECTED] == true\n✅ [STATUS] == 200\n",
		},
		{
			Name:            "triggered-with-group",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        false,
			Endpoint:        &endpoint.Endpoint{Name: "endpoint-name", Group: "group"},
			ExpectedSubject: "group/endpoint-name: Alert triggered",
			ExpectedBody:    "An alert for group/endpoint-name has been triggered due to having failed 3 time(s) in a row\n\nCondition results:\n❌ [CONNECTED] == true\n❌ [STATUS] == 200\n",
		},
		{
			Name:            "triggered-with-single-extra-label",
			Provider:        AlertProvider{},
			Alert:           alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:        false,
			Endpoint:        &endpoint.Endpoint{Name: "endpoint-name", ExtraLabels: map[string]string{"environment": "production"}},
			ExpectedSubject: "endpoint-name: Alert triggered",
			ExpectedBody:    "An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row\n\nAlert description: description-1\n\nExtra labels:\n  environment: production\n\n\nCondition results:\n❌ [CONNECTED] == true\n❌ [STATUS] == 200\n",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			subject, body := scenario.Provider.buildMessageSubjectAndBody(
				scenario.Endpoint,
				&scenario.Alert,
				&endpoint.Result{
					ConditionResults: []*endpoint.ConditionResult{
						{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
						{Condition: "[STATUS] == 200", Success: scenario.Resolved},
					},
				},
				scenario.Resolved,
			)
			if subject != scenario.ExpectedSubject {
				t.Errorf("expected subject to be %s, got %s", scenario.ExpectedSubject, subject)
			}
			if body != scenario.ExpectedBody {
				t.Errorf("expected body to be %s, got %s", scenario.ExpectedBody, body)
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
				DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "group-to@example.com"},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "group-to@example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{APIKey: "re_test", From: "from@example.com", To: "group-to@example.com"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{To: "group-to@example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"api-key": "re_override", "from": "alert-from@example.com", "to": "alert-to@example.com"}},
			ExpectedOutput: Config{APIKey: "re_override", From: "alert-from@example.com", To: "alert-to@example.com"},
		},
		{
			Name: "provider-partial-override-hierarchy",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "re_default", From: "default@example.com", To: "default@example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{From: "group@example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"to": "alert@example.com"}},
			ExpectedOutput: Config{APIKey: "re_default", From: "group@example.com", To: "alert@example.com"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.APIKey != scenario.ExpectedOutput.APIKey {
				t.Errorf("expected APIKey to be %s, got %s", scenario.ExpectedOutput.APIKey, got.APIKey)
			}
			if got.From != scenario.ExpectedOutput.From {
				t.Errorf("expected From to be %s, got %s", scenario.ExpectedOutput.From, got.From)
			}
			if got.To != scenario.ExpectedOutput.To {
				t.Errorf("expected To to be %s, got %s", scenario.ExpectedOutput.To, got.To)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

func TestAlertProvider_GetConfigWithInvalidAlertOverride(t *testing.T) {
	provider := AlertProvider{DefaultConfig: Config{APIKey: "re_test", From: "from@example.com", To: "to@example.com"}}
	if err := provider.ValidateOverrides("", &alert.Alert{ProviderOverride: map[string]any{"api-key": []string{"invalid"}}}); err == nil {
		t.Error("expected error due to invalid alert override, got none")
	}
}

func TestConfig_Validate(t *testing.T) {
	tooManyRecipients := make([]string, MaximumRecipients+1)
	for i := range tooManyRecipients {
		tooManyRecipients[i] = "to@example.com"
	}
	scenarios := []struct {
		Name          string
		Config        Config
		ExpectedError error
	}{
		{
			Name:          "missing-api-key",
			Config:        Config{APIKey: "", From: "from@example.com", To: "to@example.com"},
			ExpectedError: ErrAPIKeyNotSet,
		},
		{
			Name:          "missing-from",
			Config:        Config{APIKey: "re_test", From: "", To: "to@example.com"},
			ExpectedError: ErrFromNotSet,
		},
		{
			Name:          "missing-to",
			Config:        Config{APIKey: "re_test", From: "from@example.com", To: ""},
			ExpectedError: ErrToNotSet,
		},
		{
			Name:          "to-with-only-separators",
			Config:        Config{APIKey: "re_test", From: "from@example.com", To: " , ,"},
			ExpectedError: ErrToNotSet,
		},
		{
			Name:          "too-many-recipients",
			Config:        Config{APIKey: "re_test", From: "from@example.com", To: strings.Join(tooManyRecipients, ",")},
			ExpectedError: ErrTooManyRecipients,
		},
		{
			Name:          "maximum-recipients",
			Config:        Config{APIKey: "re_test", From: "from@example.com", To: strings.Join(tooManyRecipients[:MaximumRecipients], ",")},
			ExpectedError: nil,
		},
		{
			Name:          "valid-config",
			Config:        Config{APIKey: "re_test", From: "Gatus <from@example.com>", To: "to1@example.com,to2@example.com"},
			ExpectedError: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if err := scenario.Config.Validate(); !errors.Is(err, scenario.ExpectedError) {
				t.Errorf("expected error %v, got %v", scenario.ExpectedError, err)
			}
		})
	}
}

func TestConfig_Merge(t *testing.T) {
	config := Config{APIKey: "re_original", From: "from@example.com", To: "to@example.com", ClientConfig: &client.Config{Timeout: 10000}}
	config.Merge(&Config{APIKey: "re_override", To: "override@example.com"})
	if config.APIKey != "re_override" {
		t.Errorf("expected APIKey to be re_override, got %s", config.APIKey)
	}
	if config.From != "from@example.com" {
		t.Errorf("expected From to remain from@example.com, got %s", config.From)
	}
	if config.To != "override@example.com" {
		t.Errorf("expected To to be override@example.com, got %s", config.To)
	}
	if config.ClientConfig == nil || config.ClientConfig.Timeout != 10000 {
		t.Error("expected ClientConfig to remain unchanged")
	}
	config.Merge(&Config{ClientConfig: &client.Config{Timeout: 30000}})
	if config.ClientConfig.Timeout != 30000 {
		t.Errorf("expected ClientConfig.Timeout to be 30000, got %d", config.ClientConfig.Timeout)
	}
}

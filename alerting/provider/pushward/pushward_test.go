package pushward

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

const testIntegrationKey = "test-integration-key"

func TestAlertProvider_Validate(t *testing.T) {
	t.Parallel()
	invalidProvider := AlertProvider{DefaultConfig: Config{IntegrationKey: ""}}
	if err := invalidProvider.Validate(); !errors.Is(err, ErrIntegrationKeyNotSet) {
		t.Errorf("provider without integration key should return %v, got %v", ErrIntegrationKeyNotSet, err)
	}
	validProvider := AlertProvider{DefaultConfig: Config{IntegrationKey: testIntegrationKey}}
	if err := validProvider.Validate(); err != nil {
		t.Error("provider should've been valid")
	}
	if validProvider.DefaultConfig.Level != "time-sensitive" {
		t.Errorf("expected Level to default to time-sensitive, got %s", validProvider.DefaultConfig.Level)
	}
	if validProvider.DefaultConfig.ResolvedLevel != "active" {
		t.Errorf("expected ResolvedLevel to default to active, got %s", validProvider.DefaultConfig.ResolvedLevel)
	}
}

func TestAlertProvider_ValidateWithOverride(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		Name          string
		Overrides     []Override
		ExpectedError error
	}{
		{
			Name:          "override-without-group",
			Overrides:     []Override{{Group: "", Config: Config{Title: "title"}}},
			ExpectedError: ErrDuplicateGroupOverride,
		},
		{
			Name: "duplicate-group-override",
			Overrides: []Override{
				{Group: "group", Config: Config{Title: "title-1"}},
				{Group: "group", Config: Config{Title: "title-2"}},
			},
			ExpectedError: ErrDuplicateGroupOverride,
		},
		{
			Name:          "override-with-unknown-level",
			Overrides:     []Override{{Group: "group", Config: Config{Level: "urgent"}}},
			ExpectedError: ErrInvalidLevel,
		},
		{
			Name:          "override-with-unknown-resolved-level",
			Overrides:     []Override{{Group: "group", Config: Config{ResolvedLevel: "urgent"}}},
			ExpectedError: ErrInvalidLevel,
		},
		{
			Name: "valid-overrides",
			Overrides: []Override{
				{Group: "group1", Config: Config{Level: "critical"}},
				{Group: "group2", Config: Config{IntegrationKey: "group2-key"}},
			},
			ExpectedError: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			provider := AlertProvider{DefaultConfig: Config{IntegrationKey: testIntegrationKey}, Overrides: scenario.Overrides}
			if err := provider.Validate(); !errors.Is(err, scenario.ExpectedError) {
				t.Errorf("expected error %v, got %v", scenario.ExpectedError, err)
			}
		})
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
			Provider: AlertProvider{DefaultConfig: Config{IntegrationKey: testIntegrationKey}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{IntegrationKey: testIntegrationKey}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{IntegrationKey: testIntegrationKey}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{IntegrationKey: testIntegrationKey}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "invalid-config",
			Provider: AlertProvider{DefaultConfig: Config{IntegrationKey: ""}},
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

func TestAlertProvider_SendLevel(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	scenarios := []struct {
		Name          string
		Config        Config
		Resolved      bool
		ExpectedLevel string
	}{
		{
			Name:          "triggered-default",
			Config:        Config{IntegrationKey: testIntegrationKey},
			Resolved:      false,
			ExpectedLevel: "time-sensitive",
		},
		{
			Name:          "resolved-default",
			Config:        Config{IntegrationKey: testIntegrationKey},
			Resolved:      true,
			ExpectedLevel: "active",
		},
		{
			Name:          "triggered-custom-level",
			Config:        Config{IntegrationKey: testIntegrationKey, Level: "critical", ResolvedLevel: "passive"},
			Resolved:      false,
			ExpectedLevel: "critical",
		},
		{
			Name:          "resolved-custom-resolved-level",
			Config:        Config{IntegrationKey: testIntegrationKey, Level: "critical", ResolvedLevel: "passive"},
			Resolved:      true,
			ExpectedLevel: "passive",
		},
		{
			Name:          "resolved-with-only-level-set-stays-default",
			Config:        Config{IntegrationKey: testIntegrationKey, Level: "critical"},
			Resolved:      true,
			ExpectedLevel: "active",
		},
		{
			Name:          "triggered-with-only-resolved-level-set-stays-default",
			Config:        Config{IntegrationKey: testIntegrationKey, ResolvedLevel: "critical"},
			Resolved:      false,
			ExpectedLevel: "time-sensitive",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var level string
			client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
				var body Body
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Fatalf("failed to decode request body: %s", err)
				}
				level = body.Level
				return &http.Response{StatusCode: http.StatusCreated, Body: http.NoBody}
			})})
			provider := AlertProvider{DefaultConfig: scenario.Config}
			err := provider.Send(&endpoint.Endpoint{Name: "endpoint-name"}, &alert.Alert{SuccessThreshold: 5, FailureThreshold: 3}, &endpoint.Result{}, scenario.Resolved)
			if err != nil {
				t.Error("expected no error, got", err.Error())
			}
			if level != scenario.ExpectedLevel {
				t.Errorf("expected level to be %s, got %s", scenario.ExpectedLevel, level)
			}
		})
	}
}

func TestAlertProvider_SendRequest(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	provider := AlertProvider{DefaultConfig: Config{IntegrationKey: testIntegrationKey}}
	var requestSent bool
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		requestSent = true
		if r.Method != http.MethodPost {
			t.Errorf("expected method to be %s, got %s", http.MethodPost, r.Method)
		}
		if r.URL.String() != "https://api.pushward.app/notifications" {
			t.Errorf("expected URL to be %s, got %s", "https://api.pushward.app/notifications", r.URL.String())
		}
		if authorization := r.Header.Get("Authorization"); authorization != "Bearer "+testIntegrationKey {
			t.Errorf("expected Authorization header to be %s, got %s", "Bearer "+testIntegrationKey, authorization)
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type header to be %s, got %s", "application/json", contentType)
		}
		rawBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %s", err)
		}
		if strings.Contains(r.URL.String(), testIntegrationKey) || strings.Contains(string(rawBody), testIntegrationKey) {
			t.Error("expected integration key to only be in the Authorization header")
		}
		var body Body
		if err := json.Unmarshal(rawBody, &body); err != nil {
			t.Fatalf("failed to decode request body: %s", err)
		}
		expectedBody := Body{
			Title:             "Gatus: endpoint-name",
			Body:              "An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row\n❌ - [STATUS] == 200",
			Level:             "time-sensitive",
			Source:            "gatus",
			SourceDisplayName: "Gatus",
		}
		if !reflect.DeepEqual(body, expectedBody) {
			t.Errorf("expected body to be %+v, got %+v", expectedBody, body)
		}
		return &http.Response{StatusCode: http.StatusCreated, Body: http.NoBody}
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

func TestAlertProvider_SendErrorDoesNotContainIntegrationKey(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	scenarios := []struct {
		Name       string
		StatusCode int
	}{
		{Name: "bad-request", StatusCode: http.StatusBadRequest},
		{Name: "unauthorized", StatusCode: http.StatusUnauthorized},
		{Name: "internal-server-error", StatusCode: http.StatusInternalServerError},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: scenario.StatusCode, Body: io.NopCloser(strings.NewReader(`{"title":"request failed"}`))}
			})})
			provider := AlertProvider{DefaultConfig: Config{IntegrationKey: testIntegrationKey}}
			err := provider.Send(&endpoint.Endpoint{Name: "endpoint-name"}, &alert.Alert{SuccessThreshold: 5, FailureThreshold: 3}, &endpoint.Result{}, false)
			if err == nil {
				t.Fatal("expected error, got none")
			}
			if !strings.Contains(err.Error(), "request failed") {
				t.Errorf("expected error to contain the response body, got %s", err.Error())
			}
			if strings.Contains(err.Error(), testIntegrationKey) {
				t.Errorf("expected error to not contain the integration key, got %s", err.Error())
			}
		})
	}
}

func TestAlertProvider_buildRequestBody(t *testing.T) {
	t.Parallel()
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []struct {
		Name         string
		Provider     AlertProvider
		Endpoint     endpoint.Endpoint
		Alert        alert.Alert
		Resolved     bool
		ExpectedBody string
	}{
		{
			Name:         "triggered",
			Provider:     AlertProvider{DefaultConfig: Config{Level: "time-sensitive", ResolvedLevel: "active"}},
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"title\":\"Gatus: endpoint-name\",\"body\":\"An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row with the following description: description-1\\n❌ - [CONNECTED] == true\\n❌ - [STATUS] == 200\",\"level\":\"time-sensitive\",\"source\":\"gatus\",\"source_display_name\":\"Gatus\"}",
		},
		{
			Name:         "triggered-customtitle",
			Provider:     AlertProvider{DefaultConfig: Config{Title: "Gatus Notifications", Level: "time-sensitive", ResolvedLevel: "active"}},
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name"},
			Alert:        alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"title\":\"Gatus Notifications\",\"body\":\"An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row with the following description: description-1\\n❌ - [CONNECTED] == true\\n❌ - [STATUS] == 200\",\"level\":\"time-sensitive\",\"source\":\"gatus\",\"source_display_name\":\"Gatus\"}",
		},
		{
			Name:         "triggered-with-group",
			Provider:     AlertProvider{DefaultConfig: Config{Level: "time-sensitive", ResolvedLevel: "active"}},
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name", Group: "group"},
			Alert:        alert.Alert{SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     false,
			ExpectedBody: "{\"title\":\"Gatus: group/endpoint-name\",\"body\":\"An alert for group/endpoint-name has been triggered due to having failed 3 time(s) in a row\\n❌ - [CONNECTED] == true\\n❌ - [STATUS] == 200\",\"level\":\"time-sensitive\",\"source\":\"gatus\",\"source_display_name\":\"Gatus\"}",
		},
		{
			Name:         "resolved",
			Provider:     AlertProvider{DefaultConfig: Config{Level: "time-sensitive", ResolvedLevel: "active"}},
			Endpoint:     endpoint.Endpoint{Name: "endpoint-name"},
			Alert:        alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved:     true,
			ExpectedBody: "{\"title\":\"Gatus: endpoint-name\",\"body\":\"An alert for endpoint-name has been resolved after passing successfully 5 time(s) in a row with the following description: description-2\\n✅ - [CONNECTED] == true\\n✅ - [STATUS] == 200\",\"level\":\"active\",\"source\":\"gatus\",\"source_display_name\":\"Gatus\"}",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			body := scenario.Provider.buildRequestBody(
				&scenario.Provider.DefaultConfig,
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
			if string(body) != scenario.ExpectedBody {
				t.Errorf("expected:\n%s\ngot:\n%s", scenario.ExpectedBody, body)
			}
			out := make(map[string]interface{})
			if err := json.Unmarshal(body, &out); err != nil {
				t.Error("expected body to be valid JSON, got error:", err.Error())
			}
		})
	}
}

func TestAlertProvider_buildRequestBodyCaps(t *testing.T) {
	t.Parallel()
	const prefix = "An alert for endpoint-name has been triggered due to having failed 3 time(s) in a row with the following description: "
	const defaultTitle = "Gatus: endpoint-name"
	titleAtLimit := strings.Repeat("✅", maximumTitleLength)
	bodyAtLimit := prefix + strings.Repeat("✅", (maximumEncodedTextSize-len(titleAtLimit)-len(prefix))/len("✅"))
	longNameTitle := "Gatus: " + strings.Repeat("✅", maximumTitleLength-len("Gatus: "))
	scenarios := []struct {
		Name          string
		EndpointName  string
		Title         string
		Description   string
		ExpectedTitle string
		ExpectedBody  string
	}{
		{
			Name:          "at-the-limits",
			EndpointName:  "endpoint-name",
			Title:         titleAtLimit,
			Description:   bodyAtLimit[len(prefix):],
			ExpectedTitle: titleAtLimit,
			ExpectedBody:  bodyAtLimit,
		},
		{
			Name:          "over-the-limits",
			EndpointName:  "endpoint-name",
			Title:         titleAtLimit + "✅",
			Description:   bodyAtLimit[len(prefix):] + "✅",
			ExpectedTitle: titleAtLimit,
			ExpectedBody:  bodyAtLimit,
		},
		{
			Name:          "long-endpoint-name",
			EndpointName:  strings.Repeat("✅", maximumEncodedTextSize),
			ExpectedTitle: longNameTitle,
			ExpectedBody:  "An alert for " + strings.Repeat("✅", (maximumEncodedTextSize-len(longNameTitle)-len("An alert for "))/len("✅")),
		},
		{
			Name:          "characters-that-grow-when-encoded",
			EndpointName:  "endpoint-name",
			Description:   strings.Repeat("<", maximumEncodedTextSize),
			ExpectedTitle: defaultTitle,
			ExpectedBody:  prefix + strings.Repeat("<", (maximumEncodedTextSize-len(defaultTitle)-len(prefix))/encodedSize("<")),
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			provider := AlertProvider{}
			var body Body
			err := json.Unmarshal(provider.buildRequestBody(
				&Config{Title: scenario.Title, Level: "time-sensitive"},
				&endpoint.Endpoint{Name: scenario.EndpointName},
				&alert.Alert{Description: &scenario.Description, SuccessThreshold: 5, FailureThreshold: 3},
				&endpoint.Result{},
				false,
			), &body)
			if err != nil {
				t.Fatal("expected body to be valid JSON, got error:", err.Error())
			}
			if body.Title != scenario.ExpectedTitle {
				t.Errorf("expected title to be %s, got %s", scenario.ExpectedTitle, body.Title)
			}
			if body.Body != scenario.ExpectedBody {
				t.Errorf("expected body to be %s, got %s", scenario.ExpectedBody, body.Body)
			}
			if size := encodedSize(body.Title) + encodedSize(body.Body); size > maximumEncodedTextSize {
				t.Errorf("expected encoded title and body to fit in %d bytes, got %d", maximumEncodedTextSize, size)
			}
		})
	}
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	t.Parallel()
	if (&AlertProvider{DefaultAlert: &alert.Alert{}}).GetDefaultAlert() == nil {
		t.Error("expected default alert to be not nil")
	}
	if (&AlertProvider{DefaultAlert: nil}).GetDefaultAlert() != nil {
		t.Error("expected default alert to be nil")
	}
}

func TestAlertProvider_GetConfig(t *testing.T) {
	t.Parallel()
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
				DefaultConfig: Config{IntegrationKey: "default-key"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{IntegrationKey: "default-key", Level: "time-sensitive", ResolvedLevel: "active"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "default-key"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{IntegrationKey: "default-key", Level: "time-sensitive", ResolvedLevel: "active"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "default-key"},
				Overrides:     []Override{{Group: "group", Config: Config{Level: "critical"}}},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{IntegrationKey: "default-key", Level: "time-sensitive", ResolvedLevel: "active"},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "default-key"},
				Overrides:     []Override{{Group: "group", Config: Config{IntegrationKey: "group-key", Title: "group-title", Level: "critical", ResolvedLevel: "passive"}}},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{IntegrationKey: "group-key", Title: "group-title", Level: "critical", ResolvedLevel: "passive"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "default-key"},
				Overrides:     []Override{{Group: "group", Config: Config{IntegrationKey: "group-key", Title: "group-title", Level: "critical", ResolvedLevel: "passive"}}},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"integration-key": "alert-key", "title": "alert-title", "level": "passive", "resolved-level": "time-sensitive"}},
			ExpectedOutput: Config{IntegrationKey: "alert-key", Title: "alert-title", Level: "passive", ResolvedLevel: "time-sensitive"},
		},
		{
			Name: "provider-partial-override-hierarchy",
			Provider: AlertProvider{
				DefaultConfig: Config{IntegrationKey: "default-key", Title: "default-title"},
				Overrides:     []Override{{Group: "group", Config: Config{Level: "passive"}}},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"resolved-level": "passive"}},
			ExpectedOutput: Config{IntegrationKey: "default-key", Title: "default-title", Level: "passive", ResolvedLevel: "passive"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if *got != scenario.ExpectedOutput {
				t.Errorf("expected config to be %+v, got %+v", scenario.ExpectedOutput, *got)
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

func TestAlertProvider_GetConfigWithInvalidAlertOverride(t *testing.T) {
	t.Parallel()
	provider := AlertProvider{DefaultConfig: Config{IntegrationKey: testIntegrationKey}}
	if err := provider.ValidateOverrides("", &alert.Alert{ProviderOverride: map[string]any{"integration-key": []string{"invalid"}}}); err == nil {
		t.Error("expected error due to invalid alert override, got none")
	}
	if err := provider.ValidateOverrides("", &alert.Alert{ProviderOverride: map[string]any{"level": "urgent"}}); !errors.Is(err, ErrInvalidLevel) {
		t.Errorf("expected error %v, got %v", ErrInvalidLevel, err)
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()
	scenarios := []struct {
		Name          string
		Config        Config
		ExpectedError error
	}{
		{
			Name:          "missing-integration-key",
			Config:        Config{IntegrationKey: ""},
			ExpectedError: ErrIntegrationKeyNotSet,
		},
		{
			Name:          "unknown-level",
			Config:        Config{IntegrationKey: testIntegrationKey, Level: "urgent"},
			ExpectedError: ErrInvalidLevel,
		},
		{
			Name:          "unknown-resolved-level",
			Config:        Config{IntegrationKey: testIntegrationKey, ResolvedLevel: "urgent"},
			ExpectedError: ErrInvalidLevel,
		},
		{
			Name:          "valid-config",
			Config:        Config{IntegrationKey: testIntegrationKey},
			ExpectedError: nil,
		},
		{
			Name:          "valid-config-passive-and-critical",
			Config:        Config{IntegrationKey: testIntegrationKey, Level: "passive", ResolvedLevel: "critical"},
			ExpectedError: nil,
		},
		{
			Name:          "valid-config-active-and-time-sensitive",
			Config:        Config{IntegrationKey: testIntegrationKey, Level: "active", ResolvedLevel: "time-sensitive"},
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
	t.Parallel()
	config := Config{IntegrationKey: "original-key", Title: "original-title", Level: "active", ResolvedLevel: "passive"}
	config.Merge(&Config{})
	if expected := (Config{IntegrationKey: "original-key", Title: "original-title", Level: "active", ResolvedLevel: "passive"}); config != expected {
		t.Errorf("expected config to remain %+v, got %+v", expected, config)
	}
	config.Merge(&Config{IntegrationKey: "override-key", Title: "override-title", Level: "critical", ResolvedLevel: "time-sensitive"})
	if expected := (Config{IntegrationKey: "override-key", Title: "override-title", Level: "critical", ResolvedLevel: "time-sensitive"}); config != expected {
		t.Errorf("expected config to be %+v, got %+v", expected, config)
	}
}

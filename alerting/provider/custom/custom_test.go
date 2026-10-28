package custom

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_Validate(t *testing.T) {
	t.Run("invalid-provider", func(t *testing.T) {
		invalidProvider := AlertProvider{DefaultConfig: Config{URL: ""}}
		if err := invalidProvider.Validate(); err == nil {
			t.Error("provider shouldn't have been valid")
		}
	})
	t.Run("valid-provider", func(t *testing.T) {
		validProvider := AlertProvider{DefaultConfig: Config{URL: "https://example.com"}}
		if err := validProvider.Validate(); err != nil {
			t.Error("provider should've been valid")
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
			Provider: AlertProvider{DefaultConfig: Config{URL: "https://example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "triggered-error",
			Provider: AlertProvider{DefaultConfig: Config{URL: "https://example.com"}},
			Alert:    alert.Alert{Description: &firstDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: false,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusInternalServerError, Body: http.NoBody}
			}),
			ExpectedError: true,
		},
		{
			Name:     "resolved",
			Provider: AlertProvider{DefaultConfig: Config{URL: "https://example.com"}},
			Alert:    alert.Alert{Description: &secondDescription, SuccessThreshold: 5, FailureThreshold: 3},
			Resolved: true,
			MockRoundTripper: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
			}),
			ExpectedError: false,
		},
		{
			Name:     "resolved-error",
			Provider: AlertProvider{DefaultConfig: Config{URL: "https://example.com"}},
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
	alertProvider := &AlertProvider{
		DefaultConfig: Config{
			URL:  "https://example.com/[ENDPOINT_GROUP]/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]&url=[ENDPOINT_URL]",
			Body: "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ENDPOINT_URL],[ALERT_TRIGGERED_OR_RESOLVED]",
		},
	}
	alertDescription := "alert-description"
	scenarios := []struct {
		AlertProvider *AlertProvider
		Resolved      bool
		ExpectedURL   string
		ExpectedBody  string
	}{
		{
			AlertProvider: alertProvider,
			Resolved:      true,
			ExpectedURL:   "https://example.com/endpoint-group/endpoint-name?event=RESOLVED&description=alert-description&url=https%3A%2F%2Fexample.com",
			ExpectedBody:  "endpoint-name,endpoint-group,alert-description,https://example.com,RESOLVED",
		},
		{
			AlertProvider: alertProvider,
			Resolved:      false,
			ExpectedURL:   "https://example.com/endpoint-group/endpoint-name?event=TRIGGERED&description=alert-description&url=https%3A%2F%2Fexample.com",
			ExpectedBody:  "endpoint-name,endpoint-group,alert-description,https://example.com,TRIGGERED",
		},
	}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("resolved-%v-with-default-placeholders", scenario.Resolved), func(t *testing.T) {
			request := alertProvider.buildHTTPRequest(
				&alertProvider.DefaultConfig,
				&endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group", URL: "https://example.com"},
				&alert.Alert{Description: &alertDescription},
				&endpoint.Result{Errors: []string{}},
				scenario.Resolved,
			)
			if request.URL.String() != scenario.ExpectedURL {
				t.Error("expected URL to be", scenario.ExpectedURL, "got", request.URL.String())
			}
			body, _ := io.ReadAll(request.Body)
			if string(body) != scenario.ExpectedBody {
				t.Error("expected body to be", scenario.ExpectedBody, "got", string(body))
			}
		})
	}
}

func TestAlertProviderWithResultErrors_buildHTTPRequest(t *testing.T) {
	alertProvider := &AlertProvider{
		DefaultConfig: Config{
			URL:  "https://example.com/[ENDPOINT_GROUP]/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]&url=[ENDPOINT_URL]&error=[RESULT_ERRORS]",
			Body: "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ENDPOINT_URL],[ALERT_TRIGGERED_OR_RESOLVED],[RESULT_ERRORS]",
		},
	}
	alertDescription := "alert-description"
	scenarios := []struct {
		AlertProvider *AlertProvider
		Resolved      bool
		ExpectedURL   string
		ExpectedBody  string
		Errors        []string
	}{
		{
			AlertProvider: alertProvider,
			Resolved:      true,
			ExpectedURL:   "https://example.com/endpoint-group/endpoint-name?event=RESOLVED&description=alert-description&url=https%3A%2F%2Fexample.com&error=",
			ExpectedBody:  "endpoint-name,endpoint-group,alert-description,https://example.com,RESOLVED,",
		},
		{
			AlertProvider: alertProvider,
			Resolved:      false,
			ExpectedURL:   "https://example.com/endpoint-group/endpoint-name?event=TRIGGERED&description=alert-description&url=https%3A%2F%2Fexample.com&error=error1%2Cerror2",
			ExpectedBody:  "endpoint-name,endpoint-group,alert-description,https://example.com,TRIGGERED,error1,error2",
			Errors:        []string{"error1", "error2"},
		},
		{
			AlertProvider: alertProvider,
			Resolved:      false,
			ExpectedURL:   "https://example.com/endpoint-group/endpoint-name?event=TRIGGERED&description=alert-description&url=https%3A%2F%2Fexample.com&error=test+%22error+with+quotes%22",
			ExpectedBody:  "endpoint-name,endpoint-group,alert-description,https://example.com,TRIGGERED,test \\\"error with quotes\\\"",
			Errors:        []string{"test \"error with quotes\""},
		},
	}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("resolved-%v-with-default-placeholders-and-result-errors", scenario.Resolved), func(t *testing.T) {
			request := alertProvider.buildHTTPRequest(
				&alertProvider.DefaultConfig,
				&endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group", URL: "https://example.com"},
				&alert.Alert{Description: &alertDescription},
				&endpoint.Result{Errors: scenario.Errors},
				scenario.Resolved,
			)
			if request.URL.String() != scenario.ExpectedURL {
				t.Error("expected URL to be", scenario.ExpectedURL, "got", request.URL.String())
			}
			body, _ := io.ReadAll(request.Body)
			if string(body) != scenario.ExpectedBody {
				t.Error("expected body to be", scenario.ExpectedBody, "got", string(body))
			}
		})
	}
}

func TestAlertProvider_buildHTTPRequestWithCustomPlaceholder(t *testing.T) {
	alertProvider := &AlertProvider{
		DefaultConfig: Config{
			URL:     "https://example.com/[ENDPOINT_GROUP]/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
			Body:    "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
			Headers: nil,
			Placeholders: map[string]map[string]string{
				"ALERT_TRIGGERED_OR_RESOLVED": {
					"RESOLVED":  "fixed",
					"TRIGGERED": "boom",
				},
			},
		},
	}
	alertDescription := "alert-description"
	scenarios := []struct {
		AlertProvider *AlertProvider
		Resolved      bool
		ExpectedURL   string
		ExpectedBody  string
	}{
		{
			AlertProvider: alertProvider,
			Resolved:      true,
			ExpectedURL:   "https://example.com/endpoint-group/endpoint-name?event=fixed&description=alert-description",
			ExpectedBody:  "endpoint-name,endpoint-group,alert-description,fixed",
		},
		{
			AlertProvider: alertProvider,
			Resolved:      false,
			ExpectedURL:   "https://example.com/endpoint-group/endpoint-name?event=boom&description=alert-description",
			ExpectedBody:  "endpoint-name,endpoint-group,alert-description,boom",
		},
	}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("resolved-%v-with-custom-placeholders", scenario.Resolved), func(t *testing.T) {
			request := alertProvider.buildHTTPRequest(
				&alertProvider.DefaultConfig,
				&endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
				&alert.Alert{Description: &alertDescription},
				&endpoint.Result{},
				scenario.Resolved,
			)
			if request.URL.String() != scenario.ExpectedURL {
				t.Error("expected URL to be", scenario.ExpectedURL, "got", request.URL.String())
			}
			body, _ := io.ReadAll(request.Body)
			if string(body) != scenario.ExpectedBody {
				t.Error("expected body to be", scenario.ExpectedBody, "got", string(body))
			}
		})
	}
}

func TestAlertProvider_buildHTTPRequestWithCustomPlaceholderAndResultConditions(t *testing.T) {
	alertProvider := &AlertProvider{
		DefaultConfig: Config{
			URL:     "https://example.com/[ENDPOINT_GROUP]/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
			Body:    "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED],[RESULT_CONDITIONS]",
			Headers: nil,
			Placeholders: map[string]map[string]string{
				"ALERT_TRIGGERED_OR_RESOLVED": {
					"RESOLVED":  "fixed",
					"TRIGGERED": "boom",
				},
			},
		},
	}
	alertDescription := "alert-description"
	scenarios := []struct {
		AlertProvider *AlertProvider
		Resolved      bool
		ExpectedURL   string
		ExpectedBody  string
		NoConditions  bool
	}{
		{
			AlertProvider: alertProvider,
			Resolved:      true,
			ExpectedURL:   "https://example.com/endpoint-group/endpoint-name?event=fixed&description=alert-description",
			ExpectedBody:  "endpoint-name,endpoint-group,alert-description,fixed,✅ - `[CONNECTED] == true`, ✅ - `[STATUS] == 200`",
		},
		{
			AlertProvider: alertProvider,
			Resolved:      false,
			ExpectedURL:   "https://example.com/endpoint-group/endpoint-name?event=boom&description=alert-description",
			ExpectedBody:  "endpoint-name,endpoint-group,alert-description,boom,❌ - `[CONNECTED] == true`, ❌ - `[STATUS] == 200`",
		},
	}
	for _, scenario := range scenarios {
		t.Run(fmt.Sprintf("resolved-%v-with-custom-placeholders", scenario.Resolved), func(t *testing.T) {
			var conditionResults []*endpoint.ConditionResult
			if !scenario.NoConditions {
				conditionResults = []*endpoint.ConditionResult{
					{Condition: "[CONNECTED] == true", Success: scenario.Resolved},
					{Condition: "[STATUS] == 200", Success: scenario.Resolved},
				}
			}

			request := alertProvider.buildHTTPRequest(
				&alertProvider.DefaultConfig,
				&endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group"},
				&alert.Alert{Description: &alertDescription},
				&endpoint.Result{ConditionResults: conditionResults},
				scenario.Resolved,
			)
			if request.URL.String() != scenario.ExpectedURL {
				t.Error("expected URL to be", scenario.ExpectedURL, "got", request.URL.String())
			}
			body, _ := io.ReadAll(request.Body)
			if string(body) != scenario.ExpectedBody {
				t.Error("expected body to be", scenario.ExpectedBody, "got", string(body))
			}
		})
	}
}

func TestAlertProvider_GetAlertStatePlaceholderValueDefaults(t *testing.T) {
	alertProvider := &AlertProvider{
		DefaultConfig: Config{
			URL:  "https://example.com/[ENDPOINT_NAME]?event=[ALERT_TRIGGERED_OR_RESOLVED]&description=[ALERT_DESCRIPTION]",
			Body: "[ENDPOINT_NAME],[ENDPOINT_GROUP],[ALERT_DESCRIPTION],[ALERT_TRIGGERED_OR_RESOLVED]",
		},
	}
	if alertProvider.GetAlertStatePlaceholderValue(&alertProvider.DefaultConfig, true) != "RESOLVED" {
		t.Error("expected RESOLVED, got", alertProvider.GetAlertStatePlaceholderValue(&alertProvider.DefaultConfig, true))
	}
	if alertProvider.GetAlertStatePlaceholderValue(&alertProvider.DefaultConfig, false) != "TRIGGERED" {
		t.Error("expected TRIGGERED, got", alertProvider.GetAlertStatePlaceholderValue(&alertProvider.DefaultConfig, false))
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
				DefaultConfig: Config{URL: "http://example.com", Body: "default-body"},
				Overrides:     nil,
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "http://example.com", Body: "default-body"},
		},
		{
			Name: "provider-no-override-specify-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "http://example.com"},
				Overrides:     nil,
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "http://example.com"},
		},
		{
			Name: "provider-with-override-specify-no-group-should-default",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{URL: "http://group-example.com", Headers: map[string]string{"Cache": "true"}},
					},
				},
			},
			InputGroup:     "",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "http://example.com", Headers: map[string]string{"Cache": "true"}},
		},
		{
			Name: "provider-with-override-specify-group-should-override",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "http://example.com", Body: "default-body"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{URL: "http://group-example.com", Body: "group-body"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{},
			ExpectedOutput: Config{URL: "http://group-example.com", Body: "group-body"},
		},
		{
			Name: "provider-with-group-override-and-alert-override--alert-override-should-take-precedence",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{URL: "http://group-example.com"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"url": "http://alert-example.com", "body": "alert-body"}},
			ExpectedOutput: Config{URL: "http://alert-example.com", Body: "alert-body"},
		},
		{
			Name: "provider-with-partial-overrides",
			Provider: AlertProvider{
				DefaultConfig: Config{URL: "http://example.com"},
				Overrides: []Override{
					{
						Group:  "group",
						Config: Config{Method: "POST"},
					},
				},
			},
			InputGroup:     "group",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"body": "alert-body"}},
			ExpectedOutput: Config{URL: "http://example.com", Body: "alert-body", Method: "POST"},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if got.URL != scenario.ExpectedOutput.URL {
				t.Errorf("expected webhook URL to be %s, got %s", scenario.ExpectedOutput.URL, got.URL)
			}
			if got.Body != scenario.ExpectedOutput.Body {
				t.Errorf("expected body to be %s, got %s", scenario.ExpectedOutput.Body, got.Body)
			}
			if got.Headers != nil {
				for key, value := range scenario.ExpectedOutput.Headers {
					if got.Headers[key] != value {
						t.Errorf("expected header %s to be %s, got %s", key, value, got.Headers[key])
					}
				}
			}
			// Test ValidateOverrides as well, since it really just calls GetConfig
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); err != nil {
				t.Errorf("unexpected error: %s", err)
			}
		})
	}
}

// hostileResult evaluates an endpoint whose service answers with a JSON string that tries to break out of the string it
// is spliced into, so that the result is the one a real evaluation produces and not one built by hand.
func hostileResult(t *testing.T) *endpoint.Result {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hostile, _ := json.Marshal(map[string]string{"status": "x\", \"injected\": \"yes\"\nback\\slash"})
		_, _ = w.Write(hostile)
	}))
	t.Cleanup(server.Close)
	ep := &endpoint.Endpoint{Name: "hostile", URL: server.URL, Conditions: []endpoint.Condition{"[BODY].status == healthy"}}
	if err := ep.ValidateAndSetDefaults(); err != nil {
		t.Fatal(err)
	}
	result := ep.EvaluateHealth()
	if result.Success || len(result.ConditionResults) != 1 {
		t.Fatalf("expected a failed result with one condition result, got %+v", result)
	}
	return result
}

func TestAlertProvider_SendEscapesServiceControlledValuesInAJSONBody(t *testing.T) {
	// The result has to be evaluated before the HTTP client is replaced, because the evaluation uses the same client
	result := hostileResult(t)
	defer client.InjectHTTPClient(nil)
	var received string
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		bytes, _ := io.ReadAll(r.Body)
		received = string(bytes)
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	})})
	provider := AlertProvider{DefaultConfig: Config{
		URL:    "https://example.com/hook",
		Method: http.MethodPost,
		Body:   `{"event": "[ALERT_TRIGGERED_OR_RESOLVED]", "conditions": "[RESULT_CONDITIONS]", "errors": "[RESULT_ERRORS]"}`,
	}}
	result.Errors = append(result.Errors, "a \"quoted\" error with a backslash \\ and a\nnewline")
	if err := provider.Send(&endpoint.Endpoint{Name: "hostile"}, &alert.Alert{}, result, false); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(received), &parsed); err != nil {
		t.Fatalf("expected the body to be valid JSON, got %v for %s", err, received)
	}
	if len(parsed) != 3 {
		t.Errorf("expected exactly the 3 fields of the template, got %v", parsed)
	}
	if _, injected := parsed["injected"]; injected {
		t.Error("the monitored service managed to inject a field")
	}
	if !strings.Contains(parsed["conditions"].(string), `"injected"`) {
		t.Errorf("expected the hostile text to be in the conditions field as text, got %q", parsed["conditions"])
	}
	if expected := "a \"quoted\" error with a backslash \\ and a\nnewline"; parsed["errors"] != expected {
		t.Errorf("expected errors to round-trip as %q, got %q", expected, parsed["errors"])
	}
}

func TestAlertProvider_SendEncodesServiceControlledValuesInTheURL(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	var received *http.Request
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		received = r
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	})})
	provider := AlertProvider{DefaultConfig: Config{URL: "https://example.com/hook/[ENDPOINT_NAME]?error=[RESULT_ERRORS]&state=[ALERT_TRIGGERED_OR_RESOLVED]"}}
	result := &endpoint.Result{Errors: []string{"bad\r\nHost: evil.example #fragment&state=RESOLVED"}}
	// A newline in a value used to make the request impossible to build, and the error was ignored
	if err := provider.Send(&endpoint.Endpoint{Name: "a/b c"}, &alert.Alert{}, result, false); err != nil {
		t.Fatal(err)
	}
	if received.URL.Path != "/hook/a/b c" || received.URL.EscapedPath() != "/hook/a%2Fb%20c" {
		t.Errorf("expected the endpoint name to be one escaped path segment, got %s", received.URL.EscapedPath())
	}
	if got := received.URL.Query()["state"]; len(got) != 1 || got[0] != "TRIGGERED" {
		t.Errorf("expected the value not to be able to add a state parameter, got %v", got)
	}
	if got := received.URL.Query().Get("error"); got != "bad\r\nHost: evil.example #fragment&state=RESOLVED" {
		t.Errorf("expected the error to round-trip through the query, got %q", got)
	}
	if received.URL.Fragment != "" {
		t.Errorf("expected no fragment, got %q", received.URL.Fragment)
	}
}

func TestAlertProvider_buildHTTPRequestLeavesTheErrorsOfANonJSONBodyAsTheyAlwaysWere(t *testing.T) {
	provider := &AlertProvider{DefaultConfig: Config{Body: `errors="[RESULT_ERRORS]"`}}
	request := provider.buildHTTPRequest(&provider.DefaultConfig, &endpoint.Endpoint{}, &alert.Alert{}, &endpoint.Result{Errors: []string{`say "hi" \o/`}}, false)
	body, _ := io.ReadAll(request.Body)
	if expected := `errors="say \"hi\" \o/"`; string(body) != expected {
		t.Errorf("expected %s, got %s", expected, body)
	}
}

func buildBody(t *testing.T, cfg Config, ep *endpoint.Endpoint, result *endpoint.Result) (body, requestURL string) {
	t.Helper()
	provider := &AlertProvider{DefaultConfig: cfg}
	request := provider.buildHTTPRequest(&provider.DefaultConfig, ep, &alert.Alert{}, result, false)
	bytes, _ := io.ReadAll(request.Body)
	return string(bytes), request.URL.String()
}

func TestAlertProvider_buildHTTPRequestTrustsTheContentTypeToTellJSONFromText(t *testing.T) {
	scenarios := []struct {
		name         string
		cfg          Config
		expectedBody string
	}{
		{
			name:         "text-body-that-starts-with-a-bracket",
			cfg:          Config{Headers: map[string]string{"content-type": "text/plain"}, Body: "[[RESULT_ERRORS]]"},
			expectedBody: "[a\nb]",
		},
		{
			name:         "no-content-type-and-a-template-that-is-not-json",
			cfg:          Config{Body: "[[RESULT_ERRORS]] is a list of nothing"},
			expectedBody: "[a\nb] is a list of nothing",
		},
		{
			name:         "no-content-type-and-a-template-that-is-json",
			cfg:          Config{Body: `{"errors": "[RESULT_ERRORS]"}`},
			expectedBody: `{"errors": "a\nb"}`,
		},
		{
			name:         "json-content-type-wins-over-the-look-of-the-body",
			cfg:          Config{Headers: map[string]string{"Content-Type": "application/json"}, Body: `"[RESULT_ERRORS]"`},
			expectedBody: `"a\nb"`,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			body, _ := buildBody(t, scenario.cfg, &endpoint.Endpoint{}, &endpoint.Result{Errors: []string{"a\nb"}})
			if body != scenario.expectedBody {
				t.Errorf("expected %q, got %q", scenario.expectedBody, body)
			}
		})
	}
}

func TestAlertProvider_buildHTTPRequestDoesNotExpandAPlaceholderThatIsInAValue(t *testing.T) {
	cfg := Config{URL: "https://example.com/[ENDPOINT_NAME]?e=[RESULT_ERRORS]", Body: `{"name": "[ENDPOINT_NAME]", "errors": "[RESULT_ERRORS]"}`}
	// The name is the text of a placeholder, as the text of a monitored service can be
	ep := &endpoint.Endpoint{Name: "[RESULT_ERRORS]", Group: "[INCIDENT_ID]"}
	body, requestURL := buildBody(t, cfg, ep, &endpoint.Result{Errors: []string{"boom"}})
	if expected := `{"name": "[RESULT_ERRORS]", "errors": "boom"}`; body != expected {
		t.Errorf("expected %s, got %s", expected, body)
	}
	if expected := "https://example.com/%5BRESULT_ERRORS%5D?e=boom"; requestURL != expected {
		t.Errorf("expected %s, got %s", expected, requestURL)
	}
}

func TestAlertProvider_buildHTTPRequestEncodesValuesInAFormBody(t *testing.T) {
	cfg := Config{Method: http.MethodPost, Headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"}, Body: "event=[ALERT_TRIGGERED_OR_RESOLVED]&error=[RESULT_ERRORS]"}
	body, _ := buildBody(t, cfg, &endpoint.Endpoint{}, &endpoint.Result{Errors: []string{"a&admin=true b"}})
	if expected := "event=TRIGGERED&error=a%26admin%3Dtrue+b"; body != expected {
		t.Errorf("expected %s, got %s", expected, body)
	}
}

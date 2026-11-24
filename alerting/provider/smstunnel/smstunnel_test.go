package smstunnel

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
	"gopkg.in/yaml.v3"
)

const (
	testAPIKey = "sk_live_key-that-must-stay-secret"
	testNumber = "+14155550123"
)

func TestAlertProvider_Validate(t *testing.T) {
	scenarios := []struct {
		Name          string
		Provider      AlertProvider
		ExpectedError error
	}{
		{Name: "valid", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: testNumber}}},
		{Name: "valid-with-device-id", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: testNumber, DeviceID: "phone-1"}}},
		{Name: "valid-shortest-number", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "+6834123"}}},
		{Name: "valid-longest-number", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "+123456789012345"}}},
		{Name: "no-api-key", Provider: AlertProvider{DefaultConfig: Config{To: testNumber}}, ExpectedError: ErrAPIKeyNotSet},
		{Name: "no-to", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key"}}, ExpectedError: ErrToNotSet},
		{Name: "to-without-plus", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "14155550123"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-local-format", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "0723456789"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-with-00-prefix", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "0040723456789"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-comma-list", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: testNumber + ",+14155550124"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-semicolon-list", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: testNumber + ";+14155550124"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-space-list", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: testNumber + " +14155550124"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-with-spaces", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "+1 415 555 0123"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-with-letters", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "+1415555ABCD"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-with-country-code-0", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "+0123456789"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-too-short", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "+123456"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-too-long", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: "+1234567890123456"}}, ExpectedError: ErrInvalidTo},
		{Name: "to-with-trailing-newline", Provider: AlertProvider{DefaultConfig: Config{APIKey: "key", To: testNumber + "\n"}}, ExpectedError: ErrInvalidTo},
		{
			Name: "valid-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "key", To: testNumber},
				Overrides:     []Override{{Group: "core", Config: Config{To: "+14155550124"}}},
			},
		},
		{
			Name: "valid-group-override-without-to",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "key", To: testNumber},
				Overrides:     []Override{{Group: "core", Config: Config{DeviceID: "phone-2"}}},
			},
		},
		{
			Name: "group-override-with-invalid-to",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "key", To: testNumber},
				Overrides:     []Override{{Group: "core", Config: Config{To: "0723456789"}}},
			},
			ExpectedError: ErrInvalidTo,
		},
		{
			Name: "group-override-with-list",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "key", To: testNumber},
				Overrides:     []Override{{Group: "core", Config: Config{To: testNumber + ",+14155550124"}}},
			},
			ExpectedError: ErrInvalidTo,
		},
		{
			Name: "duplicate-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "key", To: testNumber},
				Overrides:     []Override{{Group: "core", Config: Config{To: testNumber}}, {Group: "core", Config: Config{To: testNumber}}},
			},
			ExpectedError: ErrDuplicateGroupOverride,
		},
		{
			Name: "group-override-without-group",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "key", To: testNumber},
				Overrides:     []Override{{Config: Config{To: testNumber}}},
			},
			ExpectedError: ErrDuplicateGroupOverride,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if err := scenario.Provider.Validate(); !errors.Is(err, scenario.ExpectedError) {
				t.Errorf("expected error %v, got %v", scenario.ExpectedError, err)
			}
		})
	}
}

func TestAlertProvider_ValidateErrorDoesNotContainTheNumber(t *testing.T) {
	provider := AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: "4915112345678"}}
	err := provider.Validate()
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "4915112345678") || strings.Contains(err.Error(), testAPIKey) {
		t.Errorf("the error shouldn't contain the number or the key, got: %s", err.Error())
	}
}

func TestConfig_ParsedFromYAML(t *testing.T) {
	t.Run("unquoted-number-keeps-its-plus", func(t *testing.T) {
		var provider AlertProvider
		if err := yaml.Unmarshal([]byte("api-key: key\nto: +14155550123\n"), &provider); err != nil {
			t.Fatal("unexpected error:", err)
		}
		if provider.DefaultConfig.To != testNumber {
			t.Errorf("expected to to be %s, got %s", testNumber, provider.DefaultConfig.To)
		}
		if err := provider.Validate(); err != nil {
			t.Error("expected the provider to be valid, got:", err)
		}
	})
	t.Run("list-is-refused", func(t *testing.T) {
		var provider AlertProvider
		err := yaml.Unmarshal([]byte("api-key: key\nto:\n  - \"+14155550123\"\n  - \"+14155550124\"\n"), &provider)
		if err == nil {
			t.Fatal("a list of numbers should've been refused")
		}
		if strings.Contains(err.Error(), "4155550123") {
			t.Errorf("the error shouldn't contain the number, got: %s", err.Error())
		}
	})
	t.Run("list-in-group-override-is-refused", func(t *testing.T) {
		var provider AlertProvider
		err := yaml.Unmarshal([]byte("api-key: key\nto: \"+14155550123\"\noverrides:\n  - group: core\n    to: [\"+14155550123\", \"+14155550124\"]\n"), &provider)
		if err == nil {
			t.Fatal("a list of numbers should've been refused")
		}
	})
}

func TestAlertProvider_Send(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	description := "healthcheck failed"
	scenarios := []struct {
		Name            string
		Provider        AlertProvider
		Alert           alert.Alert
		Resolved        bool
		StatusCode      int
		ExpectedMessage string
		ExpectedTo      string
		ExpectedDevice  string
		ExpectedKey     string
		ExpectedError   bool
	}{
		{
			Name:            "triggered",
			Provider:        AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber}},
			Alert:           alert.Alert{Description: &description, FailureThreshold: 3, SuccessThreshold: 2},
			StatusCode:      http.StatusOK,
			ExpectedMessage: "TRIGGERED: endpoint-name - healthcheck failed",
			ExpectedTo:      testNumber,
			ExpectedKey:     testAPIKey,
		},
		{
			Name:            "resolved",
			Provider:        AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber}},
			Alert:           alert.Alert{Description: &description, FailureThreshold: 3, SuccessThreshold: 2},
			Resolved:        true,
			StatusCode:      http.StatusOK,
			ExpectedMessage: "RESOLVED: endpoint-name - healthcheck failed",
			ExpectedTo:      testNumber,
			ExpectedKey:     testAPIKey,
		},
		{
			Name:            "without-description",
			Provider:        AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber}},
			Alert:           alert.Alert{FailureThreshold: 3, SuccessThreshold: 2},
			StatusCode:      http.StatusOK,
			ExpectedMessage: "TRIGGERED: endpoint-name",
			ExpectedTo:      testNumber,
			ExpectedKey:     testAPIKey,
		},
		{
			Name:            "with-device-id",
			Provider:        AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber, DeviceID: "phone-1"}},
			Alert:           alert.Alert{Description: &description, FailureThreshold: 3, SuccessThreshold: 2},
			StatusCode:      http.StatusOK,
			ExpectedMessage: "TRIGGERED: endpoint-name - healthcheck failed",
			ExpectedTo:      testNumber,
			ExpectedDevice:  "phone-1",
			ExpectedKey:     testAPIKey,
		},
		{
			Name: "with-alert-override",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: testAPIKey, To: testNumber},
			},
			Alert: alert.Alert{
				Description:      &description,
				FailureThreshold: 3,
				SuccessThreshold: 2,
				ProviderOverride: map[string]any{"api-key": "other-key", "to": "+14155550124", "device-id": "phone-2"},
			},
			StatusCode:      http.StatusOK,
			ExpectedMessage: "TRIGGERED: endpoint-name - healthcheck failed",
			ExpectedTo:      "+14155550124",
			ExpectedDevice:  "phone-2",
			ExpectedKey:     "other-key",
		},
		{
			Name:          "error-4xx",
			Provider:      AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber}},
			Alert:         alert.Alert{Description: &description, FailureThreshold: 3, SuccessThreshold: 2},
			StatusCode:    http.StatusTooManyRequests,
			ExpectedError: true,
		},
		{
			Name:          "error-5xx",
			Provider:      AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber}},
			Alert:         alert.Alert{Description: &description, FailureThreshold: 3, SuccessThreshold: 2},
			Resolved:      true,
			StatusCode:    http.StatusInternalServerError,
			ExpectedError: true,
		},
		{
			Name:          "error-redirect",
			Provider:      AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber}},
			Alert:         alert.Alert{Description: &description, FailureThreshold: 3, SuccessThreshold: 2},
			StatusCode:    http.StatusTemporaryRedirect,
			ExpectedError: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			var captured *http.Request
			var capturedBody []byte
			client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
				captured = r
				capturedBody, _ = io.ReadAll(r.Body)
				return &http.Response{StatusCode: scenario.StatusCode, Header: http.Header{}, Body: http.NoBody}
			})})
			err := scenario.Provider.Send(&endpoint.Endpoint{Name: "endpoint-name"}, &scenario.Alert, &endpoint.Result{}, scenario.Resolved)
			if scenario.ExpectedError {
				if err == nil {
					t.Fatal("expected an error, got none")
				}
				return
			}
			if err != nil {
				t.Fatal("expected no error, got:", err)
			}
			if captured.Method != http.MethodPost {
				t.Errorf("expected a POST, got %s", captured.Method)
			}
			if captured.URL.String() != "https://smstunnel.io/api/v1/sms/send" {
				t.Errorf("unexpected URL: %s", captured.URL.String())
			}
			if got := captured.Header.Get("X-API-Key"); got != scenario.ExpectedKey {
				t.Errorf("expected X-API-Key to be %s, got %s", scenario.ExpectedKey, got)
			}
			if got := captured.Header.Get("Content-Type"); got != "application/json" {
				t.Errorf("unexpected Content-Type: %s", got)
			}
			if len(captured.Header.Get("Idempotency-Key")) == 0 {
				t.Error("expected an Idempotency-Key header")
			}
			var body map[string]any
			if err := json.Unmarshal(capturedBody, &body); err != nil {
				t.Fatal("the body isn't valid JSON:", err)
			}
			if body["to"] != scenario.ExpectedTo {
				t.Errorf("expected to to be %s, got %v", scenario.ExpectedTo, body["to"])
			}
			if body["message"] != scenario.ExpectedMessage {
				t.Errorf("expected message to be %q, got %q", scenario.ExpectedMessage, body["message"])
			}
			if deviceID, exists := body["deviceId"]; len(scenario.ExpectedDevice) > 0 && deviceID != scenario.ExpectedDevice {
				t.Errorf("expected deviceId to be %s, got %v", scenario.ExpectedDevice, deviceID)
			} else if len(scenario.ExpectedDevice) == 0 && exists {
				t.Errorf("expected no deviceId, got %v", deviceID)
			}
			expectedFields := 2
			if len(scenario.ExpectedDevice) > 0 {
				expectedFields = 3
			}
			if len(body) != expectedFields {
				t.Errorf("the API refuses unknown fields, expected %d fields, got %v", expectedFields, body)
			}
		})
	}
}

func TestAlertProvider_SendErrorDoesNotContainSecrets(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	provider := AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber}}
	echo := `{"statusCode":400,"message":"Invalid phone number format: ` + testNumber + `, key ` + testAPIKey + `"}`
	scenarios := []struct {
		Name   string
		Header http.Header
	}{
		{Name: "first-answer", Header: http.Header{}},
		{Name: "replayed-answer", Header: http.Header{"Idempotent-Replayed": {"true"}}},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
				return &http.Response{StatusCode: http.StatusBadRequest, Header: scenario.Header, Body: io.NopCloser(strings.NewReader(echo))}
			})})
			err := provider.Send(&endpoint.Endpoint{Name: "endpoint-name"}, &alert.Alert{FailureThreshold: 3}, &endpoint.Result{}, false)
			if err == nil {
				t.Fatal("expected an error")
			}
			if strings.Contains(err.Error(), testNumber) || strings.Contains(err.Error(), testAPIKey) {
				t.Errorf("the error shouldn't contain the number or the key, got: %s", err.Error())
			}
		})
	}
}

func TestAlertProvider_SendTransportErrorDoesNotContainSecrets(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	client.InjectHTTPClient(&http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection reset by peer")
	})})
	provider := AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber}}
	err := provider.Send(&endpoint.Endpoint{Name: "endpoint-name"}, &alert.Alert{FailureThreshold: 3}, &endpoint.Result{}, false)
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), testNumber) || strings.Contains(err.Error(), testAPIKey) {
		t.Errorf("the error shouldn't contain the number or the key, got: %s", err.Error())
	}
}

func TestAlertProvider_SendDoesNotCallTheAPIWithAnInvalidNumber(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	called := false
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		called = true
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	})})
	provider := AlertProvider{DefaultConfig: Config{APIKey: testAPIKey, To: testNumber}}
	for _, to := range []any{"0723456789", 14155550123, []string{testNumber, "+14155550124"}} {
		err := provider.Send(&endpoint.Endpoint{Name: "endpoint-name"}, &alert.Alert{
			FailureThreshold: 3,
			ProviderOverride: map[string]any{"to": to},
		}, &endpoint.Result{}, false)
		if err == nil {
			t.Errorf("expected an error for to=%v", to)
		}
	}
	if called {
		t.Error("the API shouldn't have been called")
	}
}

func TestHTTPClient_DoesNotFollowRedirects(t *testing.T) {
	client.InjectHTTPClient(nil)
	targetCalled := false
	mux := http.NewServeMux()
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/target", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/target", func(w http.ResponseWriter, r *http.Request) {
		targetCalled = true
	})
	server := httptest.NewServer(mux)
	defer server.Close()
	request, err := http.NewRequest(http.MethodPost, server.URL+"/redirect", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("X-API-Key", testAPIKey)
	response, err := client.GetHTTPClient(httpClientConfig).Do(request)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusTemporaryRedirect {
		t.Errorf("expected the redirect to be returned as is, got %d", response.StatusCode)
	}
	if targetCalled {
		t.Error("the redirect shouldn't have been followed")
	}
	if httpClientConfig.Insecure {
		t.Error("TLS verification must stay on")
	}
}

func TestBuildIdempotencyKey(t *testing.T) {
	payload := []byte(`{"to":"+14155550123","message":"TRIGGERED: endpoint-name"}`)
	t.Run("format", func(t *testing.T) {
		key := buildIdempotencyKey(&endpoint.Endpoint{NumberOfFailuresInARow: 3}, &alert.Alert{FailureThreshold: 3}, false, payload)
		if !regexp.MustCompile(`^[\x21-\x7E]{1,255}$`).MatchString(key) {
			t.Errorf("the API takes 1 to 255 printable ASCII characters, got %q", key)
		}
		if strings.Contains(key, "4155550123") {
			t.Errorf("the key shouldn't contain the number, got %s", key)
		}
	})
	t.Run("same-for-a-retry-of-the-first-alert", func(t *testing.T) {
		ep, a := &endpoint.Endpoint{NumberOfFailuresInARow: 3}, &alert.Alert{FailureThreshold: 3}
		first := buildIdempotencyKey(ep, a, false, payload)
		ep.NumberOfFailuresInARow = 4
		if second := buildIdempotencyKey(ep, a, false, payload); second != first {
			t.Error("a retry must use the same key")
		}
		ep.NumberOfFailuresInARow = 50
		if third := buildIdempotencyKey(ep, a, false, payload); third != first {
			t.Error("a later retry must use the same key")
		}
	})
	t.Run("different-for-a-new-incident-even-without-a-resolved-send", func(t *testing.T) {
		ep, a := &endpoint.Endpoint{NumberOfFailuresInARow: 3}, &alert.Alert{FailureThreshold: 3}
		first := buildIdempotencyKey(ep, a, false, payload)
		// With send-on-resolved set to false, Send isn't called on resolve, so a.ResolveKey is still the old incident's.
		// All that's left to tell them apart is that the new incident has just reached the failure threshold.
		if second := buildIdempotencyKey(ep, a, false, payload); second == first {
			t.Error("a new incident must use a new key")
		}
	})
	t.Run("different-for-each-endpoint-alert", func(t *testing.T) {
		ep := &endpoint.Endpoint{NumberOfFailuresInARow: 3}
		first := buildIdempotencyKey(ep, &alert.Alert{FailureThreshold: 3}, false, payload)
		if second := buildIdempotencyKey(ep, &alert.Alert{FailureThreshold: 3}, false, payload); second == first {
			t.Error("two alerts must not share a key")
		}
	})
	t.Run("reminders", func(t *testing.T) {
		lastSent := time.Now().Add(-time.Hour)
		ep, a := &endpoint.Endpoint{NumberOfFailuresInARow: 3, LastReminderSent: lastSent}, &alert.Alert{FailureThreshold: 3}
		initial := buildIdempotencyKey(ep, a, false, payload)
		a.Triggered = true
		ep.NumberOfFailuresInARow = 10
		reminder := buildIdempotencyKey(ep, a, false, payload)
		if reminder == initial {
			t.Error("a reminder must not reuse the key of the first alert")
		}
		ep.NumberOfFailuresInARow = 11
		if retried := buildIdempotencyKey(ep, a, false, payload); retried != reminder {
			t.Error("a retry of a reminder must use the same key")
		}
		ep.LastReminderSent = time.Now()
		if next := buildIdempotencyKey(ep, a, false, payload); next == reminder {
			t.Error("the next reminder must use a new key")
		}
	})
	t.Run("resolved-is-a-different-text", func(t *testing.T) {
		ep, a := &endpoint.Endpoint{NumberOfFailuresInARow: 3}, &alert.Alert{FailureThreshold: 3}
		triggered := buildIdempotencyKey(ep, a, false, payload)
		a.Triggered = true
		if resolved := buildIdempotencyKey(ep, a, true, payload); resolved == triggered {
			t.Error("the resolved text must use its own key")
		}
	})
	t.Run("resolved-keeps-the-incident", func(t *testing.T) {
		ep, a := &endpoint.Endpoint{NumberOfFailuresInARow: 3}, &alert.Alert{FailureThreshold: 3}
		buildIdempotencyKey(ep, a, false, payload)
		incident := a.ResolveKey
		a.Triggered = false // handleAlertsToResolve clears it before calling Send
		buildIdempotencyKey(ep, a, true, payload)
		if a.ResolveKey != incident {
			t.Error("the resolved text must not start a new incident")
		}
	})
	t.Run("different-when-the-request-changes", func(t *testing.T) {
		ep, a := &endpoint.Endpoint{NumberOfFailuresInARow: 3}, &alert.Alert{FailureThreshold: 3}
		first := buildIdempotencyKey(ep, a, false, payload)
		ep.NumberOfFailuresInARow = 4
		if second := buildIdempotencyKey(ep, a, false, []byte(`{"to":"+14155550124","message":"TRIGGERED: endpoint-name"}`)); second == first {
			t.Error("a different request must use a different key, or the API would refuse it")
		}
	})
}

func TestAlertProvider_GetConfig(t *testing.T) {
	scenarios := []struct {
		Name           string
		Provider       AlertProvider
		InputGroup     string
		InputAlert     alert.Alert
		ExpectedOutput Config
		ExpectedError  error
	}{
		{
			Name:           "provider-no-override-should-default",
			Provider:       AlertProvider{DefaultConfig: Config{APIKey: "1", To: testNumber, DeviceID: "phone-1"}},
			ExpectedOutput: Config{APIKey: "1", To: testNumber, DeviceID: "phone-1"},
		},
		{
			Name: "provider-with-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "1", To: testNumber},
				Overrides:     []Override{{Group: "core", Config: Config{APIKey: "2", To: "+14155550124", DeviceID: "phone-2"}}},
			},
			InputGroup:     "core",
			ExpectedOutput: Config{APIKey: "2", To: "+14155550124", DeviceID: "phone-2"},
		},
		{
			Name: "provider-with-group-override-for-another-group",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "1", To: testNumber},
				Overrides:     []Override{{Group: "core", Config: Config{To: "+14155550124"}}},
			},
			InputGroup:     "other",
			ExpectedOutput: Config{APIKey: "1", To: testNumber},
		},
		{
			Name: "alert-override-wins-over-group-override",
			Provider: AlertProvider{
				DefaultConfig: Config{APIKey: "1", To: testNumber},
				Overrides:     []Override{{Group: "core", Config: Config{To: "+14155550124"}}},
			},
			InputGroup:     "core",
			InputAlert:     alert.Alert{ProviderOverride: map[string]any{"to": "+14155550125"}},
			ExpectedOutput: Config{APIKey: "1", To: "+14155550125"},
		},
		{
			Name:          "alert-override-with-invalid-to",
			Provider:      AlertProvider{DefaultConfig: Config{APIKey: "1", To: testNumber}},
			InputAlert:    alert.Alert{ProviderOverride: map[string]any{"to": "0723456789"}},
			ExpectedError: ErrInvalidTo,
		},
		{
			Name:     "alert-override-with-unquoted-number-is-refused-rather-than-guessed",
			Provider: AlertProvider{DefaultConfig: Config{APIKey: "1", To: testNumber}},
			// This is what YAML gives for an unquoted +14155550123 in a provider-override: an int, without the plus
			InputAlert:    alert.Alert{ProviderOverride: map[string]any{"to": 14155550123}},
			ExpectedError: ErrInvalidTo,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			got, err := scenario.Provider.GetConfig(scenario.InputGroup, &scenario.InputAlert)
			if !errors.Is(err, scenario.ExpectedError) {
				t.Fatalf("expected error %v, got %v", scenario.ExpectedError, err)
			}
			if err = scenario.Provider.ValidateOverrides(scenario.InputGroup, &scenario.InputAlert); !errors.Is(err, scenario.ExpectedError) {
				t.Errorf("ValidateOverrides: expected error %v, got %v", scenario.ExpectedError, err)
			}
			if scenario.ExpectedError != nil {
				return
			}
			if *got != scenario.ExpectedOutput {
				t.Errorf("expected %+v, got %+v", scenario.ExpectedOutput, *got)
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

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

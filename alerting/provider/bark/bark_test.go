package bark

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		config        Config
		expectedURL   string
		expectedError error
	}{
		{
			name:        "minimal configuration uses default server URL",
			config:      Config{DeviceKey: "device-key"},
			expectedURL: DefaultServerURL,
		},
		{
			name:          "device key is required",
			config:        Config{},
			expectedError: ErrDeviceKeyNotSet,
		},
		{
			name:          "server URL requires HTTP or HTTPS",
			config:        Config{ServerURL: "ftp://example.com", DeviceKey: "device-key"},
			expectedError: ErrInvalidServerURL,
		},
		{
			name:          "server URL requires a host",
			config:        Config{ServerURL: "https:///push", DeviceKey: "device-key"},
			expectedError: ErrInvalidServerURL,
		},
		{
			name:          "server URL rejects query",
			config:        Config{ServerURL: "https://example.com?key=value", DeviceKey: "device-key"},
			expectedError: ErrInvalidServerURL,
		},
		{
			name:          "server URL rejects empty query",
			config:        Config{ServerURL: "https://example.com?", DeviceKey: "device-key"},
			expectedError: ErrInvalidServerURL,
		},
		{
			name:          "server URL rejects fragment",
			config:        Config{ServerURL: "https://example.com/#fragment", DeviceKey: "device-key"},
			expectedError: ErrInvalidServerURL,
		},
		{
			name:          "server URL rejects empty fragment",
			config:        Config{ServerURL: "https://example.com#", DeviceKey: "device-key"},
			expectedError: ErrInvalidServerURL,
		},
		{
			name:        "server URL is normalized",
			config:      Config{ServerURL: "https://example.com/bark///", DeviceKey: "device-key"},
			expectedURL: "https://example.com/bark",
		},
		{
			name:        "critical level is valid",
			config:      Config{DeviceKey: "device-key", Level: "critical"},
			expectedURL: DefaultServerURL,
		},
		{
			name:        "active level is valid",
			config:      Config{DeviceKey: "device-key", Level: "active"},
			expectedURL: DefaultServerURL,
		},
		{
			name:        "timeSensitive level is valid",
			config:      Config{DeviceKey: "device-key", Level: "timeSensitive"},
			expectedURL: DefaultServerURL,
		},
		{
			name:        "passive level is valid",
			config:      Config{DeviceKey: "device-key", Level: "passive"},
			expectedURL: DefaultServerURL,
		},
		{
			name:          "unknown level is invalid",
			config:        Config{DeviceKey: "device-key", Level: "urgent"},
			expectedError: ErrInvalidLevel,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			err := test.config.Validate()
			if !errors.Is(err, test.expectedError) {
				t.Fatalf("expected error %v, got %v", test.expectedError, err)
			}
			if test.expectedURL != "" && test.config.ServerURL != test.expectedURL {
				t.Errorf("expected server URL %q, got %q", test.expectedURL, test.config.ServerURL)
			}
		})
	}
}

func TestAlertProviderValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		provider      AlertProvider
		expectedError error
	}{
		{
			name:     "valid default and partial override",
			provider: AlertProvider{DefaultConfig: Config{DeviceKey: "device-key"}, Overrides: []Override{{Group: "production", Config: Config{Level: "critical"}}}},
		},
		{
			name:          "override group is required",
			provider:      AlertProvider{DefaultConfig: Config{DeviceKey: "device-key"}, Overrides: []Override{{Config: Config{Level: "critical"}}}},
			expectedError: ErrGroupOverrideWithoutGroup,
		},
		{
			name:          "override group must be unique",
			provider:      AlertProvider{DefaultConfig: Config{DeviceKey: "device-key"}, Overrides: []Override{{Group: "production"}, {Group: "production"}}},
			expectedError: ErrDuplicateGroupOverride,
		},
		{
			name:          "override server URL is validated",
			provider:      AlertProvider{DefaultConfig: Config{DeviceKey: "device-key"}, Overrides: []Override{{Group: "production", Config: Config{ServerURL: "mailto:invalid"}}}},
			expectedError: ErrInvalidServerURL,
		},
		{
			name:          "override level is validated",
			provider:      AlertProvider{DefaultConfig: Config{DeviceKey: "device-key"}, Overrides: []Override{{Group: "production", Config: Config{Level: "urgent"}}}},
			expectedError: ErrInvalidLevel,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := test.provider.Validate(); !errors.Is(err, test.expectedError) {
				t.Fatalf("expected error %v, got %v", test.expectedError, err)
			}
		})
	}
}

func TestOverrideYAMLRoundTrip(t *testing.T) {
	t.Parallel()

	original := AlertProvider{
		DefaultConfig: Config{DeviceKey: "device-key", Group: "default-notification-group"},
		Overrides: []Override{{
			Group: "production",
			Config: Config{
				Title: "Production",
				Group: "must-not-shadow-selector",
				Level: "critical",
			},
		}},
	}
	encoded, err := yaml.Marshal(original)
	if err != nil {
		t.Fatal("expected Bark override to marshal, got", err)
	}
	var decoded AlertProvider
	if err := yaml.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal("expected marshaled Bark override to unmarshal, got", err)
	}
	if len(decoded.Overrides) != 1 {
		t.Fatalf("expected one Bark override, got %d", len(decoded.Overrides))
	}
	if decoded.DefaultConfig.Group != "default-notification-group" {
		t.Errorf("expected default Bark notification group to survive YAML round trip, got %q", decoded.DefaultConfig.Group)
	}
	actual := decoded.Overrides[0]
	if actual.Group != "production" || actual.Title != "Production" || actual.Level != "critical" {
		t.Errorf("unexpected Bark override after YAML round trip: %#v", actual)
	}
	if actual.Config.Group != "" {
		t.Errorf("expected override group to remain an endpoint selector, got Bark notification group %q", actual.Config.Group)
	}
}

func TestAlertProviderGetConfig(t *testing.T) {
	t.Parallel()

	provider := AlertProvider{
		DefaultConfig: Config{
			ServerURL: "https://default.example.com",
			DeviceKey: "default-device-key",
			Title:     "Default title",
			Group:     "default-group",
			Sound:     "default-sound",
			Icon:      "https://default.example.com/icon.png",
			URL:       "https://default.example.com/details",
			Level:     "active",
		},
		Overrides: []Override{
			{
				Group: "production",
				Config: Config{
					ServerURL: "https://group.example.com/",
					DeviceKey: "group-device-key",
					Title:     "Group title",
					Sound:     "group-sound",
					Icon:      "https://group.example.com/icon.png",
					URL:       "https://group.example.com/details",
					Level:     "timeSensitive",
				},
			},
		},
	}
	alertConfig := &alert.Alert{
		ProviderOverride: map[string]any{
			"device-key": "alert-device-key",
			"title":      "Alert title",
			"group":      "alert-group",
			"sound":      "alert-sound",
			"level":      "critical",
		},
	}

	actual, err := provider.GetConfig("production", alertConfig)
	if err != nil {
		t.Fatal("expected configuration to be valid, got", err)
	}
	expected := &Config{
		ServerURL: "https://group.example.com",
		DeviceKey: "alert-device-key",
		Title:     "Alert title",
		Group:     "alert-group",
		Sound:     "alert-sound",
		Icon:      "https://group.example.com/icon.png",
		URL:       "https://group.example.com/details",
		Level:     "critical",
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("expected configuration %#v, got %#v", expected, actual)
	}
	withoutAlertOverride, err := provider.GetConfig("production", &alert.Alert{})
	if err != nil {
		t.Fatal("expected group configuration to be valid, got", err)
	}
	if withoutAlertOverride.Group != "default-group" {
		t.Errorf("expected group override to inherit Bark notification group, got %q", withoutAlertOverride.Group)
	}
}

func TestAlertProviderGetConfigValidatesMergedConfiguration(t *testing.T) {
	t.Parallel()

	provider := AlertProvider{DefaultConfig: Config{DeviceKey: "device-key"}}
	alertConfig := &alert.Alert{ProviderOverride: map[string]any{"level": "urgent"}}

	_, err := provider.GetConfig("", alertConfig)
	if !errors.Is(err, ErrInvalidLevel) {
		t.Fatalf("expected error %v, got %v", ErrInvalidLevel, err)
	}
}

func TestAlertProviderGetConfigRejectsInvalidProviderOverrideType(t *testing.T) {
	t.Parallel()

	provider := AlertProvider{DefaultConfig: Config{DeviceKey: "device-key"}}
	alertConfig := &alert.Alert{ProviderOverride: map[string]any{"server-url": []string{"https://example.com"}}}

	if _, err := provider.GetConfig("", alertConfig); err == nil {
		t.Fatal("expected invalid provider override type to return an error")
	}
}

func TestAlertProviderGetDefaultAlert(t *testing.T) {
	t.Parallel()

	defaultAlert := &alert.Alert{FailureThreshold: 4}
	if actual := (&AlertProvider{DefaultAlert: defaultAlert}).GetDefaultAlert(); actual != defaultAlert {
		t.Error("expected configured default alert")
	}
	if actual := (&AlertProvider{}).GetDefaultAlert(); actual != nil {
		t.Error("expected nil default alert")
	}
}

func TestAlertProviderBuildRequestBody(t *testing.T) {
	t.Parallel()

	description := "degraded \"edge\""
	tests := []struct {
		name             string
		config           Config
		endpoint         endpoint.Endpoint
		alert            alert.Alert
		result           endpoint.Result
		resolved         bool
		expectedTitle    string
		expectedBody     string
		expectedOptional map[string]string
	}{
		{
			name:     "triggered alert uses threshold description and actual condition results",
			config:   Config{DeviceKey: "device-key"},
			endpoint: endpoint.Endpoint{Name: "api", Group: "production"},
			alert:    alert.Alert{FailureThreshold: 3, SuccessThreshold: 2, Description: &description},
			result: endpoint.Result{ConditionResults: []*endpoint.ConditionResult{
				{Condition: "[STATUS] == 200", Success: false},
				{Condition: "[BODY].status == \"UP\"", Success: true},
			}},
			expectedTitle: "Gatus: production/api",
			expectedBody:  "An alert has been triggered due to having failed 3 time(s) in a row with the following description: degraded \"edge\"\n🔴 [STATUS] == 200\n🟢 [BODY].status == \"UP\"",
		},
		{
			name:          "reminder uses the unresolved alert message",
			config:        Config{DeviceKey: "device-key"},
			endpoint:      endpoint.Endpoint{Name: "api"},
			alert:         alert.Alert{FailureThreshold: 5, SuccessThreshold: 2},
			expectedTitle: "Gatus: api",
			expectedBody:  "An alert has been triggered due to having failed 5 time(s) in a row",
		},
		{
			name:          "resolved alert uses success threshold and custom title",
			config:        Config{DeviceKey: "device-key", Title: "Service recovered"},
			endpoint:      endpoint.Endpoint{Name: "api"},
			alert:         alert.Alert{FailureThreshold: 3, SuccessThreshold: 2},
			resolved:      true,
			expectedTitle: "Service recovered",
			expectedBody:  "An alert has been resolved after passing successfully 2 time(s) in a row",
		},
		{
			name:          "optional Bark fields are included",
			config:        Config{DeviceKey: "device-key", Group: "gatus", Sound: "alarm", Icon: "https://example.com/icon.png", URL: "https://example.com/incidents/1", Level: "timeSensitive"},
			endpoint:      endpoint.Endpoint{Name: "api"},
			alert:         alert.Alert{FailureThreshold: 4},
			expectedTitle: "Gatus: api",
			expectedBody:  "An alert has been triggered due to having failed 4 time(s) in a row",
			expectedOptional: map[string]string{
				"group": "gatus",
				"sound": "alarm",
				"icon":  "https://example.com/icon.png",
				"url":   "https://example.com/incidents/1",
				"level": "timeSensitive",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			body := (&AlertProvider{}).buildRequestBody(&test.config, &test.endpoint, &test.alert, &test.result, test.resolved)
			var decoded map[string]string
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatal("expected valid JSON, got", err)
			}
			if decoded["device_key"] != "device-key" {
				t.Errorf("expected device key in JSON payload")
			}
			if decoded["title"] != test.expectedTitle {
				t.Errorf("expected title %q, got %q", test.expectedTitle, decoded["title"])
			}
			if decoded["body"] != test.expectedBody {
				t.Errorf("expected body %q, got %q", test.expectedBody, decoded["body"])
			}
			for _, field := range []string{"group", "sound", "icon", "url", "level"} {
				expectedValue, expected := test.expectedOptional[field]
				actualValue, present := decoded[field]
				if expected && (!present || actualValue != expectedValue) {
					t.Errorf("expected %s %q, got %q", field, expectedValue, actualValue)
				}
				if !expected && present {
					t.Errorf("expected empty %s to be omitted", field)
				}
			}
		})
	}
}

func TestAlertProviderSend(t *testing.T) {
	description := "database unavailable"
	var received Body
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", request.Method)
		}
		if request.URL.Path != "/push" {
			t.Errorf("expected path /push, got %s", request.URL.Path)
		}
		if request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content type, got %s", request.Header.Get("Content-Type"))
		}
		if strings.Contains(request.URL.String(), "secret-device-key") {
			t.Error("device key must not be included in the request URL")
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Error("expected a valid JSON request body, got", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	provider := AlertProvider{DefaultConfig: Config{ServerURL: server.URL, DeviceKey: "secret-device-key", Group: "gatus"}}
	err := provider.Send(
		&endpoint.Endpoint{Name: "database"},
		&alert.Alert{FailureThreshold: 3, Description: &description},
		&endpoint.Result{ConditionResults: []*endpoint.ConditionResult{{Condition: "[STATUS] == 200", Success: false}}},
		false,
	)
	if err != nil {
		t.Fatal("expected request to succeed, got", err)
	}
	if received.DeviceKey != "secret-device-key" || received.Title != "Gatus: database" || received.Group != "gatus" {
		t.Errorf("unexpected Bark payload: %#v", received)
	}
}

func TestAlertProviderSendBuildsPushURL(t *testing.T) {
	tests := []struct {
		name         string
		serverSuffix string
		expectedPath string
	}{
		{name: "trailing slash", serverSuffix: "/", expectedPath: "/push"},
		{name: "existing push path", serverSuffix: "/push", expectedPath: "/push"},
		{name: "self-hosted path prefix", serverSuffix: "/bark", expectedPath: "/bark/push"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var actualPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				actualPath = request.URL.Path
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			provider := AlertProvider{DefaultConfig: Config{ServerURL: server.URL + test.serverSuffix, DeviceKey: "device-key"}}
			if err := provider.Send(&endpoint.Endpoint{Name: "api"}, &alert.Alert{FailureThreshold: 1}, &endpoint.Result{}, false); err != nil {
				t.Fatal("expected request to succeed, got", err)
			}
			if actualPath != test.expectedPath {
				t.Errorf("expected path %q, got %q", test.expectedPath, actualPath)
			}
		})
	}
}

func TestAlertProviderSendReturnsSafeResponseErrors(t *testing.T) {
	tests := []struct {
		status       int
		body         string
		expectedBody string
	}{
		{status: http.StatusBadRequest, body: "invalid request", expectedBody: "invalid request"},
		{status: http.StatusInternalServerError, body: "server unavailable", expectedBody: "server unavailable"},
		{status: http.StatusBadRequest, body: "invalid device secret-device-key", expectedBody: "invalid device [REDACTED]"},
		{status: http.StatusBadRequest, body: "invalid\nsecret-device-key", expectedBody: "invalid [REDACTED]"},
		{status: http.StatusBadRequest, body: "invalid\u2028secret-device-key\u2029response", expectedBody: "invalid [REDACTED] response"},
	}

	for _, test := range tests {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			provider := AlertProvider{DefaultConfig: Config{ServerURL: server.URL, DeviceKey: "secret-device-key", Title: "secret request title"}}
			err := provider.Send(&endpoint.Endpoint{Name: "api"}, &alert.Alert{FailureThreshold: 1}, &endpoint.Result{}, false)
			if err == nil {
				t.Fatal("expected an error response")
			}
			errorText := err.Error()
			if !strings.Contains(errorText, "Bark") || !strings.Contains(errorText, strconv.Itoa(test.status)) || !strings.Contains(errorText, http.StatusText(test.status)) || !strings.Contains(errorText, test.expectedBody) {
				t.Errorf("expected provider, status, and response body in error, got %q", errorText)
			}
			if strings.Contains(errorText, "secret-device-key") || strings.Contains(errorText, "secret request title") {
				t.Errorf("request credentials or body leaked in error: %q", errorText)
			}
		})
	}
}

func TestAlertProviderSendLimitsErrorResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, strings.Repeat("x", 64*1024+100))
	}))
	defer server.Close()

	provider := AlertProvider{DefaultConfig: Config{ServerURL: server.URL, DeviceKey: "device-key"}}
	err := provider.Send(&endpoint.Endpoint{Name: "api"}, &alert.Alert{FailureThreshold: 1}, &endpoint.Result{}, false)
	if err == nil {
		t.Fatal("expected an error response")
	}
	if len(err.Error()) > 65*1024 {
		t.Errorf("expected bounded error response, got %d bytes", len(err.Error()))
	}
	if !strings.Contains(err.Error(), "[truncated]") {
		t.Errorf("expected truncation marker, got %q", err.Error())
	}
}

func TestAlertProviderSendRejectsCrossOriginRedirect(t *testing.T) {
	targetReceivedRequest := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetReceivedRequest = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		http.Redirect(w, request, target.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()

	provider := AlertProvider{DefaultConfig: Config{ServerURL: redirect.URL, DeviceKey: "secret-device-key"}}
	err := provider.Send(&endpoint.Endpoint{Name: "api"}, &alert.Alert{FailureThreshold: 1}, &endpoint.Result{}, false)
	if err == nil {
		t.Fatal("expected cross-origin redirect to fail")
	}
	if targetReceivedRequest {
		t.Error("expected request body not to be forwarded to another origin")
	}
	if strings.Contains(err.Error(), "secret-device-key") {
		t.Errorf("device key leaked in redirect error: %q", err.Error())
	}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (body *trackingReadCloser) Close() error {
	body.closed = true
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (roundTrip roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return roundTrip(request)
}

func TestAlertProviderSendClosesResponseBody(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader("ok")}
	client.InjectHTTPClient(&http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: body, Header: make(http.Header)}, nil
	})})
	t.Cleanup(func() { client.InjectHTTPClient(nil) })

	provider := AlertProvider{DefaultConfig: Config{ServerURL: "https://example.com", DeviceKey: "device-key"}}
	if err := provider.Send(&endpoint.Endpoint{Name: "api"}, &alert.Alert{FailureThreshold: 1}, &endpoint.Result{}, false); err != nil {
		t.Fatal("expected request to succeed, got", err)
	}
	if !body.closed {
		t.Error("expected response body to be closed")
	}
}

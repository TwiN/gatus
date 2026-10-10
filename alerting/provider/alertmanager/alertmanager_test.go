package alertmanager

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestAlertProvider_Validate(t *testing.T) {
	tests := []struct {
		name          string
		provider      AlertProvider
		expectedError bool
	}{
		{
			name: "valid configuration with single URL",
			provider: AlertProvider{
				DefaultConfig: Config{
					URLs: []string{"http://alertmanager:9093"},
				},
			},
			expectedError: false,
		},
		{
			name: "valid configuration with multiple URLs",
			provider: AlertProvider{
				DefaultConfig: Config{
					URLs: []string{"http://alertmanager1:9093", "http://alertmanager2:9093"},
				},
			},
			expectedError: false,
		},
		{
			name: "missing URLs",
			provider: AlertProvider{
				DefaultConfig: Config{},
			},
			expectedError: true,
		},
		{
			name: "empty URLs slice",
			provider: AlertProvider{
				DefaultConfig: Config{
					URLs: []string{},
				},
			},
			expectedError: true,
		},
		{
			name: "valid configuration with overrides",
			provider: AlertProvider{
				DefaultConfig: Config{
					URLs: []string{"http://alertmanager:9093"},
				},
				Overrides: []Override{
					{Group: "group1", Config: Config{URLs: []string{"http://alertmanager1:9093"}}},
					{Group: "group2", Config: Config{URLs: []string{"http://alertmanager2:9093"}}},
				},
			},
			expectedError: false,
		},
		{
			name: "duplicate group override",
			provider: AlertProvider{
				DefaultConfig: Config{
					URLs: []string{"http://alertmanager:9093"},
				},
				Overrides: []Override{
					{Group: "group1", Config: Config{URLs: []string{"http://alertmanager1:9093"}}},
					{Group: "group1", Config: Config{URLs: []string{"http://alertmanager2:9093"}}},
				},
			},
			expectedError: true,
		},
		{
			name: "empty group override",
			provider: AlertProvider{
				DefaultConfig: Config{
					URLs: []string{"http://alertmanager:9093"},
				},
				Overrides: []Override{
					{Group: "", Config: Config{URLs: []string{"http://alertmanager1:9093"}}},
				},
			},
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.provider.Validate()
			if test.expectedError && err == nil {
				t.Error("expected an error, but got none")
			}
			if !test.expectedError && err != nil {
				t.Errorf("expected no error, but got %v", err)
			}
		})
	}
}

func TestAlertProvider_buildAlert(t *testing.T) {
	provider := AlertProvider{
		DefaultConfig: Config{
			URLs:            []string{"http://alertmanager:9093"},
			DefaultSeverity: "warning",
			ExtraLabels: map[string]string{
				"environment": "test",
			},
			ExtraAnnotations: map[string]string{
				"runbook": "https://wiki.example.com/runbook",
			},
		},
	}

	ep := &endpoint.Endpoint{
		Name:  "Test API",
		URL:   "https://api.example.com/health",
		Group: "production",
	}

	testAlert := &alert.Alert{
		Description:      stringPtr("API health check failed"),
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}

	result := &endpoint.Result{
		ConditionResults: []*endpoint.ConditionResult{
			{Condition: "[STATUS] == 200", Success: false},
			{Condition: "[CONNECTED] == true", Success: true},
		},
	}

	// Test firing alert
	firingAlert := provider.buildAlert(&provider.DefaultConfig, ep, testAlert, result, false)

	if firingAlert.Labels["alertname"] != "GatusEndpointDown" {
		t.Errorf("expected alertname to be 'GatusEndpointDown', got %s", firingAlert.Labels["alertname"])
	}

	if firingAlert.Labels["instance"] != ep.URL {
		t.Errorf("expected instance to be %s, got %s", ep.URL, firingAlert.Labels["instance"])
	}

	if firingAlert.Labels["endpoint"] != ep.Name {
		t.Errorf("expected endpoint to be %s, got %s", ep.Name, firingAlert.Labels["endpoint"])
	}

	if firingAlert.Labels["group"] != ep.Group {
		t.Errorf("expected group to be %s, got %s", ep.Group, firingAlert.Labels["group"])
	}

	if firingAlert.Labels["severity"] != "warning" {
		t.Errorf("expected severity to be 'warning', got %s", firingAlert.Labels["severity"])
	}

	if firingAlert.Labels["environment"] != "test" {
		t.Errorf("expected environment to be 'test', got %s", firingAlert.Labels["environment"])
	}

	if firingAlert.Annotations["runbook"] != "https://wiki.example.com/runbook" {
		t.Errorf("expected runbook annotation, got %s", firingAlert.Annotations["runbook"])
	}

	if firingAlert.EndsAt != (time.Time{}) {
		t.Error("expected EndsAt to be zero for firing alert")
	}

	expectedSummary := "Gatus: Test API"
	if firingAlert.Annotations["summary"] != expectedSummary {
		t.Errorf("expected summary to be '%s', got %s", expectedSummary, firingAlert.Annotations["summary"])
	}

	// Test resolved alert
	resolvedAlert := provider.buildAlert(&provider.DefaultConfig, ep, testAlert, result, true)

	if !resolvedAlert.EndsAt.After(time.Now().Add(-time.Minute)) {
		t.Error("expected EndsAt to be set for resolved alert")
	}

	if resolvedAlert.Annotations["summary"] != expectedSummary {
		t.Errorf("expected resolved summary to be '%s', got %s", expectedSummary, resolvedAlert.Annotations["summary"])
	}
}

func TestAlertProvider_buildAlert_InstanceFallback(t *testing.T) {
	provider := AlertProvider{
		DefaultConfig: Config{
			URLs:            []string{"http://alertmanager:9093"},
			DefaultSeverity: "critical",
		},
	}
	testAlert := &alert.Alert{FailureThreshold: 3, SuccessThreshold: 2}
	result := &endpoint.Result{}

	// When ep.URL is set, instance should be ep.URL
	epWithURL := &endpoint.Endpoint{Name: "API", Group: "prod", URL: "https://api.example.com/health"}
	a := provider.buildAlert(&provider.DefaultConfig, epWithURL, testAlert, result, false)
	if a.Labels["instance"] != "https://api.example.com/health" {
		t.Errorf("expected instance to be URL, got %s", a.Labels["instance"])
	}

	// When ep.URL is empty (e.g. external endpoint via ToEndpoint()), instance falls back to DisplayName
	epNoURL := &endpoint.Endpoint{Name: "Heartbeat", Group: "ops"}
	a = provider.buildAlert(&provider.DefaultConfig, epNoURL, testAlert, result, false)
	if a.Labels["instance"] != "ops/Heartbeat" {
		t.Errorf("expected instance to fall back to DisplayName 'ops/Heartbeat', got %s", a.Labels["instance"])
	}

	// No group: fallback should be just the name
	epNoURLNoGroup := &endpoint.Endpoint{Name: "Heartbeat"}
	a = provider.buildAlert(&provider.DefaultConfig, epNoURLNoGroup, testAlert, result, false)
	if a.Labels["instance"] != "Heartbeat" {
		t.Errorf("expected instance to fall back to name 'Heartbeat', got %s", a.Labels["instance"])
	}
}

func TestSendToURL_URLNormalization(t *testing.T) {
	tests := []struct {
		name        string
		inputURL    string
		expectedPath string
	}{
		{"base URL", "http://am:9093", "/api/v2/alerts"},
		{"base URL with trailing slash", "http://am:9093/", "/api/v2/alerts"},
		{"URL with /api/v2", "http://am:9093/api/v2", "/api/v2/alerts"},
		{"URL with /api/v2/", "http://am:9093/api/v2/", "/api/v2/alerts"},
		{"URL with /api/v2/alerts", "http://am:9093/api/v2/alerts", "/api/v2/alerts"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			// Replace the host in inputURL with the test server's host
			inputURL := "http://" + server.Listener.Addr().String() + test.inputURL[len("http://am:9093"):]

			provider := &AlertProvider{}
			cfg := &Config{}
			_ = provider.sendToURL(cfg, inputURL, []byte(`[]`))

			if gotPath != test.expectedPath {
				t.Errorf("input %q: expected path %q, got %q", test.inputURL, test.expectedPath, gotPath)
			}
		})
	}
}

func TestAlertProvider_Send(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/v2/alerts" {
			t.Errorf("expected path /api/v2/alerts, got %s", r.URL.Path)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var alerts []AlertmanagerAlert
		if err := json.NewDecoder(r.Body).Decode(&alerts); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		if len(alerts) != 1 {
			t.Errorf("expected 1 alert, got %d", len(alerts))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := AlertProvider{
		DefaultConfig: Config{
			URLs: []string{server.URL},
		},
	}

	ep := &endpoint.Endpoint{
		Name: "Test API",
		URL:  "https://api.example.com/health",
	}

	testAlert := &alert.Alert{
		Description:      stringPtr("Test alert"),
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}

	result := &endpoint.Result{
		Success: false,
		Errors:  []string{"test error"},
	}

	err := provider.Send(ep, testAlert, result, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAlertProvider_SendMultipleURLs(t *testing.T) {
	var server1Calls, server2Calls int32

	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&server1Calls, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server1.Close()

	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&server2Calls, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server2.Close()

	provider := AlertProvider{
		DefaultConfig: Config{
			URLs: []string{server1.URL, server2.URL},
		},
	}

	ep := &endpoint.Endpoint{
		Name: "Test API",
		URL:  "https://api.example.com/health",
	}

	testAlert := &alert.Alert{
		Description:      stringPtr("Test alert"),
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}

	result := &endpoint.Result{
		Success: false,
	}

	err := provider.Send(ep, testAlert, result, false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if atomic.LoadInt32(&server1Calls) != 1 {
		t.Errorf("expected server1 to be called once, got %d", server1Calls)
	}

	if atomic.LoadInt32(&server2Calls) != 1 {
		t.Errorf("expected server2 to be called once, got %d", server2Calls)
	}
}

func TestAlertProvider_SendPartialFailure(t *testing.T) {
	var successCalls int32

	failServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer failServer.Close()

	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&successCalls, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer successServer.Close()

	provider := AlertProvider{
		DefaultConfig: Config{
			URLs: []string{failServer.URL, successServer.URL},
		},
	}

	ep := &endpoint.Endpoint{
		Name: "Test API",
		URL:  "https://api.example.com/health",
	}

	testAlert := &alert.Alert{
		Description:      stringPtr("Test alert"),
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}

	result := &endpoint.Result{
		Success: false,
	}

	// Should succeed because at least one Alertmanager succeeded
	err := provider.Send(ep, testAlert, result, false)
	if err != nil {
		t.Errorf("expected no error when at least one Alertmanager succeeds, got: %v", err)
	}

	if atomic.LoadInt32(&successCalls) != 1 {
		t.Errorf("expected success server to be called once, got %d", successCalls)
	}
}

func TestAlertProvider_SendAllFail(t *testing.T) {
	failServer1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer failServer1.Close()

	failServer2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}))
	defer failServer2.Close()

	provider := AlertProvider{
		DefaultConfig: Config{
			URLs: []string{failServer1.URL, failServer2.URL},
		},
	}

	ep := &endpoint.Endpoint{
		Name: "Test API",
		URL:  "https://api.example.com/health",
	}

	testAlert := &alert.Alert{
		Description:      stringPtr("Test alert"),
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}

	result := &endpoint.Result{
		Success: false,
	}

	// Should fail because all Alertmanagers failed
	err := provider.Send(ep, testAlert, result, false)
	if err == nil {
		t.Error("expected error when all Alertmanagers fail, got none")
	}
}

func TestConfig_Merge(t *testing.T) {
	base := Config{
		URLs:            []string{"http://base:9093"},
		DefaultSeverity: "critical",
		ExtraLabels: map[string]string{
			"team": "platform",
		},
	}

	override := Config{
		URLs:            []string{"http://override1:9093", "http://override2:9093"},
		DefaultSeverity: "warning",
		ExtraLabels: map[string]string{
			"environment": "test",
		},
		ExtraAnnotations: map[string]string{
			"runbook": "https://wiki.example.com",
		},
	}

	base.Merge(&override)

	// URLs should be completely replaced by override
	if len(base.URLs) != 2 {
		t.Errorf("expected URLs to be replaced with 2 items, got %d", len(base.URLs))
	}
	if base.URLs[0] != "http://override1:9093" || base.URLs[1] != "http://override2:9093" {
		t.Errorf("expected URLs to be overridden, got %v", base.URLs)
	}

	if base.DefaultSeverity != "warning" {
		t.Errorf("expected severity to be overridden to 'warning', got %s", base.DefaultSeverity)
	}

	if base.ExtraLabels["team"] != "platform" {
		t.Error("expected original label to be preserved")
	}

	if base.ExtraLabels["environment"] != "test" {
		t.Error("expected override label to be added")
	}

	if base.ExtraAnnotations["runbook"] != "https://wiki.example.com" {
		t.Error("expected override annotation to be added")
	}
}

func stringPtr(s string) *string {
	return &s
}

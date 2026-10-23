package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
)

const testMaxRemoteBody = 1024

func testRemoteInstance(remoteServerURL string) remote.Instance {
	return remote.Instance{
		EndpointPrefix:         "remote-",
		URL:                    remoteServerURL + "/api/v1/endpoints/statuses",
		AllowPrivateNetworks:   true,
	}
}

func TestGetEndpointStatusesFromRemoteInstances(t *testing.T) {
	t.Parallel()

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/endpoints/statuses" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode([]*endpoint.Status{{
			Name:  "backend",
			Group: "core",
			Key:   "core_backend",
			Results: []*endpoint.Result{{
				Success:   true,
				Timestamp: time.Now(),
				Duration:  time.Millisecond,
			}},
		}})
	}))
	defer remoteServer.Close()

	remoteConfig := &remote.Config{
		Instances: []remote.Instance{testRemoteInstance(remoteServer.URL)},
	}
	_ = remoteConfig.ValidateAndSetDefaults()

	statuses, err := getEndpointStatusesFromRemoteInstances(remoteConfig, nil)
	if err != nil {
		t.Fatalf("expected no error, got %s", err.Error())
	}
	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d", len(statuses))
	}
	if statuses[0].Key != "@remote:0:core_backend" {
		t.Fatalf("expected prefixed key, got %s", statuses[0].Key)
	}
	if statuses[0].Name != "remote-backend" {
		t.Fatalf("expected prefixed name, got %s", statuses[0].Name)
	}
}

func TestGetEndpointStatusesFromRemoteInstancesSkipsOversizedBody(t *testing.T) {
	t.Parallel()

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, testMaxRemoteBody+1))
	}))
	defer remoteServer.Close()

	remoteConfig := &remote.Config{
		MaxResponseBody: testMaxRemoteBody,
		Instances:       []remote.Instance{testRemoteInstance(remoteServer.URL)},
	}
	_ = remoteConfig.ValidateAndSetDefaults()

	statuses, err := getEndpointStatusesFromRemoteInstances(remoteConfig, nil)
	if err == nil {
		t.Fatal("expected error when all remotes return oversized bodies")
	}
	if len(statuses) != 0 {
		t.Fatalf("expected no statuses, got %d", len(statuses))
	}
}

func TestEndpointStatusProxiesRemoteEndpoint(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/endpoints/core_backend/statuses":
			_ = json.NewEncoder(w).Encode(endpoint.Status{
				Name:  "backend",
				Group: "core",
				Key:   "core_backend",
				Results: []*endpoint.Result{{
					Success:   true,
					Timestamp: time.Now(),
					Duration:  2 * time.Millisecond,
				}},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer remoteServer.Close()

	cfg := &config.Config{
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			Instances: []remote.Instance{testRemoteInstance(remoteServer.URL)},
		},
	}
	_ = cfg.Remote.ValidateAndSetDefaults()

	app := New(cfg).Router()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/@remote:0:core_backend/statuses?page=1&pageSize=20", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, string(body))
	}

	var status endpoint.Status
	if err := json.NewDecoder(response.Body).Decode(&status); err != nil {
		t.Fatalf("failed to decode response: %s", err.Error())
	}
	if status.Key != "@remote:0:core_backend" {
		t.Fatalf("expected prefixed key in response, got %s", status.Key)
	}
	if status.Name != "remote-backend" {
		t.Fatalf("expected prefixed name in response, got %s", status.Name)
	}
}

func TestEndpointStatusDoesNotProbeRemotesForNonPrefixedKey(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()

	var remoteRequestCount atomic.Int32
	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteRequestCount.Add(1)
		http.NotFound(w, r)
	}))
	defer remoteServer.Close()

	cfg := &config.Config{
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			Instances: []remote.Instance{testRemoteInstance(remoteServer.URL)},
		},
	}
	_ = cfg.Remote.ValidateAndSetDefaults()

	app := New(cfg).Router()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/core_backend/statuses", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 404, got %d: %s", response.StatusCode, string(body))
	}
	if remoteRequestCount.Load() != 0 {
		t.Fatalf("expected no remote requests, got %d", remoteRequestCount.Load())
	}
}

func TestEndpointStatusReturns502ForOversizedRemoteBody(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, testMaxRemoteBody+1))
	}))
	defer remoteServer.Close()

	cfg := &config.Config{
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			MaxResponseBody: testMaxRemoteBody,
			Instances:       []remote.Instance{testRemoteInstance(remoteServer.URL)},
		},
	}
	_ = cfg.Remote.ValidateAndSetDefaults()

	app := New(cfg).Router()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/@remote:0:core_backend/statuses", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadGateway {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 502, got %d: %s", response.StatusCode, string(body))
	}
}

func TestProxyRemoteEndpointForwardsBadge(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/endpoints/core_backend/health/badge.svg" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte("<svg>remote</svg>"))
	}))
	defer remoteServer.Close()

	cfg := &config.Config{
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			Instances: []remote.Instance{{
				URL:                  remoteServer.URL + "/api/v1/endpoints/statuses",
				AllowPrivateNetworks: true,
			}},
		},
	}
	_ = cfg.Remote.ValidateAndSetDefaults()

	app := New(cfg).Router()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/@remote:0:core_backend/health/badge.svg", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, string(body))
	}
	body, _ := io.ReadAll(response.Body)
	if string(body) != "<svg>remote</svg>" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestProxyRemoteEndpointForcesSafeContentType(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/endpoints/core_backend/health/badge.svg" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<svg>remote</svg>"))
	}))
	defer remoteServer.Close()

	cfg := &config.Config{
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			Instances: []remote.Instance{{
				URL:                  remoteServer.URL + "/api/v1/endpoints/statuses",
				AllowPrivateNetworks: true,
			}},
		},
	}
	_ = cfg.Remote.ValidateAndSetDefaults()

	app := New(cfg).Router()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/@remote:0:core_backend/health/badge.svg", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 200, got %d: %s", response.StatusCode, string(body))
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "image/svg+xml" {
		t.Fatalf("expected image/svg+xml, got %s", contentType)
	}
}

func TestInstanceAuthorizationOverridesInboundHeaders(t *testing.T) {
	t.Parallel()

	var receivedAuthorization string
	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuthorization = r.Header.Get("Authorization")
		_ = json.NewEncoder(w).Encode([]*endpoint.Status{{
			Name: "backend",
			Key:  "core_backend",
		}})
	}))
	defer remoteServer.Close()

	instance := remote.Instance{
		URL:                  remoteServer.URL + "/api/v1/endpoints/statuses",
		Authorization:        "Bearer instance-token",
		AllowPrivateNetworks: true,
	}
	remoteConfig := &remote.Config{Instances: []remote.Instance{instance}}
	_ = remoteConfig.ValidateAndSetDefaults()

	forwardHeaders := make(http.Header)
	forwardHeaders.Set("Authorization", "Bearer inbound-token")
	forwardHeaders.Set("Cookie", "session=abc")

	_, err := getEndpointStatusesFromRemoteInstances(remoteConfig, forwardHeaders)
	if err != nil {
		t.Fatalf("expected no error, got %s", err.Error())
	}
	if receivedAuthorization != "Bearer instance-token" {
		t.Fatalf("expected instance authorization to be used, got %q", receivedAuthorization)
	}
}

func TestContentTypeForSubPath(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"/health/badge.svg":                 "image/svg+xml",
		"/response-times/24h/chart.svg":     "image/svg+xml",
		"/health/badge.shields":             "application/json",
		"/response-times/24h/history":       "application/json",
		"/uptimes/24h":                      "text/plain",
		"/response-times/24h":               "text/plain",
	}
	for subPath, expected := range tests {
		if got := contentTypeForSubPath(subPath); got != expected {
			t.Fatalf("contentTypeForSubPath(%q) = %q, expected %q", subPath, got, expected)
		}
	}
}

func TestReadBoundedBodyRejectsOversizedResponse(t *testing.T) {
	t.Parallel()

	body := io.NopCloser(strings.NewReader(strings.Repeat("a", testMaxRemoteBody+1)))
	_, err := readBoundedBody(body, testMaxRemoteBody)
	if err != errResponseBodyTooLarge {
		t.Fatalf("expected errResponseBodyTooLarge, got %v", err)
	}
}

func TestGetEndpointStatusesFromRemoteInstancesFetchesInParallel(t *testing.T) {
	t.Parallel()

	var slowRequestCount atomic.Int32
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slowRequestCount.Add(1)
		time.Sleep(200 * time.Millisecond)
		_ = json.NewEncoder(w).Encode([]*endpoint.Status{{
			Name: "slow",
			Key:  "slow_backend",
		}})
	}))
	defer slowServer.Close()

	fastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]*endpoint.Status{{
			Name: "fast",
			Key:  "fast_backend",
		}})
	}))
	defer fastServer.Close()

	remoteConfig := &remote.Config{
		Instances: []remote.Instance{
			testRemoteInstance(slowServer.URL),
			testRemoteInstance(fastServer.URL),
		},
	}
	_ = remoteConfig.ValidateAndSetDefaults()

	startedAt := time.Now()
	statuses, err := getEndpointStatusesFromRemoteInstances(remoteConfig, nil)
	elapsed := time.Since(startedAt)
	if err != nil {
		t.Fatalf("expected no error, got %s", err.Error())
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
	if slowRequestCount.Load() != 1 {
		t.Fatalf("expected slow remote to be called once, got %d", slowRequestCount.Load())
	}
	if elapsed >= 350*time.Millisecond {
		t.Fatalf("expected parallel fetch to finish in under 350ms, took %s", elapsed)
	}
}

func TestEndpointStatusProxiesRemote500Response(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream failed", http.StatusInternalServerError)
	}))
	defer remoteServer.Close()

	cfg := &config.Config{
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			Instances: []remote.Instance{testRemoteInstance(remoteServer.URL)},
		},
	}
	_ = cfg.Remote.ValidateAndSetDefaults()

	app := New(cfg).Router()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/@remote:0:core_backend/statuses", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusInternalServerError {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 500, got %d: %s", response.StatusCode, string(body))
	}
	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected application/json, got %s", contentType)
	}
}

func TestProxyRemoteEndpointTimesOutSlowRemote(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	defer resetRemoteCircuitBreakers()

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte("<svg>remote</svg>"))
	}))
	defer remoteServer.Close()

	cfg := &config.Config{
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			Instances: []remote.Instance{testRemoteInstance(remoteServer.URL)},
			ClientConfig: &client.Config{
				Timeout: 50 * time.Millisecond,
			},
		},
	}
	_ = cfg.Remote.ValidateAndSetDefaults()

	app := New(cfg).Router()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/@remote:0:core_backend/health/badge.svg", http.NoBody)
	response, err := app.Test(request, int(2*time.Second/time.Millisecond))
	if err != nil {
		t.Fatalf("request failed: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusGatewayTimeout {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 504 after remote timeout, got %d: %s", response.StatusCode, string(body))
	}
}

func TestRemoteCircuitBreakerOpensAfterConsecutiveFailures(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()
	defer resetRemoteCircuitBreakers()

	var remoteRequestCount atomic.Int32
	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remoteRequestCount.Add(1)
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte("<svg>remote</svg>"))
	}))
	defer remoteServer.Close()

	cfg := &config.Config{
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			Instances: []remote.Instance{testRemoteInstance(remoteServer.URL)},
			ClientConfig: &client.Config{
				Timeout: 50 * time.Millisecond,
			},
			CircuitBreaker: &remote.CircuitBreakerConfig{
				FailureThreshold: 2,
				OpenDuration:     time.Hour,
			},
		},
	}
	_ = cfg.Remote.ValidateAndSetDefaults()

	app := New(cfg).Router()
	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/@remote:0:core_backend/health/badge.svg", http.NoBody)
		response, err := app.Test(request, int(2*time.Second/time.Millisecond))
		if err != nil {
			t.Fatalf("request %d failed: %s", i+1, err.Error())
		}
		_ = response.Body.Close()
		if response.StatusCode != http.StatusGatewayTimeout {
			t.Fatalf("expected 504 on attempt %d, got %d", i+1, response.StatusCode)
		}
	}

	startedAt := time.Now()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/@remote:0:core_backend/health/badge.svg", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("circuit breaker request failed: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("expected 503 with open circuit breaker, got %d: %s", response.StatusCode, string(body))
	}
	if time.Since(startedAt) > 100*time.Millisecond {
		t.Fatalf("expected open circuit breaker to fail fast, took %s", time.Since(startedAt))
	}
	if remoteRequestCount.Load() != 2 {
		t.Fatalf("expected 2 remote requests before circuit opened, got %d", remoteRequestCount.Load())
	}
}

func TestProxyRemoteEndpointDoesNotForwardSetCookie(t *testing.T) {
	defer store.Get().Clear()
	defer cache.Clear()

	remoteServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Set-Cookie", "session=evil")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<svg>remote</svg>"))
	}))
	defer remoteServer.Close()

	cfg := &config.Config{
		Storage: &storage.Config{
			MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
			MaximumNumberOfEvents:  storage.DefaultMaximumNumberOfEvents,
		},
		Remote: &remote.Config{
			Instances: []remote.Instance{testRemoteInstance(remoteServer.URL)},
		},
	}
	_ = cfg.Remote.ValidateAndSetDefaults()

	app := New(cfg).Router()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/endpoints/@remote:0:core_backend/health/badge.svg", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %s", err.Error())
	}
	defer response.Body.Close()
	if response.Header.Get("Set-Cookie") != "" {
		t.Fatalf("expected Set-Cookie not to be forwarded, got %q", response.Header.Get("Set-Cookie"))
	}
}

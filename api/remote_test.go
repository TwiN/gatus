package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
)

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
		Instances: []remote.Instance{{
			EndpointPrefix: "remote-",
			URL:            remoteServer.URL + "/api/v1/endpoints/statuses",
		}},
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
			Instances: []remote.Instance{{
				EndpointPrefix: "remote-",
				URL:            remoteServer.URL + "/api/v1/endpoints/statuses",
			}},
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
				URL: remoteServer.URL + "/api/v1/endpoints/statuses",
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

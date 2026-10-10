package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/watchdog"
)

func TestRecoveryRetriesAfterSQLiteRestart(t *testing.T) {
	t.Setenv("MOCK_ALERT_PROVIDER", "false")
	cfgStorage := &storage.Config{Type: storage.TypeSQLite, Path: filepath.Join(t.TempDir(), "alerts.db")}
	if err := cfgStorage.ValidateAndSetDefaults(); err != nil {
		t.Fatal(err)
	}
	var requests, status atomic.Int32
	status.Store(http.StatusOK)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(int(status.Load()))
	}))
	defer server.Close()
	newConfig := func() *config.Config {
		enabled := true
		return &config.Config{
			Storage: cfgStorage,
			Endpoints: []*endpoint.Endpoint{{Name: "recovery-restart", Alerts: []*alert.Alert{{
				Type: alert.TypeCustom, Enabled: &enabled, SendOnResolved: &enabled,
				FailureThreshold: 1, SuccessThreshold: 2,
			}}}},
			Alerting: &alerting.Config{Custom: &custom.AlertProvider{DefaultConfig: custom.Config{URL: server.URL, Method: http.MethodPost}}},
		}
	}
	cfg := newConfig()
	initializeStorage(cfg)
	t.Cleanup(func() {
		store.Get().Close()
		if err := store.Initialize(nil); err != nil {
			t.Error(err)
		}
	})
	ep := cfg.Endpoints[0]
	watchdog.HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	if requests.Load() != 1 || !ep.Alerts[0].Triggered {
		t.Fatal("initial alert was not delivered")
	}
	if exists, _, _, err := store.Get().GetTriggeredEndpointAlert(ep, ep.Alerts[0]); err != nil || !exists {
		t.Fatalf("initial alert must be persisted: exists=%v err=%v", exists, err)
	}
	status.Store(http.StatusServiceUnavailable)
	watchdog.HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	if requests.Load() != 1 {
		t.Fatal("recovery sent before the success threshold")
	}
	watchdog.HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	if requests.Load() != 2 || !ep.Alerts[0].Triggered {
		t.Fatal("failed recovery must remain pending")
	}
	exists, _, persistedSuccesses, err := store.Get().GetTriggeredEndpointAlert(ep, ep.Alerts[0])
	if err != nil || !exists {
		t.Fatalf("failed recovery must remain persisted: exists=%v err=%v", exists, err)
	}
	// Reopen the same database with fresh objects through the application startup path.
	store.Get().Close()
	cfg = newConfig()
	initializeStorage(cfg)
	ep = cfg.Endpoints[0]
	if !ep.Alerts[0].Triggered || ep.NumberOfSuccessesInARow != persistedSuccesses {
		t.Fatalf("pending recovery was not restored: triggered=%v successes=%d", ep.Alerts[0].Triggered, ep.NumberOfSuccessesInARow)
	}
	if requests.Load() != 2 {
		t.Fatal("loading persisted state must not send a notification")
	}
	status.Store(http.StatusOK)
	watchdog.HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	if requests.Load() != 3 || ep.Alerts[0].Triggered {
		t.Fatal("pending recovery was not delivered after restart")
	}
	if exists, _, _, err := store.Get().GetTriggeredEndpointAlert(ep, ep.Alerts[0]); err != nil || exists {
		t.Fatalf("delivered recovery must be removed from storage: exists=%v err=%v", exists, err)
	}
	watchdog.HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	if requests.Load() != 3 {
		t.Fatal("successful recovery sent twice")
	}
	// A second restart must not restore or resend the delivered recovery.
	store.Get().Close()
	cfg = newConfig()
	initializeStorage(cfg)
	ep = cfg.Endpoints[0]
	if ep.Alerts[0].Triggered {
		t.Fatal("delivered recovery was restored after restart")
	}
	watchdog.HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	watchdog.HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	if requests.Load() != 3 {
		t.Fatal("delivered recovery sent again after restart")
	}
}

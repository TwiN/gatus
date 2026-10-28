package watchdog

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
)

func TestRecoveryRetriesRealProvider(t *testing.T) {
	t.Setenv("MOCK_ALERT_PROVIDER", "false")
	cfgStorage := &storage.Config{Type: storage.TypeSQLite, Path: t.TempDir() + "/alerts.db"}
	if err := cfgStorage.ValidateAndSetDefaults(); err != nil {
		t.Fatal(err)
	}
	if err := store.Initialize(cfgStorage); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		store.Get().Close()
		if err := store.Initialize(nil); err != nil {
			t.Error(err)
		}
	})
	var requests, status atomic.Int32
	status.Store(http.StatusServiceUnavailable)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(int(status.Load()))
	}))
	defer server.Close()
	enabled := true
	ep := &endpoint.Endpoint{Name: "recovery-retry", Alerts: []*alert.Alert{{Type: alert.TypeCustom, Enabled: &enabled, Triggered: true, SendOnResolved: &enabled, SuccessThreshold: 2}}}
	cfg := &alerting.Config{Custom: &custom.AlertProvider{DefaultConfig: custom.Config{URL: server.URL, Method: "POST"}}}
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg)
	if requests.Load() != 0 {
		t.Fatal("recovery sent before threshold")
	}
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg)
	if requests.Load() != 1 || !ep.Alerts[0].Triggered {
		t.Fatal("failed recovery was lost")
	}
	if exists, _, _, err := store.Get().GetTriggeredEndpointAlert(ep, ep.Alerts[0]); err != nil || !exists {
		t.Fatalf("failed recovery must remain persisted: exists=%v err=%v", exists, err)
	}
	// A new failure before delivery belongs to the same pending incident.
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg)
	if requests.Load() != 1 || !ep.Alerts[0].Triggered {
		t.Fatal("pending recovery should not start a second incident")
	}
	status.Store(http.StatusOK)
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg)
	if requests.Load() != 1 {
		t.Fatal("recovery retried before the success threshold after recurrence")
	}
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg)
	if requests.Load() != 2 || ep.Alerts[0].Triggered {
		t.Fatal("pending recovery was not retried")
	}
	if exists, _, _, err := store.Get().GetTriggeredEndpointAlert(ep, ep.Alerts[0]); err != nil || exists {
		t.Fatalf("delivered recovery must be removed from storage: exists=%v err=%v", exists, err)
	}
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg)
	if requests.Load() != 2 {
		t.Fatal("successful recovery sent twice")
	}
}

func TestRecoveryWaitsForConfiguredProvider(t *testing.T) {
	t.Setenv("MOCK_ALERT_PROVIDER", "true")
	t.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")
	enabled := true
	ep := &endpoint.Endpoint{Name: "missing-provider", Alerts: []*alert.Alert{{Type: alert.TypeCustom, Triggered: true, SendOnResolved: &enabled, SuccessThreshold: 1}}}
	HandleAlerting(ep, &endpoint.Result{Success: true}, &alerting.Config{})
	if !ep.Alerts[0].Triggered {
		t.Fatal("recovery without a provider must remain pending")
	}
	cfg := &alerting.Config{Custom: &custom.AlertProvider{}}
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg)
	if ep.Alerts[0].Triggered {
		t.Fatal("recovery must resolve after the provider becomes available")
	}
}

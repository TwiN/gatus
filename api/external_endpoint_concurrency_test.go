package api

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/storage"
	"github.com/TwiN/gatus/v5/storage/store"
)

func TestExternalEndpointConcurrentRecovery(t *testing.T) {
	for _, scenario := range []struct {
		name          string
		secondSuccess bool
		pendingState  string
		recoveryFails bool
	}{
		{name: "another recovery", secondSuccess: true, pendingState: "RESOLVED"},
		{name: "a new failure", secondSuccess: false, pendingState: "RESOLVED"},
		{name: "recovery during initial delivery", secondSuccess: true, pendingState: "TRIGGERED"},
		{name: "failure during failed recovery", secondSuccess: false, pendingState: "RESOLVED", recoveryFails: true},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			t.Setenv("MOCK_ALERT_PROVIDER", "false")
			cfgStorage := &storage.Config{Type: storage.TypeSQLite, Path: filepath.Join(t.TempDir(), "alerts.db")}
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
			var triggered, resolved atomic.Int32
			deliveryEntered := make(chan struct{})
			releaseDelivery := make(chan struct{})
			var entered, release sync.Once
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				state := r.URL.Query().Get("state")
				if state == "RESOLVED" {
					resolved.Add(1)
				} else {
					triggered.Add(1)
				}
				if state == scenario.pendingState {
					entered.Do(func() { close(deliveryEntered) })
					<-releaseDelivery
				}
				if state == "RESOLVED" && scenario.recoveryFails {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(provider.Close)
			enabled := true
			ee := &endpoint.ExternalEndpoint{Name: "concurrent", Token: "test-token", Alerts: []*alert.Alert{{
				Type: alert.TypeCustom, SendOnResolved: &enabled, FailureThreshold: 1, SuccessThreshold: 1,
			}}}
			cfg := &config.Config{
				ExternalEndpoints: []*endpoint.ExternalEndpoint{ee, {Name: "independent", Token: "test-token"}},
				Maintenance:       &maintenance.Config{},
				Alerting: &alerting.Config{Custom: &custom.AlertProvider{DefaultConfig: custom.Config{
					URL: provider.URL + "?state=[ALERT_TRIGGERED_OR_RESOLVED]", Method: http.MethodPost,
				}}},
			}
			app := New(cfg).Router()
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			serveError := make(chan error, 1)
			go func() { serveError <- app.Listener(listener) }()
			t.Cleanup(func() {
				release.Do(func() { close(releaseDelivery) })
				if err := app.Shutdown(); err != nil {
					t.Error(err)
				}
				if err := <-serveError; err != nil {
					t.Error(err)
				}
			})
			client := &http.Client{Timeout: 3 * time.Second}
			t.Cleanup(client.CloseIdleConnections)
			push := func(name string, success bool) error {
				request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("http://%s/api/v1/endpoints/_%s/external?success=%t", listener.Addr(), name, success), http.NoBody)
				if err != nil {
					return err
				}
				request.Header.Set("Authorization", "Bearer test-token")
				response, err := client.Do(request)
				if err != nil {
					return err
				}
				defer response.Body.Close()
				if response.StatusCode != http.StatusOK {
					return fmt.Errorf("unexpected status: %d", response.StatusCode)
				}
				return nil
			}
			if scenario.pendingState == "RESOLVED" {
				if err := push("concurrent", false); err != nil {
					t.Fatal(err)
				}
			}
			firstDone := make(chan error, 1)
			go func() { firstDone <- push("concurrent", scenario.pendingState == "RESOLVED") }()
			select {
			case <-deliveryEntered:
			case <-time.After(3 * time.Second):
				t.Fatal("notification did not reach the provider")
			}
			secondDone := make(chan error, 1)
			go func() { secondDone <- push("concurrent", scenario.secondSuccess) }()
			select {
			case err := <-secondDone:
				t.Fatalf("concurrent update completed while alert delivery was pending: %v", err)
			case <-time.After(100 * time.Millisecond):
			}
			// A slow provider for this endpoint must not block other endpoints.
			if err := push("independent", true); err != nil {
				t.Fatal(err)
			}
			release.Do(func() { close(releaseDelivery) })
			if err := <-firstDone; err != nil {
				t.Fatal(err)
			}
			if err := <-secondDone; err != nil {
				t.Fatal(err)
			}
			wantTriggered := int32(1)
			if !scenario.secondSuccess && !scenario.recoveryFails {
				wantTriggered = 2
			}
			if triggered.Load() != wantTriggered || resolved.Load() != 1 {
				t.Fatalf("unexpected notifications: triggered=%d resolved=%d", triggered.Load(), resolved.Load())
			}
			if ee.Alerts[0].Triggered != !scenario.secondSuccess {
				t.Fatal("concurrent update left the wrong incident state")
			}
			if exists, _, _, err := store.Get().GetTriggeredEndpointAlert(ee.ToEndpoint(), ee.Alerts[0]); err != nil || exists != !scenario.secondSuccess {
				t.Fatalf("concurrent update left the wrong persisted state: exists=%v err=%v", exists, err)
			}
		})
	}
}

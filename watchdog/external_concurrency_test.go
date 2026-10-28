package watchdog_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/api"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/heartbeat"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/test"
	"github.com/TwiN/gatus/v5/watchdog"
)

// The push handler and the heartbeat ticker both handle an external endpoint, on copies of it that share its alerts. This
// drives the real push route and the real heartbeat check at the same time and counts what the receiver gets.
//
// Run it with -race. The per-endpoint lock in HandleAlerting alone keeps the notification count at one, so removing the
// lock from the push handler or from the heartbeat is only caught by the race detector, on the counters and the shared alerts.
func TestExternalEndpointPushedAndHeartbeatCheckedConcurrentlySendsOneNotification(t *testing.T) {
	var mutex sync.Mutex
	var bodies []string
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		body, _ := io.ReadAll(r.Body)
		time.Sleep(20 * time.Millisecond)
		mutex.Lock()
		bodies = append(bodies, string(body))
		mutex.Unlock()
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	})})
	defer client.InjectHTTPClient(nil)
	watchdog.InitializeForTest()
	enabled := true
	ee := &endpoint.ExternalEndpoint{
		Name:      "concurrent",
		Group:     "external",
		Token:     "secret-token",
		Heartbeat: heartbeat.Config{Interval: time.Minute},
		Alerts:    []*alert.Alert{{Type: alert.TypeCustom, Enabled: &enabled, FailureThreshold: 1, SuccessThreshold: 1, SendOnResolved: &enabled}},
	}
	if err := ee.ValidateAndSetDefaults(); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		ExternalEndpoints: []*endpoint.ExternalEndpoint{ee},
		Maintenance:       &maintenance.Config{},
		Alerting: &alerting.Config{Custom: &custom.AlertProvider{DefaultConfig: custom.Config{
			URL: "https://example.com/hook", Method: http.MethodPost, Body: "[ALERT_TRIGGERED_OR_RESOLVED] [INCIDENT_ID]",
		}}},
	}
	router := api.New(cfg).Router()
	var waitGroup sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 6; i++ {
		waitGroup.Add(2)
		go func() {
			defer waitGroup.Done()
			<-start
			request := httptest.NewRequest(http.MethodPost, "/api/v1/endpoints/"+ee.Key()+"/external?success=false&error=down", http.NoBody)
			request.Header.Set("Authorization", "Bearer secret-token")
			response, err := router.Test(request)
			if err != nil {
				t.Error(err)
				return
			}
			_ = response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Errorf("expected the push to return 200, got %d", response.StatusCode)
			}
		}()
		go func() {
			defer waitGroup.Done()
			<-start
			watchdog.ExecuteExternalEndpointHeartbeat(ee, cfg, nil)
		}()
	}
	close(start)
	waitGroup.Wait()
	if len(bodies) != 1 || !strings.HasPrefix(bodies[0], "TRIGGERED ") {
		t.Errorf("expected exactly one TRIGGERED notification, got %d: %v", len(bodies), bodies)
	}
}

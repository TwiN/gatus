package watchdog

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

// An external endpoint is handled from the push handler and from the heartbeat ticker, each on its own copy of the endpoint
// and both on the same alerts. One incident must send one notification, with one incident id.
func TestHandleAlertingOfAnExternalEndpointFromSeveralGoroutines(t *testing.T) {
	var mutex sync.Mutex
	var bodies []string
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		body, _ := io.ReadAll(r.Body)
		time.Sleep(5 * time.Millisecond)
		mutex.Lock()
		bodies = append(bodies, string(body))
		mutex.Unlock()
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	})})
	defer client.InjectHTTPClient(nil)
	alertingConfig := newIncidentIDAlertingConfig()
	enabled := true
	ee := &endpoint.ExternalEndpoint{Name: "x", Group: "g", Token: "t", Alerts: []*alert.Alert{
		{Type: alert.TypeCustom, Enabled: &enabled, FailureThreshold: 1, SuccessThreshold: 1, SendOnResolved: &enabled},
	}}
	var waitGroup sync.WaitGroup
	for i := 0; i < 8; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			unlock := LockExternalEndpoint(ee.Key())
			defer unlock()
			HandleAlerting(ee.ToEndpoint(), &endpoint.Result{Success: false}, alertingConfig)
		}()
	}
	waitGroup.Wait()
	if len(bodies) != 1 || !strings.HasPrefix(bodies[0], "TRIGGERED ") {
		t.Errorf("expected exactly one TRIGGERED notification, got %d: %v", len(bodies), bodies)
	}
}

// HandleAlerting itself must hold up when it is called concurrently for the same endpoint, whoever the callers are.
func TestHandleAlertingFromSeveralGoroutinesSendsOneNotification(t *testing.T) {
	var mutex sync.Mutex
	var bodies []string
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		body, _ := io.ReadAll(r.Body)
		time.Sleep(5 * time.Millisecond)
		mutex.Lock()
		bodies = append(bodies, string(body))
		mutex.Unlock()
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	})})
	defer client.InjectHTTPClient(nil)
	alertingConfig := newIncidentIDAlertingConfig()
	enabled := true
	ee := &endpoint.ExternalEndpoint{Name: "y", Group: "g", Token: "t", Alerts: []*alert.Alert{
		{Type: alert.TypeCustom, Enabled: &enabled, FailureThreshold: 1, SuccessThreshold: 1, SendOnResolved: &enabled},
	}}
	var waitGroup sync.WaitGroup
	for i := 0; i < 8; i++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			HandleAlerting(ee.ToEndpoint(), &endpoint.Result{Success: false}, alertingConfig)
		}()
	}
	waitGroup.Wait()
	if len(bodies) != 1 {
		t.Errorf("expected exactly one notification, got %d: %v", len(bodies), bodies)
	}
}

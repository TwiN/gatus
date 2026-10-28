package watchdog

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

// incidentIDRecorder replaces the HTTP client so that the body of every request sent by the custom alert provider is
// recorded, which allows the test to observe the incident id the way a receiver would.
func incidentIDRecorder(t *testing.T, statusCode *int) *[]string {
	t.Helper()
	var bodies []string
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		return &http.Response{StatusCode: *statusCode, Body: http.NoBody}
	})})
	t.Cleanup(func() { client.InjectHTTPClient(nil) })
	return &bodies
}

func newIncidentIDAlertingConfig() *alerting.Config {
	return &alerting.Config{
		Custom: &custom.AlertProvider{
			DefaultConfig: custom.Config{
				URL:    "https://example.com/webhook",
				Method: "POST",
				Body:   "[ALERT_TRIGGERED_OR_RESOLVED] [INCIDENT_ID]",
			},
		},
	}
}

func TestHandleAlertingIncidentID(t *testing.T) {
	statusCode := http.StatusOK
	bodies := incidentIDRecorder(t, &statusCode)
	alertingConfig := newIncidentIDAlertingConfig()
	enabled := true
	ep := &endpoint.Endpoint{
		URL: "https://example.com",
		Alerts: []*alert.Alert{
			{Type: alert.TypeCustom, Enabled: &enabled, FailureThreshold: 2, SuccessThreshold: 2, SendOnResolved: &enabled, MinimumReminderInterval: 5 * time.Minute},
		},
	}
	failure, success := &endpoint.Result{Success: false}, &endpoint.Result{Success: true}
	// First incident: triggered, reminded, then resolved. All three notifications must carry the same id.
	HandleAlerting(ep, failure, alertingConfig)
	HandleAlerting(ep, failure, alertingConfig)
	ep.LastReminderSent = time.Now().Add(-time.Hour)
	HandleAlerting(ep, failure, alertingConfig)
	HandleAlerting(ep, success, alertingConfig)
	HandleAlerting(ep, success, alertingConfig)
	if len(*bodies) != 3 {
		t.Fatalf("expected 3 notifications, got %d: %v", len(*bodies), *bodies)
	}
	firstID := (*bodies)[0][len("TRIGGERED "):]
	if len(firstID) == 0 {
		t.Fatal("expected the notification to carry an incident id")
	}
	if (*bodies)[1] != "TRIGGERED "+firstID || (*bodies)[2] != "RESOLVED "+firstID {
		t.Errorf("expected the reminder and the resolution to carry the id %q, got %v", firstID, *bodies)
	}
	if ep.Alerts[0].ResolveKey != "" {
		t.Error("expected the incident id to be cleared once the alert is resolved")
	}
	// Second incident: must get a new id
	HandleAlerting(ep, failure, alertingConfig)
	HandleAlerting(ep, failure, alertingConfig)
	secondID := (*bodies)[3][len("TRIGGERED "):]
	if len(secondID) == 0 || secondID == firstID {
		t.Errorf("expected a new incident id for the second incident, got %q (first was %q)", secondID, firstID)
	}
}

func TestHandleAlertingIncidentIDIsClearedWhenSendOnResolvedIsFalse(t *testing.T) {
	statusCode := http.StatusOK
	bodies := incidentIDRecorder(t, &statusCode)
	alertingConfig := newIncidentIDAlertingConfig()
	enabled, disabled := true, false
	ep := &endpoint.Endpoint{
		URL:    "https://example.com",
		Alerts: []*alert.Alert{{Type: alert.TypeCustom, Enabled: &enabled, FailureThreshold: 1, SuccessThreshold: 1, SendOnResolved: &disabled}},
	}
	HandleAlerting(ep, &endpoint.Result{Success: false}, alertingConfig)
	HandleAlerting(ep, &endpoint.Result{Success: true}, alertingConfig)
	HandleAlerting(ep, &endpoint.Result{Success: false}, alertingConfig)
	if len(*bodies) != 2 || (*bodies)[0] == (*bodies)[1] {
		t.Errorf("expected two TRIGGERED notifications with different incident ids, got %v", *bodies)
	}
}

func TestHandleAlertingIncidentIDIsReusedWhenTheInitialNotificationFailsThenClearedOnRecovery(t *testing.T) {
	statusCode := http.StatusInternalServerError
	bodies := incidentIDRecorder(t, &statusCode)
	alertingConfig := newIncidentIDAlertingConfig()
	enabled := true
	ep := &endpoint.Endpoint{
		URL:    "https://example.com",
		Alerts: []*alert.Alert{{Type: alert.TypeCustom, Enabled: &enabled, FailureThreshold: 1, SuccessThreshold: 1, SendOnResolved: &enabled}},
	}
	// The receiver is down, so the alert is never marked as triggered and the notification is attempted again.
	// Both attempts describe the same incident, so they must carry the same id.
	HandleAlerting(ep, &endpoint.Result{Success: false}, alertingConfig)
	HandleAlerting(ep, &endpoint.Result{Success: false}, alertingConfig)
	if len(*bodies) != 2 || (*bodies)[0] != (*bodies)[1] {
		t.Fatalf("expected two attempts with the same incident id, got %v", *bodies)
	}
	// The endpoint recovers before the receiver does: nothing is sent, and the id of that incident must not leak
	HandleAlerting(ep, &endpoint.Result{Success: true}, alertingConfig)
	statusCode = http.StatusOK
	HandleAlerting(ep, &endpoint.Result{Success: false}, alertingConfig)
	if len(*bodies) != 3 || (*bodies)[2] == (*bodies)[0] {
		t.Errorf("expected a new incident id after the recovery, got %v", *bodies)
	}
}

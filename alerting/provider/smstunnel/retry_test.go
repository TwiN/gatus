package smstunnel_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/smstunnel"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/watchdog"
)

// fakeAPI behaves like the API's send route when it comes to Idempotency-Key: a key it has already seen gets the
// first answer back and no new text. These tests run it behind the real alerting loop (watchdog.HandleAlerting).
type fakeAPI struct {
	texts   []string       // every text the API accepted
	answers map[string]int // status of the first answer for each key

	loseNextAnswer     bool // the text is accepted, then the connection drops before the answer arrives
	failBeforeNextSend bool // the connection drops before the request reaches the API
	nextStatus         int  // the next new key gets this error, stored for later repeats
}

func (api *fakeAPI) RoundTrip(r *http.Request) (*http.Response, error) {
	if api.failBeforeNextSend {
		api.failBeforeNextSend = false
		return nil, errors.New("connection refused")
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) == 0 {
		return nil, errors.New("request without an Idempotency-Key")
	}
	if status, seen := api.answers[key]; seen {
		return &http.Response{StatusCode: status, Header: http.Header{"Idempotent-Replayed": {"true"}}, Body: http.NoBody}, nil
	}
	var body smstunnel.Body
	raw, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, err
	}
	api.texts = append(api.texts, body.Message)
	status := http.StatusOK
	if api.nextStatus != 0 {
		status, api.nextStatus = api.nextStatus, 0
	}
	api.answers[key] = status
	if api.loseNextAnswer {
		api.loseNextAnswer = false
		return nil, errors.New("context deadline exceeded (Client.Timeout exceeded while awaiting headers)")
	}
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: http.NoBody}, nil
}

func setup(t *testing.T, sendOnResolved bool, reminder time.Duration) (*fakeAPI, *endpoint.Endpoint, *alerting.Config) {
	t.Helper()
	api := &fakeAPI{answers: make(map[string]int)}
	client.InjectHTTPClient(&http.Client{Transport: api})
	t.Cleanup(func() { client.InjectHTTPClient(nil) })
	description := "healthcheck failed"
	ep := &endpoint.Endpoint{
		Name: "website",
		URL:  "https://example.org",
		Alerts: []*alert.Alert{{
			Type:                    alert.TypeSMSTunnel,
			Description:             &description,
			FailureThreshold:        2,
			SuccessThreshold:        2,
			SendOnResolved:          &sendOnResolved,
			MinimumReminderInterval: reminder,
		}},
	}
	cfg := &alerting.Config{SMSTunnel: &smstunnel.AlertProvider{
		DefaultConfig: smstunnel.Config{APIKey: "key", To: "+14155550123"},
	}}
	return api, ep, cfg
}

func checkFails(ep *endpoint.Endpoint, cfg *alerting.Config) {
	watchdog.HandleAlerting(ep, &endpoint.Result{Success: false}, cfg)
}

func checkSucceeds(ep *endpoint.Endpoint, cfg *alerting.Config) {
	watchdog.HandleAlerting(ep, &endpoint.Result{Success: true}, cfg)
}

func expectTexts(t *testing.T, api *fakeAPI, expected ...string) {
	t.Helper()
	if strings.Join(api.texts, "\n") != strings.Join(expected, "\n") {
		t.Errorf("expected the texts %q, got %q", expected, api.texts)
	}
}

const (
	triggeredText = "TRIGGERED: website - healthcheck failed"
	resolvedText  = "RESOLVED: website - healthcheck failed"
)

func TestLoop_NormalIncidentSendsOneAlertAndOneRecovery(t *testing.T) {
	api, ep, cfg := setup(t, true, 0)
	checkFails(ep, cfg)
	expectTexts(t, api)
	for range 5 {
		checkFails(ep, cfg)
	}
	expectTexts(t, api, triggeredText)
	checkSucceeds(ep, cfg)
	expectTexts(t, api, triggeredText)
	checkSucceeds(ep, cfg)
	expectTexts(t, api, triggeredText, resolvedText)
	if ep.Alerts[0].Triggered {
		t.Error("the alert should be resolved")
	}
}

func TestLoop_LostAnswerDoesNotSendASecondText(t *testing.T) {
	api, ep, cfg := setup(t, true, 0)
	checkFails(ep, cfg)
	api.loseNextAnswer = true
	checkFails(ep, cfg) // The API accepts the text, but Send gets a timeout
	expectTexts(t, api, triggeredText)
	if ep.Alerts[0].Triggered {
		t.Fatal("Send returned an error, so the alert shouldn't be marked as triggered yet")
	}
	checkFails(ep, cfg) // The watchdog retries on the next check, and the API answers with the first result
	expectTexts(t, api, triggeredText)
	if !ep.Alerts[0].Triggered {
		t.Fatal("the retry succeeded, so the alert should be marked as triggered")
	}
	checkFails(ep, cfg)
	checkSucceeds(ep, cfg)
	checkSucceeds(ep, cfg)
	expectTexts(t, api, triggeredText, resolvedText)
}

func TestLoop_AlertIsDeliveredAfterAFailureBeforeTheAPI(t *testing.T) {
	api, ep, cfg := setup(t, true, 0)
	checkFails(ep, cfg)
	api.failBeforeNextSend = true
	checkFails(ep, cfg) // Nothing reached the API
	expectTexts(t, api)
	checkFails(ep, cfg) // The retry is what delivers the alert
	expectTexts(t, api, triggeredText)
	if !ep.Alerts[0].Triggered {
		t.Error("the alert should be marked as triggered")
	}
}

func TestLoop_NewIncidentSendsANewTextWithoutSendOnResolved(t *testing.T) {
	api, ep, cfg := setup(t, false, 0)
	checkFails(ep, cfg)
	checkFails(ep, cfg)
	expectTexts(t, api, triggeredText)
	checkSucceeds(ep, cfg)
	checkSucceeds(ep, cfg)
	expectTexts(t, api, triggeredText) // send-on-resolved is false, so Send wasn't called
	checkFails(ep, cfg)
	checkFails(ep, cfg)
	expectTexts(t, api, triggeredText, triggeredText)
}

func TestLoop_NewIncidentAfterALostAnswerSendsANewText(t *testing.T) {
	api, ep, cfg := setup(t, true, 0)
	checkFails(ep, cfg)
	api.loseNextAnswer = true
	checkFails(ep, cfg)
	expectTexts(t, api, triggeredText)
	// The endpoint recovers before the retry. The alert never counted as triggered, so there's no recovery text.
	checkSucceeds(ep, cfg)
	checkSucceeds(ep, cfg)
	expectTexts(t, api, triggeredText)
	// This is another incident, so it must not be answered with the first incident's result
	checkFails(ep, cfg)
	checkFails(ep, cfg)
	expectTexts(t, api, triggeredText, triggeredText)
}

func TestLoop_LostReminderIsNotSentTwiceAndTheNextOneIs(t *testing.T) {
	api, ep, cfg := setup(t, true, 5*time.Minute)
	checkFails(ep, cfg)
	checkFails(ep, cfg)
	expectTexts(t, api, triggeredText)
	ep.LastReminderSent = time.Now().Add(-time.Hour) // The reminder is due
	api.loseNextAnswer = true
	checkFails(ep, cfg)
	expectTexts(t, api, triggeredText, triggeredText)
	// LastReminderSent didn't move, so the reminder is still due and goes through the same key
	checkFails(ep, cfg)
	expectTexts(t, api, triggeredText, triggeredText)
	if time.Since(ep.LastReminderSent) > time.Minute {
		t.Fatal("the retried reminder should've counted as sent")
	}
	checkFails(ep, cfg) // Not due anymore
	expectTexts(t, api, triggeredText, triggeredText)
	ep.LastReminderSent = time.Now().Add(-2 * time.Hour) // The next reminder is due
	checkFails(ep, cfg)
	expectTexts(t, api, triggeredText, triggeredText, triggeredText)
}

func TestLoop_StoredServerErrorIsNotSentAgain(t *testing.T) {
	api, ep, cfg := setup(t, true, 0)
	checkFails(ep, cfg)
	api.nextStatus = http.StatusInternalServerError // The text may have been created before the error
	checkFails(ep, cfg)
	expectTexts(t, api, triggeredText)
	for range 3 {
		checkFails(ep, cfg)
	}
	expectTexts(t, api, triggeredText)
	if ep.Alerts[0].Triggered {
		t.Error("the alert can't be marked as triggered while the API answers with an error")
	}
}

func TestLoop_LostRecoveryAnswerIsNotRetried(t *testing.T) {
	api, ep, cfg := setup(t, true, 0)
	checkFails(ep, cfg)
	checkFails(ep, cfg)
	checkSucceeds(ep, cfg)
	api.loseNextAnswer = true
	checkSucceeds(ep, cfg)
	expectTexts(t, api, triggeredText, resolvedText)
	for range 3 {
		checkSucceeds(ep, cfg)
	}
	expectTexts(t, api, triggeredText, resolvedText)
}

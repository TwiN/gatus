package custom

import (
	"io"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/test"
)

func TestAlertProvider_buildHTTPRequestWithIncidentID(t *testing.T) {
	alertProvider := &AlertProvider{
		DefaultConfig: Config{
			URL:  "https://example.com/incidents/[INCIDENT_ID]",
			Body: `{"id":"[INCIDENT_ID]","state":"[ALERT_TRIGGERED_OR_RESOLVED]"}`,
		},
	}
	request := alertProvider.buildHTTPRequest(
		&alertProvider.DefaultConfig,
		&endpoint.Endpoint{Name: "endpoint-name", Group: "endpoint-group", URL: "https://example.com"},
		&alert.Alert{ResolveKey: "1b671a64-40d5-491e-99b0-da01ff1f3341"},
		&endpoint.Result{Errors: []string{}},
		false,
	)
	if expected := "https://example.com/incidents/1b671a64-40d5-491e-99b0-da01ff1f3341"; request.URL.String() != expected {
		t.Error("expected URL to be", expected, "got", request.URL.String())
	}
	body, _ := io.ReadAll(request.Body)
	if expected := `{"id":"1b671a64-40d5-491e-99b0-da01ff1f3341","state":"TRIGGERED"}`; string(body) != expected {
		t.Error("expected body to be", expected, "got", string(body))
	}
}

func TestAlertProvider_SendKeepsTheSameIncidentIDUntilTheAlertIsCleared(t *testing.T) {
	defer client.InjectHTTPClient(nil)
	var bodies []string
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
	})})
	provider := AlertProvider{DefaultConfig: Config{URL: "https://example.com", Method: http.MethodPost, Body: "[ALERT_TRIGGERED_OR_RESOLVED] [INCIDENT_ID]"}}
	a := &alert.Alert{SuccessThreshold: 2, FailureThreshold: 3}
	ep := &endpoint.Endpoint{Name: "endpoint-name"}
	for _, resolved := range []bool{false, false, true} {
		if err := provider.Send(ep, a, &endpoint.Result{}, resolved); err != nil {
			t.Fatal("expected no error, got", err.Error())
		}
	}
	if len(a.ResolveKey) == 0 {
		t.Fatal("expected the alert's ResolveKey to be set")
	}
	for i, state := range []string{"TRIGGERED", "TRIGGERED", "RESOLVED"} {
		if expected := state + " " + a.ResolveKey; bodies[i] != expected {
			t.Errorf("expected body %d to be %q, got %q", i, expected, bodies[i])
		}
	}
	// An incident id that was already set (i.e. loaded from the storage after a restart) must be used as is
	a = &alert.Alert{SuccessThreshold: 2, FailureThreshold: 3, ResolveKey: "persisted-id"}
	if err := provider.Send(ep, a, &endpoint.Result{}, true); err != nil {
		t.Fatal("expected no error, got", err.Error())
	}
	if expected := "RESOLVED persisted-id"; bodies[3] != expected {
		t.Errorf("expected body to be %q, got %q", expected, bodies[3])
	}
}

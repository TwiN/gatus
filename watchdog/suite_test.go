package watchdog

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/test"
	"golang.org/x/sync/semaphore"
)

func TestExecuteSuite_AlertsUseExecutedEndpoint(t *testing.T) {
	t.Setenv("MOCK_ALERT_PROVIDER", "true")
	var requests []string
	client.InjectHTTPClient(&http.Client{Transport: test.MockRoundTripper(func(r *http.Request) *http.Response {
		requests = append(requests, r.URL.Path)
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok"))}
	})})
	t.Cleanup(func() { client.InjectHTTPClient(nil) })
	oldSemaphore, oldContext := monitoringSemaphore, ctx
	monitoringSemaphore = semaphore.NewWeighted(1)
	ctx = context.Background()
	t.Cleanup(func() {
		monitoringSemaphore = oldSemaphore
		ctx = oldContext
	})
	enabled := true
	skippedAlert := &alert.Alert{
		Type: alert.TypeCustom, Enabled: &enabled, FailureThreshold: 1,
		SuccessThreshold: 1, Triggered: true, SendOnResolved: &enabled,
	}
	first := &endpoint.Endpoint{
		Name: "first", URL: "http://example.com/first",
		Conditions: []endpoint.Condition{"[STATUS] == 500"},
	}
	skipped := &endpoint.Endpoint{
		Name: "skipped", URL: "http://example.com/skipped",
		Conditions: []endpoint.Condition{"[STATUS] == 200"},
		Alerts:     []*alert.Alert{skippedAlert}, NumberOfFailuresInARow: 3,
	}
	cleanup := &endpoint.Endpoint{
		Name: "cleanup", URL: "http://example.com/cleanup", AlwaysRun: true,
		Conditions: []endpoint.Condition{"[STATUS] == 200"},
	}
	s := &suite.Suite{Name: "alert-association", Endpoints: []*endpoint.Endpoint{first, skipped, cleanup}}
	if err := s.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("suite validation failed: %v", err)
	}
	cfg := &config.Config{
		Alerting: &alerting.Config{Custom: &custom.AlertProvider{
			DefaultConfig: custom.Config{URL: "http://example.com/alert", Method: http.MethodGet},
		}},
		Maintenance: maintenance.GetDefaultConfig(),
	}
	executeSuite(s, cfg, nil)
	if !reflect.DeepEqual(requests, []string{"/first", "/cleanup"}) {
		t.Errorf("expected only first and cleanup requests, got %v", requests)
	}
	if first.NumberOfFailuresInARow != 1 || first.NumberOfSuccessesInARow != 0 {
		t.Errorf("first endpoint should receive its failure result, got failures=%d successes=%d", first.NumberOfFailuresInARow, first.NumberOfSuccessesInARow)
	}
	verify(t, skipped, 3, 0, true, "skipped endpoint's incident and counters should remain unchanged")
	if cleanup.NumberOfSuccessesInARow != 1 || cleanup.NumberOfFailuresInARow != 0 {
		t.Errorf("cleanup endpoint should receive its success result, got failures=%d successes=%d", cleanup.NumberOfFailuresInARow, cleanup.NumberOfSuccessesInARow)
	}
}

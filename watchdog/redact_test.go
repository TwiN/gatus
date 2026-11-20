package watchdog

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/logr"
)

const (
	secretPath  = "T0KEN-IN-THE-PATH"
	secretQuery = "T0KEN-IN-THE-QUERY"
)

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

type syncBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.Write(p)
}

func (b *syncBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.String()
}

// TestHandleAlertingDoesNotLogTheURLOfAFailedAlert goes through HandleAlerting with a real provider whose request fails at
// the transport, which is what produces a *url.Error that contains the URL, and reads what ends up in the log.
func TestHandleAlertingDoesNotLogTheURLOfAFailedAlert(t *testing.T) {
	_ = os.Unsetenv("MOCK_ALERT_PROVIDER")
	output := &syncBuffer{}
	logr.SetOutput(output)
	defer logr.SetOutput(os.Stdout)
	client.InjectHTTPClient(&http.Client{Transport: failingTransport{}})
	defer client.InjectHTTPClient(nil)
	alertingConfig := &alerting.Config{Custom: &custom.AlertProvider{DefaultConfig: custom.Config{
		URL:    "https://hooks.example.com/services/" + secretPath + "?access_token=" + secretQuery,
		Method: http.MethodPost,
	}}}
	enabled := true
	ep := &endpoint.Endpoint{
		URL:    "https://example.com",
		Alerts: []*alert.Alert{{Type: alert.TypeCustom, Enabled: &enabled, FailureThreshold: 1, SuccessThreshold: 1, SendOnResolved: &enabled}},
	}
	HandleAlerting(ep, &endpoint.Result{Success: false}, alertingConfig)
	// The alert was never triggered, because it could not be sent. Force the resolution to exercise its error path too.
	ep.Alerts[0].Triggered = true
	HandleAlerting(ep, &endpoint.Result{Success: true}, alertingConfig)
	logged := output.String()
	if strings.Count(logged, "Failed to send an alert") != 2 {
		t.Fatalf("expected both failures to be logged, got: %s", logged)
	}
	for _, secret := range []string{secretPath, secretQuery} {
		if strings.Contains(logged, secret) {
			t.Errorf("the log contains %q: %s", secret, logged)
		}
	}
	if !strings.Contains(logged, "hooks.example.com") {
		t.Errorf("expected the log to still say which host failed: %s", logged)
	}
}

func TestRedactError(t *testing.T) {
	scenarios := []struct {
		name     string
		err      error
		expected string
	}{
		{name: "nil", err: nil, expected: ""},
		{name: "not-a-url-error", err: errors.New("boom"), expected: "boom"},
		{
			name:     "url-error",
			err:      &url.Error{Op: "Post", URL: "https://user:pass@hooks.example.com/a/b?token=x#frag", Err: errors.New("timeout")},
			expected: `Post "https://hooks.example.com/<redacted>": timeout`,
		},
		{
			name:     "wrapped-url-error",
			err:      fmt.Errorf("sending: %w", &url.Error{Op: "Get", URL: "http://host:8080/secret", Err: errors.New("refused")}),
			expected: `sending: Get "http://host:8080/<redacted>": refused`,
		},
		{
			name:     "url-error-that-is-not-wrapped",
			err:      fmt.Errorf("sending failed: %v", &url.Error{Op: "Post", URL: "https://hooks.example.com/services/T0KEN?x=y", Err: errors.New("timeout")}),
			expected: `sending failed: Post "https://hooks.example.com/<redacted>": timeout`,
		},
		{
			name:     "url-in-plain-text",
			err:      errors.New("call failed (see https://api.example.com:8443/v1/hook/T0KEN)."),
			expected: "call failed (see https://api.example.com:8443/<redacted>).",
		},
		{
			name:     "unparseable-url",
			err:      &url.Error{Op: "Get", URL: "://secret", Err: errors.New("bad")},
			expected: `Get "<redacted>": bad`,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			if got := redactError(scenario.err); got != scenario.expected {
				t.Errorf("expected %q, got %q", scenario.expected, got)
			}
		})
	}
}

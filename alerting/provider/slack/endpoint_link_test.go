package slack

import (
	"encoding/json"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/ui"
	"testing"
)

func TestNotificationEndpointLink(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name, url      string
		hidden, linked bool
	}{
		{"public", "https://example.com/health", false, true},
		{"hidden", "https://example.com/health", true, false},
		{"credentials", "https://user:secret@example.com/health", false, false},
		{"query", "https://example.com/health?token=secret", false, false},
		{"non-http", "tcp://example.com:443", false, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			for _, resolved := range []bool{false, true} {
				ep := &endpoint.Endpoint{Name: "demo", URL: scenario.url, UIConfig: &ui.Config{HideURL: scenario.hidden}}
				raw := (&AlertProvider{}).buildRequestBody(&Config{}, ep, &alert.Alert{}, &endpoint.Result{}, resolved)
				var body Body
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Fatal(err)
				}
				actual := ""
				if len(body.Attachments[0].Fields) > 0 {
					actual = body.Attachments[0].Fields[0].Value
				}
				expected := ""
				if scenario.linked {
					expected = "<" + scenario.url + "|Open endpoint>"
				}
				if actual != expected {
					t.Fatalf("resolved=%v: expected %q, got %q", resolved, expected, actual)
				}
			}
		})
	}
}

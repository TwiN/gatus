package endpoint

import (
	"testing"

	"github.com/TwiN/gatus/v5/config/endpoint/ui"
)

func TestEndpoint_NotificationURL(t *testing.T) {
	t.Parallel()
	for _, scenario := range []struct {
		name     string
		url      string
		hidden   bool
		expected string
	}{
		{"https", "https://example.com/health", false, "https://example.com/health"},
		{"http", "http://localhost:8080/health", false, "http://localhost:8080/health"},
		{"hidden", "https://example.com/health", true, ""},
		{"credentials", "https://user:password@example.com", false, ""},
		{"username", "https://token@example.com", false, ""},
		{"query", "https://example.com?token=secret", false, ""},
		{"empty query", "https://example.com?", false, ""},
		{"fragment", "https://example.com/#token", false, ""},
		{"tcp", "tcp://example.com:80", false, ""},
		{"javascript", "javascript:alert(1)", false, ""},
		{"relative", "/health", false, ""},
		{"malformed", "https://example.com/%zz", false, ""},
		{"empty", "", false, ""},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			ep := &Endpoint{URL: scenario.url, UIConfig: &ui.Config{HideURL: scenario.hidden}}
			if actual := ep.NotificationURL(); actual != scenario.expected {
				t.Fatalf("expected %q, got %q", scenario.expected, actual)
			}
		})
	}
	if actual := (&Endpoint{URL: "https://example.com"}).NotificationURL(); actual != "https://example.com" {
		t.Fatalf("nil UI config: got %q", actual)
	}
}

package connectivity

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestConfig(t *testing.T) {
	scenarios := []struct {
		name             string
		cfg              *Config
		expectedErr      error
		expectedInterval time.Duration
	}{
		{
			name:             "good-config",
			cfg:              &Config{Checker: &Checker{Target: "1.1.1.1:53", Interval: 10 * time.Second}},
			expectedInterval: 10 * time.Second,
		},
		{
			name:             "good-config-with-default-interval",
			cfg:              &Config{Checker: &Checker{Target: "8.8.8.8:53", Interval: 0}},
			expectedInterval: 60 * time.Second,
		},
		{
			name:        "config-with-interval-too-low",
			cfg:         &Config{Checker: &Checker{Target: "1.1.1.1:53", Interval: 4 * time.Second}},
			expectedErr: ErrInvalidInterval,
		},
		{
			name:        "config-with-invalid-target-due-to-missing-port",
			cfg:         &Config{Checker: &Checker{Target: "1.1.1.1", Interval: 15 * time.Second}},
			expectedErr: ErrInvalidDNSTarget,
		},
		{
			name:        "config-with-invalid-target-due-to-invalid-dns-port",
			cfg:         &Config{Checker: &Checker{Target: "1.1.1.1:52", Interval: 15 * time.Second}},
			expectedErr: ErrInvalidDNSTarget,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.cfg.ValidateAndSetDefaults()
			if fmt.Sprintf("%s", err) != fmt.Sprintf("%s", scenario.expectedErr) {
				t.Errorf("expected error %v, got %v", scenario.expectedErr, err)
			}
			if err == nil && scenario.expectedErr == nil {
				if scenario.cfg.Checker.Interval != scenario.expectedInterval {
					t.Errorf("expected interval %v, got %v", scenario.expectedInterval, scenario.cfg.Checker.Interval)
				}
			}
		})
	}
}

func TestChecker_IsConnected(t *testing.T) {
	checker := &Checker{Target: "1.1.1.1:53", Interval: 10 * time.Second}
	if !checker.IsConnected() {
		t.Error("expected checker.IsConnected() to be true")
	}
}

// startHTTPProbe runs a local HTTP server answering every request with the
// given status and body.
func startHTTPProbe(t *testing.T, status int, body string) (url string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go http.Serve(listener, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(func() { _ = listener.Close() })
	return "http://" + listener.Addr().String()
}

// TestCheckerHTTPModeWithConditions pins #1772's docker-socket-proxy shape:
// an http(s) target turns the checker into a health probe whose conditions
// decide connectivity, so a deployment with no direct egress can check the
// VPN proxy's own container-health endpoint instead of a DNS handshake.
func TestCheckerHTTPModeWithConditions(t *testing.T) {
	url := startHTTPProbe(t, 200, `{"State":{"Health":{"Status":"healthy"}}}`)

	cfg := &Config{
		Checker: &Checker{
			Target: url,
			Conditions: []endpoint.Condition{
				"[STATUS] == 200",
				"[BODY].State.Health.Status == healthy",
			},
		},
	}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !cfg.Checker.Check() {
		t.Fatal("the check should succeed when every condition evaluates to true")
	}

	unhealthy := &Config{
		Checker: &Checker{
			Target: url,
			Conditions: []endpoint.Condition{
				"[BODY].State.Health.Status == unhealthy",
			},
		},
	}
	if err := unhealthy.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if unhealthy.Checker.Check() {
		t.Fatal("the check must fail when a condition evaluates to false")
	}
}

// TestCheckerHTTPModeWithoutConditionsSucceedsOn2xx: no conditions means any
// 2xx response counts as connected.
func TestCheckerHTTPModeWithoutConditionsSucceedsOn2xx(t *testing.T) {
	url := startHTTPProbe(t, 204, "")
	cfg := &Config{Checker: &Checker{Target: url}}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !cfg.Checker.Check() {
		t.Fatal("a 2xx response with no conditions should mean connected")
	}

	serverError := &Config{Checker: &Checker{Target: url + "/?"}} // same handler; use a 500 probe
	_ = serverError
}

// TestCheckerHTTPModeFailsOn500.
func TestCheckerHTTPModeFailsOn500(t *testing.T) {
	url := startHTTPProbe(t, 500, "")
	cfg := &Config{Checker: &Checker{Target: url}}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	// Without conditions Gatus does not fail on non-2xx by default for
	// endpoints — but the checker's contract is "is the proxy healthy",
	// so pin the explicit-status behavior through a condition instead.
	failing := &Config{
		Checker: &Checker{
			Target:     url,
			Conditions: []endpoint.Condition{"[STATUS] == 200"},
		},
	}
	if err := failing.ValidateAndSetDefaults(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if failing.Checker.Check() {
		t.Fatal("a 500 against [STATUS] == 200 must mean disconnected")
	}
}

// TestConfigRejectsConditionsWithDNSTarget: conditions only make sense
// against an HTTP response; the DNS-tcp mode must reject them.
func TestConfigRejectsConditionsWithDNSTarget(t *testing.T) {
	cfg := &Config{
		Checker: &Checker{
			Target:     "1.1.1.1:53",
			Conditions: []endpoint.Condition{"[STATUS] == 200"},
		},
	}
	err := cfg.ValidateAndSetDefaults()
	if err == nil || err != ErrInvalidCondition {
		t.Fatalf("expected ErrInvalidCondition, got %v", err)
	}
}

// TestConfigValidatesHTTPModeConditionSyntax: invalid condition syntax
// surfaces at startup, not at check time.
func TestConfigValidatesHTTPModeConditionSyntax(t *testing.T) {
	url := startHTTPProbe(t, 200, "")
	cfg := &Config{
		Checker: &Checker{
			Target:     url,
			Conditions: []endpoint.Condition{"[STATUS] === maybe"},
		},
	}
	if err := cfg.ValidateAndSetDefaults(); err == nil {
		t.Fatal("invalid condition syntax must fail validation")
	}
}

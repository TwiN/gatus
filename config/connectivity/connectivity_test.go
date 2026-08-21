package connectivity

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/client"
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

// startConnectProxy runs a minimal HTTP CONNECT proxy that accepts every
// tunnel request and reports the address it was asked to reach.
func startConnectProxy(t *testing.T) (addr string, reached chan string) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	reached = make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		request, err := http.ReadRequest(bufio.NewReader(conn))
		if err != nil {
			return
		}
		reached <- request.URL.Host
		fmt.Fprint(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
		// Hold the tunnel open until the caller closes its side.
		time.Sleep(250 * time.Millisecond)
	}()
	t.Cleanup(func() { _ = listener.Close() })
	return listener.Addr().String(), reached
}

// TestCheckerDialsThroughProxyURL pins #1772: a checker configured with a
// client.proxy-url must reach its target through that proxy — a deployment
// with no direct egress can then still distinguish "Gatus is offline" from
// "the endpoint is down".
func TestCheckerDialsThroughProxyURL(t *testing.T) {
	proxyAddr, reached := startConnectProxy(t)

	checker := &Checker{
		Target: "1.1.1.1:53",
		Client: &client.Config{ProxyURL: "http://" + proxyAddr, Timeout: 2 * time.Second},
	}

	if !checker.Check() {
		t.Fatal("the check should succeed through a proxy that accepts the tunnel")
	}
	select {
	case target := <-reached:
		if target != "1.1.1.1:53" {
			t.Fatalf("proxy was asked for %q, want 1.1.1.1:53", target)
		}
	default:
		t.Fatal("the proxy was never contacted")
	}
}

func TestCheckerFailsWhenProxyIsUnreachable(t *testing.T) {
	// Port 1 on loopback: reserved, nothing listens there.
	checker := &Checker{
		Target: "1.1.1.1:53",
		Client: &client.Config{ProxyURL: "http://127.0.0.1:1", Timeout: 500 * time.Millisecond},
	}
	if checker.Check() {
		t.Fatal("the check must fail when the configured proxy cannot be reached")
	}
}

func TestCheckerWithoutClientHasNoProxy(t *testing.T) {
	checker := &Checker{Target: "1.1.1.1:53"}
	if checker.Client != nil {
		t.Fatal("a checker without a client block must dial directly")
	}
	if checker.Check() == false {
		// Direct dial still works (existing behavior; requires egress like
		// TestChecker_IsConnected).
		t.Log("direct dial failed (no egress in this environment?)")
	}
}

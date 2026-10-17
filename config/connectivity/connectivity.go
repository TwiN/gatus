package connectivity

import (
	"errors"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/ui"
)

var (
	ErrInvalidInterval  = errors.New("connectivity.checker.interval must be 5s or higher")
	ErrInvalidDNSTarget = errors.New("connectivity.checker.target must be suffixed with :53")
	ErrInvalidCondition = errors.New("connectivity.checker.conditions is only supported with an http(s) target")
)

// Config is the configuration for the connectivity checker.
type Config struct {
	Checker *Checker `yaml:"checker,omitempty"`
}

func (c *Config) ValidateAndSetDefaults() error {
	if c.Checker != nil {
		if c.Checker.Interval == 0 {
			c.Checker.Interval = 60 * time.Second
		} else if c.Checker.Interval < 5*time.Second {
			return ErrInvalidInterval
		}
		if c.Checker.isHTTPTarget() {
			// HTTP(S) mode: the target is a URL whose response — and,
			// optionally, the conditions evaluated against it — decides
			// connectivity (#1772). Deployments behind a VPN proxy or a
			// docker-socket-proxy can point this at the proxy's own health
			// endpoint.
			// The probe carries the checker's conditions when present; a
			// conditionless HTTP probe falls back to a bare [STATUS] == 2xx
			// check so the internal endpoint always satisfies the endpoint
			// contract (at least one condition).
			conditions := c.Checker.Conditions
			if len(conditions) == 0 {
				conditions = []endpoint.Condition{
					"[STATUS] < 300",
					"[STATUS] >= 200",
				}
			}
			probe := &endpoint.Endpoint{
				Name:         "connectivity-checker",
				Group:        "internal",
				URL:          c.Checker.Target,
				Interval:     10 * time.Minute,
				Conditions:   conditions,
				ClientConfig: c.Checker.Client,
			}
			if probe.ClientConfig == nil {
				probe.ClientConfig = client.GetDefaultConfig()
			}
			if probe.UIConfig == nil {
				probe.UIConfig = ui.GetDefaultConfig()
			}
			if err := probe.ValidateAndSetDefaults(); err != nil {
				return err
			}
			c.Checker.probe = probe
		} else {
			if !strings.HasSuffix(c.Checker.Target, ":53") {
				return ErrInvalidDNSTarget
			}
			if len(c.Checker.Conditions) > 0 {
				return ErrInvalidCondition
			}
			if c.Checker.Client != nil {
				if err := c.Checker.Client.ValidateAndSetDefaults(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Checker is the configuration for making sure Gatus has access to the internet.
type Checker struct {
	Target   string        `yaml:"target"` // e.g. 1.1.1.1:53, or an http(s) URL in HTTP mode
	Interval time.Duration `yaml:"interval,omitempty"`
	// Conditions are only supported with an http(s) target; they are evaluated
	// against the target's response exactly like endpoint conditions
	// (e.g. "[STATUS] == 200"). Without conditions, HTTP mode succeeds on any
	// 2xx response (#1772).
	Conditions []endpoint.Condition `yaml:"conditions,omitempty"`
	// Client is the client configuration used by the checker. In HTTP mode it
	// is the probe's client; in DNS mode only proxy-url has an effect on the
	// raw TCP dial (#1772).
	Client *client.Config `yaml:"client,omitempty"`

	isConnected bool
	lastCheck   time.Time
	probe       *endpoint.Endpoint
}

func (c *Checker) isHTTPTarget() bool {
	return strings.HasPrefix(c.Target, "http://") || strings.HasPrefix(c.Target, "https://")
}

func (c *Checker) Check() bool {
	if c.probe != nil {
		// HTTP(S) mode: connectivity is the probe's health — the request
		// succeeded and every configured condition evaluated to true.
		return c.probe.EvaluateHealth().Success
	}
	cfg := &client.Config{Timeout: 5 * time.Second}
	if c.Client != nil {
		if c.Client.ProxyURL != "" {
			cfg.ProxyURL = c.Client.ProxyURL
		}
		if c.Client.Timeout > 0 {
			cfg.Timeout = c.Client.Timeout
		}
	}
	connected, _ := client.CanCreateNetworkConnection("tcp", c.Target, "", cfg)
	return connected
}

func (c *Checker) IsConnected() bool {
	if now := time.Now(); now.After(c.lastCheck.Add(c.Interval)) {
		c.lastCheck, c.isConnected = now, c.Check()
	}
	return c.isConnected
}

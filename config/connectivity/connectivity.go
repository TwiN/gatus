package connectivity

import (
	"errors"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/client"
)

var (
	ErrInvalidInterval  = errors.New("connectivity.checker.interval must be 5s or higher")
	ErrInvalidDNSTarget = errors.New("connectivity.checker.target must be suffixed with :53")
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
		if !strings.HasSuffix(c.Checker.Target, ":53") {
			return ErrInvalidDNSTarget
		}
		if c.Checker.Client != nil {
			if err := c.Checker.Client.ValidateAndSetDefaults(); err != nil {
				return err
			}
		}
	}
	return nil
}

// Checker is the configuration for making sure Gatus has access to the internet.
type Checker struct {
	Target string `yaml:"target"` // e.g. 1.1.1.1:53
	Interval time.Duration `yaml:"interval,omitempty"`
	// Client is the client configuration used by the checker. Only the
	// proxy-url option has an effect on the checker's raw TCP dial; deployments
	// where Gatus itself has no direct egress can point the check at the same
	// proxy their endpoints already use (#1772).
	Client *client.Config `yaml:"client,omitempty"`

	isConnected bool
	lastCheck   time.Time
}

func (c *Checker) Check() bool {
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

package heartbeat

import "time"

// Config used to check if the external endpoint has received new results when it should have.
// This configuration is used to trigger alerts when an external endpoint has no new results for a defined period of time
type Config struct {
	// Interval is the time interval at which Gatus verifies whether the external endpoint has received new results
	// If no new result is received within the interval, the endpoint is marked as failed and alerts are triggered
	Interval time.Duration `yaml:"interval"`
	// GracePeriod is an additional time buffer added to the interval before marking the endpoint as failed
	// This allows for small delays in webhook delivery without triggering false alerts
	// Optional field - if not specified, defaults to 0 (no grace period)
	GracePeriod time.Duration `yaml:"grace-period,omitempty"`
}

// GetEffectiveInterval returns the total time to wait before considering the heartbeat failed
// This is the sum of Interval and GracePeriod
func (c *Config) GetEffectiveInterval() time.Duration {
	return c.Interval + c.GracePeriod
}

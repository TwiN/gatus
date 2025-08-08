package heartbeat

import "time"

// Config used to check if the external endpoint has received new results when it should have.
// This configuration is used to trigger alerts when an external endpoint has no new results for a defined period of time
type Config struct {
	// Interval is the time interval at which Gatus verifies whether the external endpoint has received new results
	// If no new result is received within the interval, the endpoint is marked as failed and alerts are triggered
	Interval time.Duration `yaml:"interval"`
}

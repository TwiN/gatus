package core

// Uptime is the struct that contains the relevant data for calculating the uptime as well as the uptime itself
// and some other statistics
type Uptime struct {
	LastSevenDays       float64 `json:"7d"`  // Uptime percentage over the past 7 days
	LastTwentyFourHours float64 `json:"24h"` // Uptime percentage over the past 24 hours
	LastHour            float64 `json:"1h"`  // Uptime percentage over the past hour

	// SuccessfulExecutionsPerHour is a map containing the number of successes (value)
	// for every hourly unix timestamps (key)
	// Deprecated
	SuccessfulExecutionsPerHour map[int64]uint64 `json:"-"`

	// TotalExecutionsPerHour is a map containing the total number of checks (value)
	// for every hourly unix timestamps (key)
	// Deprecated
	TotalExecutionsPerHour map[int64]uint64 `json:"-"`

	// HourlyStatistics is a map containing metrics collected (value) for every hourly unix timestamps (key)
	HourlyStatistics map[int64]*HourlyUptimeStatistics `json:"-"`
}

// HourlyUptimeStatistics is a struct containing all metrics collected over the course of an hour
type HourlyUptimeStatistics struct {
	TotalExecutions             uint64 // Total number of checks
	SuccessfulExecutions        uint64 // Number of successful executions
	TotalExecutionsResponseTime uint64 // Total response time for all executions
}

// NewUptime creates a new Uptime
func NewUptime() *Uptime {
	return &Uptime{
		HourlyStatistics: make(map[int64]*HourlyUptimeStatistics),
	}
}

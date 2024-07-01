package endpoint

// Uptime is the struct that contains the relevant data for calculating the uptime as well as the uptime itself
// and some other statistics
type Uptime struct {
	// HourlyStatistics is a map containing metrics collected (value) for every hourly unix timestamps (key)
	//
	// Used only if the storage type is memory
	HourlyStatistics map[int64]*HourlyUptimeStatistics `json:"-"`
}

// HourlyUptimeStatistics is a struct containing all metrics collected over the course of an hour
type HourlyUptimeStatistics struct {
	TotalExecutions             uint64 // Total number of checks
	SuccessfulExecutions        uint64 // Number of successful executions
	TotalExecutionsResponseTime uint64 // Total response time for all executions in milliseconds
}

// NewUptime creates a new Uptime
func NewUptime() *Uptime {
	return &Uptime{
		HourlyStatistics: make(map[int64]*HourlyUptimeStatistics),
	}
}

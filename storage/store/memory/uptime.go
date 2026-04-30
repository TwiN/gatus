package memory

import (
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
)

const (
	uptimeCleanUpThreshold = 92 * 24
	uptimeRetention        = 90 * 24 * time.Hour
)

// processUptimeAfterResult processes the result by extracting the relevant from the result and recalculating the uptime
// if necessary
func processUptimeAfterResult(uptime *endpoint.Uptime, result *endpoint.Result) {
	if uptime.HourlyStatistics == nil {
		uptime.HourlyStatistics = make(map[int64]*endpoint.HourlyUptimeStatistics)
	}
	unixTimestampFlooredAtHour := result.Timestamp.Truncate(time.Hour).Unix()
	hourlyStats, _ := uptime.HourlyStatistics[unixTimestampFlooredAtHour]
	if hourlyStats == nil {
		hourlyStats = &endpoint.HourlyUptimeStatistics{}
		uptime.HourlyStatistics[unixTimestampFlooredAtHour] = hourlyStats
	}
	if result.Success {
		hourlyStats.SuccessfulExecutions++
	}
	hourlyStats.TotalExecutions++
	hourlyStats.TotalExecutionsResponseTime += uint64(result.Duration.Milliseconds())
	// Clean up only when we're starting to have too many useless keys
	// Note that this is only triggered when there are more entries than there should be after
	// 92 days, despite the fact that we are deleting everything that's older than 90 days.
	// This is to prevent re-iterating on every `processUptimeAfterResult` as soon as the uptime has been logged for 90 days.
	if len(uptime.HourlyStatistics) > uptimeCleanUpThreshold {
		cutoff := time.Now().Add(-(uptimeRetention + time.Hour)).Unix()
		for hourlyUnixTimestamp := range uptime.HourlyStatistics {
			if cutoff > hourlyUnixTimestamp {
				delete(uptime.HourlyStatistics, hourlyUnixTimestamp)
			}
		}
	}
}

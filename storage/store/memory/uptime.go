package memory

import (
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
)

const (
	defaultUptimeRetention = 30 * 24 * time.Hour // Default minimum duration of uptime data to keep (configurable via storage.uptime-retention)
	uptimeCleanUpBuffer    = 2 * 24 * time.Hour  // Cleanup is triggered for uptime data older than retention + this buffer
)

// uptimeRetention is the minimum duration that uptime data must be kept; uptimeCleanUpThreshold is the maximum number
// of hourly entries to keep before triggering a cleanup (i.e. retention + buffer, expressed in hours). Both are
// configurable via SetUptimeRetention; the defaults preserve the historical 30-day behaviour.
var (
	uptimeRetention        = defaultUptimeRetention
	uptimeCleanUpThreshold = int((defaultUptimeRetention + uptimeCleanUpBuffer).Hours())
)

// SetUptimeRetention configures how long uptime data is retained. A value <= 0 resets it to the default (30 days).
// It must be called once before the store starts processing results (e.g. from storage.Initialize).
func SetUptimeRetention(retention time.Duration) {
	if retention <= 0 {
		retention = defaultUptimeRetention
	}
	uptimeRetention = retention
	uptimeCleanUpThreshold = int((retention + uptimeCleanUpBuffer).Hours())
}

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
	// 32 days, despite the fact that we are deleting everything that's older than 30 days.
	// This is to prevent re-iterating on every `processUptimeAfterResult` as soon as the uptime has been logged for 30 days.
	if len(uptime.HourlyStatistics) > uptimeCleanUpThreshold {
		sevenDaysAgo := time.Now().Add(-(uptimeRetention + time.Hour)).Unix()
		for hourlyUnixTimestamp := range uptime.HourlyStatistics {
			if sevenDaysAgo > hourlyUnixTimestamp {
				delete(uptime.HourlyStatistics, hourlyUnixTimestamp)
			}
		}
	}
}

package memory

import (
	"log"
	"time"

	"github.com/TwinProduction/gatus/core"
)

const (
	numberOfHoursInTenDays = 10 * 24
	sevenDays              = 7 * 24 * time.Hour
)

// processUptimeAfterResult processes the result by extracting the relevant from the result and recalculating the uptime
// if necessary
func processUptimeAfterResult(uptime *core.Uptime, result *core.Result) {
	// XXX: Remove this on v3.0.0
	if len(uptime.SuccessfulExecutionsPerHour) != 0 || len(uptime.TotalExecutionsPerHour) != 0 {
		migrateUptimeToHourlyStatistics(uptime)
	}
	if uptime.HourlyStatistics == nil {
		uptime.HourlyStatistics = make(map[int64]*core.HourlyUptimeStatistics)
	}
	unixTimestampFlooredAtHour := result.Timestamp.Truncate(time.Hour).Unix()
	hourlyStats, _ := uptime.HourlyStatistics[unixTimestampFlooredAtHour]
	if hourlyStats == nil {
		hourlyStats = &core.HourlyUptimeStatistics{}
		uptime.HourlyStatistics[unixTimestampFlooredAtHour] = hourlyStats
	}
	if result.Success {
		hourlyStats.SuccessfulExecutions++
	}
	hourlyStats.TotalExecutions++
	hourlyStats.TotalExecutionsResponseTime += uint64(result.Duration.Milliseconds())
	// Clean up only when we're starting to have too many useless keys
	// Note that this is only triggered when there are more entries than there should be after
	// 10 days, despite the fact that we are deleting everything that's older than 7 days.
	// This is to prevent re-iterating on every `processUptimeAfterResult` as soon as the uptime has been logged for 7 days.
	if len(uptime.HourlyStatistics) > numberOfHoursInTenDays {
		sevenDaysAgo := time.Now().Add(-(sevenDays + time.Hour)).Unix()
		for hourlyUnixTimestamp := range uptime.HourlyStatistics {
			if sevenDaysAgo > hourlyUnixTimestamp {
				delete(uptime.HourlyStatistics, hourlyUnixTimestamp)
			}
		}
	}
}

// XXX: Remove this on v3.0.0
// Deprecated
func migrateUptimeToHourlyStatistics(uptime *core.Uptime) {
	log.Println("[migrateUptimeToHourlyStatistics] Got", len(uptime.SuccessfulExecutionsPerHour), "entries for successful executions and", len(uptime.TotalExecutionsPerHour), "entries for total executions")
	uptime.HourlyStatistics = make(map[int64]*core.HourlyUptimeStatistics)
	for hourlyUnixTimestamp, totalExecutions := range uptime.TotalExecutionsPerHour {
		if totalExecutions == 0 {
			log.Println("[migrateUptimeToHourlyStatistics] Skipping entry at", hourlyUnixTimestamp, "because total number of executions is 0")
			continue
		}
		uptime.HourlyStatistics[hourlyUnixTimestamp] = &core.HourlyUptimeStatistics{
			TotalExecutions:             totalExecutions,
			SuccessfulExecutions:        uptime.SuccessfulExecutionsPerHour[hourlyUnixTimestamp],
			TotalExecutionsResponseTime: 0,
		}
	}
	log.Println("[migrateUptimeToHourlyStatistics] Migrated", len(uptime.HourlyStatistics), "entries")
	uptime.SuccessfulExecutionsPerHour = nil
	uptime.TotalExecutionsPerHour = nil
}

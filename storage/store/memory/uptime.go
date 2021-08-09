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
	if result.Success {
		// Recalculate uptime if at least one of the 1h, 24h or 7d uptime are not 100%
		// If they're all 100%, then recalculating the uptime would be useless unless
		// the result added was a failure (!result.Success)
		if uptime.LastSevenDays != 1 || uptime.LastTwentyFourHours != 1 || uptime.LastHour != 1 {
			recalculateUptime(uptime)
		}
	} else {
		// Recalculate uptime if at least one of the 1h, 24h or 7d uptime are not 0%
		// If they're all 0%, then recalculating the uptime would be useless unless
		// the result added was a success (result.Success)
		if uptime.LastSevenDays != 0 || uptime.LastTwentyFourHours != 0 || uptime.LastHour != 0 {
			recalculateUptime(uptime)
		}
	}
}

// recalculateUptime calculates the uptime over the past 7 days, 24 hours and 1 hour.
func recalculateUptime(uptime *core.Uptime) {
	uptimeBrackets := make(map[string]uint64)
	now := time.Now()
	// The oldest uptime bracket starts 7 days ago, so we'll start from there
	timestamp := now.Add(-sevenDays)
	for now.Sub(timestamp) >= 0 {
		hourlyUnixTimestamp := timestamp.Truncate(time.Hour).Unix()
		hourlyStats := uptime.HourlyStatistics[hourlyUnixTimestamp]
		if hourlyStats == nil || hourlyStats.TotalExecutions == 0 {
			timestamp = timestamp.Add(time.Hour)
			continue
		}
		uptimeBrackets["7d_success"] += hourlyStats.SuccessfulExecutions
		uptimeBrackets["7d_total"] += hourlyStats.TotalExecutions
		if now.Sub(timestamp) <= 24*time.Hour {
			uptimeBrackets["24h_success"] += hourlyStats.SuccessfulExecutions
			uptimeBrackets["24h_total"] += hourlyStats.TotalExecutions
		}
		if now.Sub(timestamp) <= time.Hour {
			uptimeBrackets["1h_success"] += hourlyStats.SuccessfulExecutions
			uptimeBrackets["1h_total"] += hourlyStats.TotalExecutions
		}
		timestamp = timestamp.Add(time.Hour)
	}
	if uptimeBrackets["7d_total"] > 0 {
		uptime.LastSevenDays = float64(uptimeBrackets["7d_success"]) / float64(uptimeBrackets["7d_total"])
	}
	if uptimeBrackets["24h_total"] > 0 {
		uptime.LastTwentyFourHours = float64(uptimeBrackets["24h_success"]) / float64(uptimeBrackets["24h_total"])
	}
	if uptimeBrackets["1h_total"] > 0 {
		uptime.LastHour = float64(uptimeBrackets["1h_success"]) / float64(uptimeBrackets["1h_total"])
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

package core

import (
	"log"
	"time"
)

const (
	numberOfHoursInTenDays = 10 * 24
	sevenDays              = 7 * 24 * time.Hour
)

// Uptime is the struct that contains the relevant data for calculating the uptime as well as the uptime itself
// and some other statistics
type Uptime struct {
	// LastSevenDays is the uptime percentage over the past 7 days
	LastSevenDays float64 `json:"7d"`

	// LastTwentyFourHours is the uptime percentage over the past 24 hours
	LastTwentyFourHours float64 `json:"24h"`

	// LastHour is the uptime percentage over the past hour
	LastHour float64 `json:"1h"`

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

// ProcessResult processes the result by extracting the relevant from the result and recalculating the uptime
// if necessary
func (uptime *Uptime) ProcessResult(result *Result) {
	// XXX: Remove this on v3.0.0
	if len(uptime.SuccessfulExecutionsPerHour) != 0 || len(uptime.TotalExecutionsPerHour) != 0 {
		uptime.migrateToHourlyStatistics()
	}
	if uptime.HourlyStatistics == nil {
		uptime.HourlyStatistics = make(map[int64]*HourlyUptimeStatistics)
	}
	unixTimestampFlooredAtHour := result.Timestamp.Unix() - (result.Timestamp.Unix() % 3600)
	hourlyStats, _ := uptime.HourlyStatistics[unixTimestampFlooredAtHour]
	if hourlyStats == nil {
		hourlyStats = &HourlyUptimeStatistics{}
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
	// This is to prevent re-iterating on every `ProcessResult` as soon as the uptime has been logged for 7 days.
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
			uptime.recalculate()
		}
	} else {
		// Recalculate uptime if at least one of the 1h, 24h or 7d uptime are not 0%
		// If they're all 0%, then recalculating the uptime would be useless unless
		// the result added was a success (result.Success)
		if uptime.LastSevenDays != 0 || uptime.LastTwentyFourHours != 0 || uptime.LastHour != 0 {
			uptime.recalculate()
		}
	}
	// cute print
	//b, _ := json.MarshalIndent(uptime.TotalExecutionsPerHour, "", "  ")
	//fmt.Println("TotalExecutionsPerHour:", string(b))
	//b, _ = json.MarshalIndent(uptime.SuccessfulExecutionsPerHour, "", "  ")
	//fmt.Println("SuccessfulExecutionsPerHour:", string(b))
	//b, _ = json.MarshalIndent(uptime.TotalRequestResponseTimePerHour, "", "  ")
	//fmt.Println("TotalRequestResponseTimePerHour:", string(b))
	//for unixTimestamp, executions := range uptime.TotalExecutionsPerHour {
	//	fmt.Printf("average for %d was %d\n", unixTimestamp, uptime.TotalRequestResponseTimePerHour[unixTimestamp]/executions)
	//}
}

func (uptime *Uptime) recalculate() {
	uptimeBrackets := make(map[string]uint64)
	now := time.Now()
	// The oldest uptime bracket starts 7 days ago, so we'll start from there
	timestamp := now.Add(-sevenDays)
	for now.Sub(timestamp) >= 0 {
		hourlyUnixTimestamp := timestamp.Unix() - (timestamp.Unix() % 3600)
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

func (uptime *Uptime) migrateToHourlyStatistics() {
	log.Println("[migrateToHourlyStatistics] Got", len(uptime.SuccessfulExecutionsPerHour), "entries for successful executions and", len(uptime.TotalExecutionsPerHour), "entries for total executions")
	uptime.HourlyStatistics = make(map[int64]*HourlyUptimeStatistics)
	for hourlyUnixTimestamp, totalExecutions := range uptime.TotalExecutionsPerHour {
		if totalExecutions == 0 {
			log.Println("[migrateToHourlyStatistics] Skipping entry at", hourlyUnixTimestamp, "because total number of executions is 0")
			continue
		}
		uptime.HourlyStatistics[hourlyUnixTimestamp] = &HourlyUptimeStatistics{
			TotalExecutions:             totalExecutions,
			SuccessfulExecutions:        uptime.SuccessfulExecutionsPerHour[hourlyUnixTimestamp],
			TotalExecutionsResponseTime: 0,
		}
	}
	log.Println("[migrateToHourlyStatistics] Migrated", len(uptime.HourlyStatistics), "entries")
	uptime.SuccessfulExecutionsPerHour = nil
	uptime.TotalExecutionsPerHour = nil
}

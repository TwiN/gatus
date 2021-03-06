package core

import (
	"time"
)

const (
	numberOfHoursInTenDays = 10 * 24
	sevenDays              = 7 * 24 * time.Hour
)

// Uptime is the struct that contains the relevant data for calculating the uptime as well as the uptime itself
type Uptime struct {
	// LastSevenDays is the uptime percentage over the past 7 days
	LastSevenDays float64 `json:"7d"`

	// LastTwentyFourHours is the uptime percentage over the past 24 hours
	LastTwentyFourHours float64 `json:"24h"`

	// LastHour is the uptime percentage over the past hour
	LastHour float64 `json:"1h"`

	// SuccessfulExecutionsPerHour is a map containing the number of successes (value)
	// for every hourly unix timestamps (key)
	SuccessfulExecutionsPerHour map[int64]uint64 `json:"-"`

	// TotalExecutionsPerHour is a map containing the total number of checks (value)
	// for every hourly unix timestamps (key)
	TotalExecutionsPerHour map[int64]uint64 `json:"-"`
}

// NewUptime creates a new Uptime
func NewUptime() *Uptime {
	return &Uptime{
		SuccessfulExecutionsPerHour: make(map[int64]uint64),
		TotalExecutionsPerHour:      make(map[int64]uint64),
	}
}

// ProcessResult processes the result by extracting the relevant from the result and recalculating the uptime
// if necessary
func (uptime *Uptime) ProcessResult(result *Result) {
	if uptime.SuccessfulExecutionsPerHour == nil || uptime.TotalExecutionsPerHour == nil {
		uptime.SuccessfulExecutionsPerHour = make(map[int64]uint64)
		uptime.TotalExecutionsPerHour = make(map[int64]uint64)
	}
	unixTimestampFlooredAtHour := result.Timestamp.Unix() - (result.Timestamp.Unix() % 3600)
	if result.Success {
		uptime.SuccessfulExecutionsPerHour[unixTimestampFlooredAtHour]++
	}
	uptime.TotalExecutionsPerHour[unixTimestampFlooredAtHour]++
	// Clean up only when we're starting to have too many useless keys
	// Note that this is only triggered when there are more entries than there should be after
	// 10 days, despite the fact that we are deleting everything that's older than 7 days.
	// This is to prevent re-iterating on every `ProcessResult` as soon as the uptime has been logged for 7 days.
	if len(uptime.TotalExecutionsPerHour) > numberOfHoursInTenDays {
		sevenDaysAgo := time.Now().Add(-(sevenDays + time.Hour)).Unix()
		for hourlyUnixTimestamp := range uptime.TotalExecutionsPerHour {
			if sevenDaysAgo > hourlyUnixTimestamp {
				delete(uptime.TotalExecutionsPerHour, hourlyUnixTimestamp)
				delete(uptime.SuccessfulExecutionsPerHour, hourlyUnixTimestamp)
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
}

func (uptime *Uptime) recalculate() {
	uptimeBrackets := make(map[string]uint64)
	now := time.Now()
	// The oldest uptime bracket starts 7 days ago, so we'll start from there
	timestamp := now.Add(-sevenDays)
	for now.Sub(timestamp) >= 0 {
		hourlyUnixTimestamp := timestamp.Unix() - (timestamp.Unix() % 3600)
		successCountForTimestamp := uptime.SuccessfulExecutionsPerHour[hourlyUnixTimestamp]
		totalCountForTimestamp := uptime.TotalExecutionsPerHour[hourlyUnixTimestamp]
		uptimeBrackets["7d_success"] += successCountForTimestamp
		uptimeBrackets["7d_total"] += totalCountForTimestamp
		if now.Sub(timestamp) <= 24*time.Hour {
			uptimeBrackets["24h_success"] += successCountForTimestamp
			uptimeBrackets["24h_total"] += totalCountForTimestamp
		}
		if now.Sub(timestamp) <= time.Hour {
			uptimeBrackets["1h_success"] += successCountForTimestamp
			uptimeBrackets["1h_total"] += totalCountForTimestamp
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

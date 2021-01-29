package core

import (
	"log"
	"time"
)

const (
	// RFC3339WithoutMinutesAndSeconds is the format defined by RFC3339 (see time.RFC3339) but with the minutes
	// and seconds hardcoded to 0.
	RFC3339WithoutMinutesAndSeconds = "2006-01-02T15:00:00Z07:00"

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

	successCountPerHour map[string]uint64
	totalCountPerHour   map[string]uint64
}

// NewUptime creates a new Uptime
func NewUptime() *Uptime {
	return &Uptime{
		successCountPerHour: make(map[string]uint64),
		totalCountPerHour:   make(map[string]uint64),
	}
}

// ProcessResult processes the result by extracting the relevant from the result and recalculating the uptime
// if necessary
func (uptime *Uptime) ProcessResult(result *Result) {
	if uptime.successCountPerHour == nil {
		uptime.successCountPerHour = make(map[string]uint64)
	}
	if uptime.totalCountPerHour == nil {
		uptime.totalCountPerHour = make(map[string]uint64)
	}

	timestampDateWithHour := result.Timestamp.Format(RFC3339WithoutMinutesAndSeconds)
	if result.Success {
		uptime.successCountPerHour[timestampDateWithHour]++
	}
	uptime.totalCountPerHour[timestampDateWithHour]++
	// Clean up only when we're starting to have too many useless keys
	// Note that this is only triggered when there are more entries than there should be after
	// 10 days, despite the fact that we are deleting everything that's older than 7 days.
	// This is to prevent re-iterating on every `ProcessResult` as soon as the uptime has been logged for 7 days.
	if len(uptime.totalCountPerHour) > numberOfHoursInTenDays {
		sevenDaysAgo := time.Now().Add(-(sevenDays + time.Hour))
		for k := range uptime.totalCountPerHour {
			dateWithHour, err := time.Parse(time.RFC3339, k)
			if err != nil {
				// This shouldn't happen, but we'll log it in case it does happen
				log.Println("[uptime][ProcessResult] Failed to parse programmatically generated timestamp:", err.Error())
				continue
			}
			if sevenDaysAgo.Unix() > dateWithHour.Unix() {
				delete(uptime.totalCountPerHour, k)
				delete(uptime.successCountPerHour, k)
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
		timestampDateWithHour := timestamp.Format(RFC3339WithoutMinutesAndSeconds)
		successCountForTimestamp := uptime.successCountPerHour[timestampDateWithHour]
		totalCountForTimestamp := uptime.totalCountPerHour[timestampDateWithHour]
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

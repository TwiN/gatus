package endpoint

import (
	"math"
	"time"

	"github.com/TwiN/gatus/v5/config/key"
)

// Status contains the evaluation Results of an Endpoint
// This is essentially a DTO
type Status struct {
	// Name of the endpoint
	Name string `json:"name,omitempty"`

	// Group the endpoint is a part of. Used for grouping multiple endpoints together on the front end.
	Group string `json:"group,omitempty"`

	// Key of the Endpoint
	Key string `json:"key"`

	// Results is the list of endpoint evaluation results
	Results []*Result `json:"results"`

	// Events is a list of events
	Events []*Event `json:"events,omitempty"`

	// Uptime information on the endpoint's uptime
	//
	// Used by the memory store.
	//
	// To retrieve the uptime between two time, use store.GetUptimeByKey.
	Uptime *Uptime `json:"-"`

	// Period is the configured time window for the "Recent Checks" display.
	// When set, indicates that the endpoint has a custom period configured.
	// Represented as a duration string (e.g., "30d", "7d", "24h").
	Period string `json:"period,omitempty"`

	// UptimeStats contains uptime percentages for standard time windows.
	// Populated by the API layer, not by the store.
	UptimeStats *UptimeStats `json:"uptime,omitempty"`
}

// UptimeStats contains uptime percentages for standard time windows.
// Values are floats between 0.0 and 1.0.
type UptimeStats struct {
	Hour  float64 `json:"hour"`
	Day   float64 `json:"day"`
	Week  float64 `json:"week"`
	Month float64 `json:"month"`
}

// NewStatus creates a new Status
func NewStatus(group, name string) *Status {
	return &Status{
		Name:    name,
		Group:   group,
		Key:     key.ConvertGroupAndNameToKey(group, name),
		Results: make([]*Result, 0),
		Events:  make([]*Event, 0),
		Uptime:  NewUptime(),
	}
}

// SetPeriod sets the Period field from a time.Duration
func (s *Status) SetPeriod(d time.Duration) {
	if d == 0 {
		return
	}
	hours := d.Hours()
	if hours >= 24 && int(hours)%24 == 0 {
		s.Period = formatDurationString(int(hours)/24, "d")
	} else {
		s.Period = formatDurationString(int(hours), "h")
	}
}

// NormalizeUptime clamps an uptime value to [0, 1] and rounds to 4 decimal places
func NormalizeUptime(uptime float64) float64 {
	uptime = math.Max(0, math.Min(1, uptime))
	return math.Round(uptime*10000) / 10000
}

func formatDurationString(value int, unit string) string {
	// Simple integer to string conversion without fmt dependency
	if value == 0 {
		return "0" + unit
	}
	digits := make([]byte, 0, 4)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	// Reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits) + unit
}

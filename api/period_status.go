package api

import (
	"errors"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/gofiber/fiber/v2"
)

// PeriodStatusResponse is the response from the period-statuses endpoint
type PeriodStatusResponse struct {
	Duration string            `json:"duration"`
	Parts    int               `json:"parts"`
	Uptime   float64           `json:"uptime"`
	Results  []PeriodResult    `json:"results"`
}

// PeriodResult represents a single time slice within a period.
// It mirrors the endpoint.Result structure returned by /api/v1/endpoints/statuses,
// with an additional Missing field to indicate slices with no data.
type PeriodResult struct {
	HTTPStatus       int                       `json:"status,omitempty"`
	Hostname         string                    `json:"hostname,omitempty"`
	Duration         time.Duration             `json:"duration"`
	Errors           []string                  `json:"errors,omitempty"`
	ConditionResults []*endpoint.ConditionResult `json:"conditionResults,omitempty"`
	Success          bool                      `json:"success"`
	Timestamp        time.Time                 `json:"timestamp"`
	Missing          bool                      `json:"missing,omitempty"`
}

// PeriodStatuses returns uptime and response time data for an endpoint, sampled evenly across a time period.
//
// Each result mirrors the structure of /api/v1/endpoints/statuses results.
// Time slices with no data are marked with missing: true.
//
// Supported duration formats: 30d, 7d, 24h, 1h, or any custom duration like 14d, 60d, 90d, 2h, etc.
// Parts defines the number of evenly spaced samples to return (1-100).
func PeriodStatuses(c *fiber.Ctx) error {
	durationStr := c.Params("duration")
	parsedDuration, err := ParseCustomDuration(durationStr)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	parts, err := strconv.Atoi(c.Params("parts"))
	if err != nil || parts < 1 {
		return c.Status(400).SendString("parts must be a positive integer")
	}
	if parts > 100 {
		return c.Status(400).SendString("parts must be at most 100")
	}
	key, err := url.QueryUnescape(c.Params("key"))
	if err != nil {
		return c.Status(400).SendString("invalid key encoding")
	}
	now := time.Now()
	from := now.Add(-parsedDuration)

	// Calculate overall uptime for the entire period
	overallUptime, err := store.Get().GetUptimeByKey(key, from, now)
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return c.Status(404).SendString(err.Error())
		}
		return c.Status(500).SendString(err.Error())
	}
	overallUptime = math.Max(0, math.Min(1, overallUptime))
	overallUptime = math.Round(overallUptime*10000) / 10000

	// Build per-slice results
	sliceDuration := time.Duration(int64(parsedDuration) / int64(parts))
	results := make([]PeriodResult, 0, parts)
	for i := 0; i < parts; i++ {
		sliceFrom := from.Add(time.Duration(i) * sliceDuration)
		sliceTo := sliceFrom.Add(sliceDuration)
		if i == parts-1 {
			sliceTo = now
		}

		uptime, err := store.Get().GetUptimeByKey(key, sliceFrom, sliceTo)
		if err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return c.Status(404).SendString(err.Error())
			}
			// For other errors, mark as missing
			results = append(results, PeriodResult{
				Timestamp: sliceTo,
				Missing:   true,
			})
			continue
		}

		responseTime, err := store.Get().GetAverageResponseTimeByKey(key, sliceFrom, sliceTo)
		if err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return c.Status(404).SendString(err.Error())
			}
			results = append(results, PeriodResult{
				Timestamp: sliceTo,
				Missing:   true,
			})
			continue
		}

		// If both uptime and responseTime are zero, this slice likely has no data
		if uptime == 0 && responseTime == 0 {
			results = append(results, PeriodResult{
				Timestamp: sliceTo,
				Missing:   true,
			})
			continue
		}

		uptime = math.Max(0, math.Min(1, uptime))
		uptime = math.Round(uptime*10000) / 10000
		success := uptime >= 0.99

		results = append(results, PeriodResult{
			Success:   success,
			Timestamp: sliceTo,
			Duration:  time.Duration(responseTime) * time.Millisecond,
		})
	}

	c.Set("Content-Type", "application/json")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Expires", "0")
	return c.Status(200).JSON(PeriodStatusResponse{
		Duration: durationStr,
		Parts:    parts,
		Uptime:   overallUptime,
		Results:  results,
	})
}

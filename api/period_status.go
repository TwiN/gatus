package api

import (
	"errors"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/gofiber/fiber/v2"
)

// PeriodStatusSlice represents a single time slice within a period
type PeriodStatusSlice struct {
	Timestamp    int64   `json:"timestamp"`
	Uptime       float64 `json:"uptime"`
	ResponseTime int     `json:"response_time"`
}

// PeriodStatusResponse is the response from the period-statuses endpoint
type PeriodStatusResponse struct {
	Duration string              `json:"duration"`
	Parts    int                 `json:"parts"`
	Slices   []PeriodStatusSlice `json:"slices"`
}

// PeriodStatuses returns uptime and response time data for an endpoint, sampled evenly across a time period.
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
	sliceDuration := time.Duration(int64(parsedDuration) / int64(parts))
	slices := make([]PeriodStatusSlice, 0, parts)
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
			return c.Status(500).SendString(err.Error())
		}
		responseTime, err := store.Get().GetAverageResponseTimeByKey(key, sliceFrom, sliceTo)
		if err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return c.Status(404).SendString(err.Error())
			}
			return c.Status(500).SendString(err.Error())
		}
		// Normalize uptime: clamp to [0, 1] and round to 4 decimal places
		uptime = math.Max(0, math.Min(1, uptime))
		uptime = math.Round(uptime*10000) / 10000
		slices = append(slices, PeriodStatusSlice{
			Timestamp:    sliceTo.UnixMilli(),
			Uptime:       uptime,
			ResponseTime: responseTime,
		})
	}
	c.Set("Content-Type", "application/json")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Expires", "0")
	return c.Status(200).JSON(PeriodStatusResponse{
		Duration: durationStr,
		Parts:    parts,
		Slices:   slices,
	})
}

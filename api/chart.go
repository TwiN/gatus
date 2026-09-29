package api

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

const timeFormat = "3:04PM"

var (
	gridStyle = chart.Style{
		StrokeColor: drawing.Color{R: 119, G: 119, B: 119, A: 40},
		StrokeWidth: 1.0,
	}
	axisStyle = chart.Style{
		FontColor: drawing.Color{R: 119, G: 119, B: 119, A: 255},
	}
	transparentStyle = chart.Style{
		FillColor: drawing.Color{R: 255, G: 255, B: 255, A: 0},
	}
)

func ResponseTimeChart(c *fiber.Ctx) error {
	duration := c.Params("duration")
	chartTimestampFormatter := chart.TimeValueFormatterWithFormat(timeFormat)
	var from time.Time
	switch duration {
	case "30d":
		from = time.Now().Truncate(time.Hour).Add(-30 * 24 * time.Hour)
		chartTimestampFormatter = chart.TimeDateValueFormatter
	case "7d":
		from = time.Now().Truncate(time.Hour).Add(-7 * 24 * time.Hour)
	case "24h":
		from = time.Now().Truncate(time.Hour).Add(-24 * time.Hour)
	default:
		return c.Status(400).SendString("Durations supported: 30d, 7d, 24h")
	}
	key, err := url.QueryUnescape(c.Params("key"))
	if err != nil {
		return c.Status(400).SendString("invalid key encoding")
	}
	hourlyAverageResponseTime, err := store.Get().GetHourlyAverageResponseTimeByKey(key, from, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return c.Status(404).SendString(err.Error())
		} else if errors.Is(err, common.ErrInvalidTimeRange) {
			return c.Status(400).SendString(err.Error())
		}
		return c.Status(500).SendString(err.Error())
	}
	if len(hourlyAverageResponseTime) == 0 {
		return c.Status(204).SendString("")
	}
	series := chart.TimeSeries{
		Name: "Average response time per hour",
		Style: chart.Style{
			StrokeWidth: 1.5,
			DotWidth:    2.0,
		},
	}
	keys := make([]int, 0, len(hourlyAverageResponseTime))
	earliestTimestamp := int64(0)
	for hourlyTimestamp := range hourlyAverageResponseTime {
		keys = append(keys, int(hourlyTimestamp))
		if earliestTimestamp == 0 || hourlyTimestamp < earliestTimestamp {
			earliestTimestamp = hourlyTimestamp
		}
	}
	for earliestTimestamp > from.Unix() {
		earliestTimestamp -= int64(time.Hour.Seconds())
		keys = append(keys, int(earliestTimestamp))
	}
	sort.Ints(keys)
	var maxAverageResponseTime float64
	for _, key := range keys {
		averageResponseTime := float64(hourlyAverageResponseTime[int64(key)])
		if maxAverageResponseTime < averageResponseTime {
			maxAverageResponseTime = averageResponseTime
		}
		series.XValues = append(series.XValues, time.Unix(int64(key), 0))
		series.YValues = append(series.YValues, averageResponseTime)
	}
	graph := chart.Chart{
		Canvas:     transparentStyle,
		Background: transparentStyle,
		Width:      1280,
		Height:     300,
		XAxis: chart.XAxis{
			ValueFormatter: chartTimestampFormatter,
			GridMajorStyle: gridStyle,
			GridMinorStyle: gridStyle,
			Style:          axisStyle,
			NameStyle:      axisStyle,
		},
		YAxis: chart.YAxis{
			Name:           "Average response time",
			GridMajorStyle: gridStyle,
			GridMinorStyle: gridStyle,
			Style:          axisStyle,
			NameStyle:      axisStyle,
			Range: &chart.ContinuousRange{
				Min: 0,
				Max: math.Ceil(maxAverageResponseTime * 1.25),
			},
		},
		Series: []chart.Series{series},
	}
	c.Set("Content-Type", "image/svg+xml")
	c.Set("Cache-Control", "no-cache, no-store")
	c.Set("Expires", "0")
	c.Status(http.StatusOK)
	if err := graph.Render(chart.SVG, c); err != nil {
		logr.Errorf("[api.ResponseTimeChart] Failed to render response time chart: %s", err.Error())
		return c.Status(500).SendString(err.Error())
	}
	return nil
}

func ResponseTimeHistory(c *fiber.Ctx) error {
	duration := c.Params("duration")
	var from time.Time
	switch duration {
	case "30d":
		from = time.Now().Truncate(time.Hour).Add(-30 * 24 * time.Hour)
	case "7d":
		from = time.Now().Truncate(time.Hour).Add(-7 * 24 * time.Hour)
	case "24h":
		from = time.Now().Truncate(time.Hour).Add(-24 * time.Hour)
	default:
		return c.Status(400).SendString("Durations supported: 30d, 7d, 24h")
	}
	endpointKey, err := url.QueryUnescape(c.Params("key"))
	if err != nil {
		return c.Status(400).SendString("invalid key encoding")
	}
	hourlyAverageResponseTime, err := store.Get().GetHourlyAverageResponseTimeByKey(endpointKey, from, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return c.Status(404).SendString(err.Error())
		}
		if errors.Is(err, common.ErrInvalidTimeRange) {
			return c.Status(400).SendString(err.Error())
		}
		return c.Status(500).SendString(err.Error())
	}
	if len(hourlyAverageResponseTime) == 0 {
		return c.Status(200).JSON(map[string]interface{}{
			"timestamps": []int64{},
			"values":     []int{},
		})
	}
	hourlyTimestamps := make([]int, 0, len(hourlyAverageResponseTime))
	earliestTimestamp := int64(0)
	for hourlyTimestamp := range hourlyAverageResponseTime {
		hourlyTimestamps = append(hourlyTimestamps, int(hourlyTimestamp))
		if earliestTimestamp == 0 || hourlyTimestamp < earliestTimestamp {
			earliestTimestamp = hourlyTimestamp
		}
	}
	for earliestTimestamp > from.Unix() {
		earliestTimestamp -= int64(time.Hour.Seconds())
		hourlyTimestamps = append(hourlyTimestamps, int(earliestTimestamp))
	}
	sort.Ints(hourlyTimestamps)
	timestamps := make([]int64, 0, len(hourlyTimestamps))
	values := make([]int, 0, len(hourlyTimestamps))
	for _, hourlyTimestamp := range hourlyTimestamps {
		timestamp := int64(hourlyTimestamp)
		averageResponseTime := hourlyAverageResponseTime[timestamp]
		timestamps = append(timestamps, timestamp*1000)
		values = append(values, averageResponseTime)
	}
	return c.Status(http.StatusOK).JSON(map[string]interface{}{
		"timestamps": timestamps,
		"values":     values,
	})
}

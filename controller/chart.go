package controller

import (
	"log"
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/TwinProduction/gatus/storage"
	"github.com/TwinProduction/gatus/storage/store/common"
	"github.com/gorilla/mux"
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

func responseTimeChartHandler(writer http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	duration := vars["duration"]
	var from time.Time
	switch duration {
	case "7d":
		from = time.Now().Truncate(time.Hour).Add(-24 * 7 * time.Hour)
	case "24h":
		from = time.Now().Truncate(time.Hour).Add(-24 * time.Hour)
	default:
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("Durations supported: 7d, 24h"))
		return
	}
	hourlyAverageResponseTime, err := storage.Get().GetHourlyAverageResponseTimeByKey(vars["key"], from, time.Now())
	if err != nil {
		if err == common.ErrServiceNotFound {
			writer.WriteHeader(http.StatusNotFound)
		} else if err == common.ErrInvalidTimeRange {
			writer.WriteHeader(http.StatusBadRequest)
		} else {
			writer.WriteHeader(http.StatusInternalServerError)
		}
		_, _ = writer.Write([]byte(err.Error()))
		return
	}
	if len(hourlyAverageResponseTime) == 0 {
		writer.WriteHeader(http.StatusNoContent)
		_, _ = writer.Write(nil)
		return
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
			ValueFormatter: chart.TimeValueFormatterWithFormat(timeFormat),
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
	writer.Header().Set("Content-Type", "image/svg+xml")
	if err := graph.Render(chart.SVG, writer); err != nil {
		log.Println("[controller][responseTimeChartHandler] Failed to render response time chart:", err.Error())
		return
	}
}

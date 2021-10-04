package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/v3/storage"
	"github.com/TwinProduction/gatus/v3/storage/store/common"
	"github.com/gorilla/mux"
)

const (
	badgeColorHexAwesome  = "#40cc11"
	badgeColorHexGreat    = "#94cc11"
	badgeColorHexGood     = "#ccd311"
	badgeColorHexPassable = "#ccb311"
	badgeColorHexBad      = "#cc8111"
	badgeColorHexVeryBad  = "#c7130a"
)

// UptimeBadge handles the automatic generation of badge based on the group name and service name passed.
//
// Valid values for {duration}: 7d, 24h, 1h
func UptimeBadge(writer http.ResponseWriter, request *http.Request) {
	variables := mux.Vars(request)
	duration := variables["duration"]
	var from time.Time
	switch duration {
	case "7d":
		from = time.Now().Add(-7 * 24 * time.Hour)
	case "24h":
		from = time.Now().Add(-24 * time.Hour)
	case "1h":
		from = time.Now().Add(-2 * time.Hour) // Because uptime metrics are stored by hour, we have to cheat a little
	default:
		http.Error(writer, "Durations supported: 7d, 24h, 1h", http.StatusBadRequest)
		return
	}
	key := variables["key"]
	uptime, err := storage.Get().GetUptimeByKey(key, from, time.Now())
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
	formattedDate := time.Now().Format(http.TimeFormat)
	writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	writer.Header().Set("Date", formattedDate)
	writer.Header().Set("Expires", formattedDate)
	writer.Header().Set("Content-Type", "image/svg+xml")
	_, _ = writer.Write(generateUptimeBadgeSVG(duration, uptime))
}

// ResponseTimeBadge handles the automatic generation of badge based on the group name and service name passed.
//
// Valid values for {duration}: 7d, 24h, 1h
func ResponseTimeBadge(writer http.ResponseWriter, request *http.Request) {
	variables := mux.Vars(request)
	duration := variables["duration"]
	var from time.Time
	switch duration {
	case "7d":
		from = time.Now().Add(-7 * 24 * time.Hour)
	case "24h":
		from = time.Now().Add(-24 * time.Hour)
	case "1h":
		from = time.Now().Add(-2 * time.Hour) // Because response time metrics are stored by hour, we have to cheat a little
	default:
		http.Error(writer, "Durations supported: 7d, 24h, 1h", http.StatusBadRequest)
		return
	}
	key := variables["key"]
	averageResponseTime, err := storage.Get().GetAverageResponseTimeByKey(key, from, time.Now())
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
	formattedDate := time.Now().Format(http.TimeFormat)
	writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	writer.Header().Set("Date", formattedDate)
	writer.Header().Set("Expires", formattedDate)
	writer.Header().Set("Content-Type", "image/svg+xml")
	_, _ = writer.Write(generateResponseTimeBadgeSVG(duration, averageResponseTime))
}

func generateUptimeBadgeSVG(duration string, uptime float64) []byte {
	var labelWidth, valueWidth, valueWidthAdjustment int
	switch duration {
	case "7d":
		labelWidth = 65
	case "24h":
		labelWidth = 70
	case "1h":
		labelWidth = 65
	default:
	}
	color := getBadgeColorFromUptime(uptime)
	sanitizedValue := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", uptime*100), "0"), ".") + "%"
	if strings.Contains(sanitizedValue, ".") {
		valueWidthAdjustment = -10
	}
	valueWidth = (len(sanitizedValue) * 11) + valueWidthAdjustment
	width := labelWidth + valueWidth
	labelX := labelWidth / 2
	valueX := labelWidth + (valueWidth / 2)
	svg := []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="%d" height="20" rx="3" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h%dv20H0z"/>
    <path fill="%s" d="M%d 0h%dv20H%dz"/>
    <path fill="url(#b)" d="M0 0h%dv20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      uptime %s
    </text>
    <text x="%d" y="14">
      uptime %s
    </text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      %s
    </text>
    <text x="%d" y="14">
      %s
    </text>
  </g>
</svg>`, width, width, labelWidth, color, labelWidth, valueWidth, labelWidth, width, labelX, duration, labelX, duration, valueX, sanitizedValue, valueX, sanitizedValue))
	return svg
}

func getBadgeColorFromUptime(uptime float64) string {
	if uptime >= 0.975 {
		return badgeColorHexAwesome
	} else if uptime >= 0.95 {
		return badgeColorHexGreat
	} else if uptime >= 0.9 {
		return badgeColorHexGood
	} else if uptime >= 0.8 {
		return badgeColorHexPassable
	} else if uptime >= 0.65 {
		return badgeColorHexBad
	}
	return badgeColorHexVeryBad
}

func generateResponseTimeBadgeSVG(duration string, averageResponseTime int) []byte {
	var labelWidth, valueWidth int
	switch duration {
	case "7d":
		labelWidth = 105
	case "24h":
		labelWidth = 110
	case "1h":
		labelWidth = 105
	default:
	}
	color := getBadgeColorFromResponseTime(averageResponseTime)
	sanitizedValue := strconv.Itoa(averageResponseTime) + "ms"
	valueWidth = len(sanitizedValue) * 11
	width := labelWidth + valueWidth
	labelX := labelWidth / 2
	valueX := labelWidth + (valueWidth / 2)
	svg := []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="20">
  <linearGradient id="b" x2="0" y2="100%%">
    <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
    <stop offset="1" stop-opacity=".1"/>
  </linearGradient>
  <mask id="a">
    <rect width="%d" height="20" rx="3" fill="#fff"/>
  </mask>
  <g mask="url(#a)">
    <path fill="#555" d="M0 0h%dv20H0z"/>
    <path fill="%s" d="M%d 0h%dv20H%dz"/>
    <path fill="url(#b)" d="M0 0h%dv20H0z"/>
  </g>
  <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      response time %s
    </text>
    <text x="%d" y="14">
      response time %s
    </text>
    <text x="%d" y="15" fill="#010101" fill-opacity=".3">
      %s
    </text>
    <text x="%d" y="14">
      %s
    </text>
  </g>
</svg>`, width, width, labelWidth, color, labelWidth, valueWidth, labelWidth, width, labelX, duration, labelX, duration, valueX, sanitizedValue, valueX, sanitizedValue))
	return svg
}

func getBadgeColorFromResponseTime(responseTime int) string {
	if responseTime <= 50 {
		return badgeColorHexAwesome
	} else if responseTime <= 200 {
		return badgeColorHexGreat
	} else if responseTime <= 300 {
		return badgeColorHexGood
	} else if responseTime <= 500 {
		return badgeColorHexPassable
	} else if responseTime <= 750 {
		return badgeColorHexBad
	}
	return badgeColorHexVeryBad
}

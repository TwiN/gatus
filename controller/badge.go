package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/watchdog"
	"github.com/gorilla/mux"
)

// badgeHandler handles the automatic generation of badge based on the group name and service name passed.
//
// Valid values for {duration}: 7d, 24h, 1h
// Pattern for {identifier}: <KEY>.svg
func badgeHandler(writer http.ResponseWriter, request *http.Request) {
	variables := mux.Vars(request)
	duration := variables["duration"]
	if duration != "7d" && duration != "24h" && duration != "1h" {
		writer.WriteHeader(http.StatusBadRequest)
		_, _ = writer.Write([]byte("Durations supported: 7d, 24h, 1h"))
		return
	}
	identifier := variables["identifier"]
	key := strings.TrimSuffix(identifier, ".svg")
	uptime := watchdog.GetUptimeByKey(key)
	if uptime == nil {
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte("Requested service not found"))
		return
	}
	formattedDate := time.Now().Format(http.TimeFormat)
	writer.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	writer.Header().Set("Date", formattedDate)
	writer.Header().Set("Expires", formattedDate)
	writer.Header().Set("Content-Type", "image/svg+xml")
	_, _ = writer.Write(generateSVG(duration, uptime))
}

func generateSVG(duration string, uptime *core.Uptime) []byte {
	var labelWidth, valueWidth, valueWidthAdjustment int
	var color string
	var value float64
	switch duration {
	case "7d":
		labelWidth = 65
		value = uptime.LastSevenDays
	case "24h":
		labelWidth = 70
		value = uptime.LastTwentyFourHours
	case "1h":
		labelWidth = 65
		value = uptime.LastHour
	default:
	}
	if value >= 0.8 {
		color = "#40cc11"
	} else {
		color = "#c7130a"
	}
	sanitizedValue := strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", value*100), "0"), ".") + "%"
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

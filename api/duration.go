package api

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var durationRegex = regexp.MustCompile(`^(\d+)(h|d)$`)

// ParseCustomDuration parses a duration string like "30d", "7d", "24h", "1h" into a time.Duration.
// Supports "h" (hours) and "d" (days) suffixes.
// Returns an error if the format is invalid or the value is out of range (1h to 90d).
func ParseCustomDuration(s string) (time.Duration, error) {
	matches := durationRegex.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid duration format: %s (expected format: <number>h or <number>d, e.g., 24h, 7d, 30d)", s)
	}
	value, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration value: %s", matches[1])
	}
	if value <= 0 {
		return 0, fmt.Errorf("duration must be positive: %s", s)
	}
	switch matches[2] {
	case "h":
		d := time.Duration(value) * time.Hour
		if d > 90*24*time.Hour {
			return 0, fmt.Errorf("duration too large: maximum is 90d, got %s", s)
		}
		return d, nil
	case "d":
		d := time.Duration(value) * 24 * time.Hour
		if d > 90*24*time.Hour {
			return 0, fmt.Errorf("duration too large: maximum is 90d, got %s", s)
		}
		return d, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %s (use 'h' for hours or 'd' for days)", matches[2])
	}
}

// CalculateLabelWidth calculates the appropriate SVG label width for a given duration string.
// Uses a heuristic based on the length of the duration string.
func CalculateLabelWidth(duration string, baseLabel string) int {
	// "uptime " = 7 chars, "response time " = 14 chars
	fullLabel := baseLabel + " " + duration
	// Each character is approximately 6.5px wide at font-size 11
	charWidth := 7
	minWidth := 65
	width := len(fullLabel)*charWidth + 10 // 10px padding
	if width < minWidth {
		return minWidth
	}
	return width
}

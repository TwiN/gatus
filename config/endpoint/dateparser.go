package endpoint

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

type DateFormatSpec struct {
	provider func(value string, matches []int) (layout string, start, end int)
	re       *regexp.Regexp
}

// ParseDate parses the given value as a date using a list of formats, and returns the timestamp or an error if parsing fails.
func ParseDate(value string) (ts time.Time, err error) {
	if len(value) > 128 {
		value = value[:128] // limit to first 128 characters to avoid performance issues with regexes
	}
	for _, format := range __formats {
		if matches := format.re.FindStringSubmatchIndex(value); matches != nil {
			layout, start, end := format.provider(value, matches)
			ts, err = time.ParseInLocation(layout, value[start:end], time.Local)
			if ts.Year() == 0 {
				now := time.Now()
				ts = ts.AddDate(now.Year(), int(now.Month()-1), now.Day()-1)
			}
			return
		}
	}
	return time.Time{}, fmt.Errorf(`failed to parse "%s": unknown format`, value)
}

var __formats = make([]DateFormatSpec, 0)

// RegisterDateFormat allows registering a custom date format with a regex and a provider function that extracts the layout and timestamp substring from the input value.
func RegisterDateFormat(regex string, provider func(value string, matches []int) (layout string, start, end int)) {
	__formats = append(__formats, DateFormatSpec{
		provider: provider,
		re:       regexp.MustCompile(regex),
	})
}

func init() {
	simple := func(format string) func(value string, matches []int) (layout string, start, end int) {
		return func(value string, matches []int) (string, int, int) {
			if len(matches) >= 4 && matches[2] > -1 && matches[3] > -1 {
				return format, matches[2], matches[3] // first submatch group
			}
			return format, matches[0], matches[1]
		}
	}

	val := func(value string, match []int) string {
		if match[0] == -1 || match[1] == -1 {
			return ""
		}
		return value[match[0]:match[1]]
	}

	// RFC3339/Nano / ISO 8601 formats - fast path for common formats (one regex for all)
	RegisterDateFormat(
		`\d{4}-\d{2}-\d{2}([Tt ])\d{2}:\d{2}:\d{2}(\.\d+)?([Zz]|[\+\-]\d{2}:*\d{2})?`,
		func(value string, matches []int) (layout string, start, end int) {
			start, end = matches[0], matches[1]
			s1 := val(value, matches[2:4])
			s3 := val(value, matches[6:8])
			if s1 == "T" || s1 == "t" {
				if s3 == "Z" || s3 == "z" || strings.Contains(s3, ":") {
					layout = time.RFC3339
				} else if s3 != "" {
					layout = "2006-01-02T15:04:05Z0700" // ISO 8601 without colon in timezone
				} else {
					layout = "2006-01-02T15:04:05" // ISO 8601 without timezone
				}
			} else {
				if s3 == "" {
					layout = time.DateTime
				} else if strings.Contains(s3, ":") {
					layout = "2006-01-02 15:04:05Z07:00" // older format with colon in timezone
				} else {
					layout = "2006-01-02 15:04:05Z0700" // older format without colon in timezone
				}
			}
			return
		})

	// other common formats
	RegisterDateFormat(`(\d{2}/\d{2} \d{2}:\d{2}:\d{2}(AM|PM) '\d{2} [+-]\d{4})`, simple(time.Layout))
	RegisterDateFormat(`[A-Z][a-z]{2,10} [A-Z][a-z]{2,10}\s+\d{1,2} \d{2}:\d{2}:\d{2} \d{4}`, simple(time.ANSIC))
	RegisterDateFormat(`([A-Z][a-z]{2,10} [A-Z][a-z]{2,10}\s+\d{1,2} \d{2}:\d{2}:\d{2}(\.\d+)? [A-Z]{3} \d{4})`, simple(time.UnixDate))
	RegisterDateFormat(`([A-Z][a-z]{2,10} [A-Z][a-z]{2,10} \d{2} \d{2}:\d{2}:\d{2}(\.\d+)? [+-]\d{4} \d{4})`, simple(time.RubyDate))
	RegisterDateFormat(`(\d{2} [A-Z][a-z]{2,10} \d{2} \d{2}:\d{2}(\.\d+)? [A-Z]{3})`, simple(time.RFC822))
	RegisterDateFormat(`(\d{2} [A-Z][a-z]{2,10} \d{2} \d{2}:\d{2}(\.\d+)? [+-]\d{4})`, simple(time.RFC822Z))
	RegisterDateFormat(`([A-Z][a-z]{2,10}, \d{2}-[A-Z][a-z]{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)? [A-Z]{3})`, simple(time.RFC850))
	RegisterDateFormat(`([A-Z][a-z]{2,10}, \d{2} [A-Z][a-z]{2,10} \d{4} \d{2}:\d{2}:\d{2}(\.\d+)? [A-Z]{3})`, simple(time.RFC1123))
	RegisterDateFormat(`([A-Z][a-z]{2,10}, \d{2} [A-Z][a-z]{2,10} \d{4} \d{2}:\d{2}:\d{2}(\.\d+)? [+-]\d{4})`, simple(time.RFC1123Z))

	// nginx format (common in access logs)
	RegisterDateFormat(
		`\d{2}/\w{3}/\d{4}([ :])\d{2}:\d{2}:\d{2}(\.\d+)?( [\+\-]?\d{4})?`,
		func(value string, matches []int) (layout string, start, end int) {
			start, end = matches[0], matches[1]
			s1 := val(value, matches[2:4])
			s3 := val(value, matches[6:8])
			if s1 == ":" {
				if s3 == "" {
					layout = "02/Jan/2006:15:04:05"
				} else {
					layout = "02/Jan/2006:15:04:05 Z0700"
				}
			} else {
				if s3 == "" {
					layout = "02/Jan/2006 15:04:05"
				} else {
					layout = "02/Jan/2006 15:04:05 Z0700"
				}
			}
			return
		})
	RegisterDateFormat(`\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}`, simple("2006/01/02 15:04:05"))

	// partial date/time formats
	RegisterDateFormat(`^[^0-9]*(\d{4}-\d{2}-\d{2})[^0-9]*$`, simple(time.DateOnly))
	RegisterDateFormat(`^[^0-9]*(\d{2}:\d{2}:\d{2})[^0-9]*$`, simple(time.TimeOnly))
	RegisterDateFormat(`^[^0-9]*(\d{1,2}:\d{2}(AM|PM))`, simple(time.Kitchen))
}

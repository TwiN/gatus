// Package now is a time toolkit for golang.
//
// More details README here: https://github.com/jinzhu/now
//
//  import "github.com/jinzhu/now"
//
//  now.BeginningOfMinute() // 2013-11-18 17:51:00 Mon
//  now.BeginningOfDay()    // 2013-11-18 00:00:00 Mon
//  now.EndOfDay()          // 2013-11-18 23:59:59.999999999 Mon
package now

import "time"

// WeekStartDay set week start day, default is sunday
var WeekStartDay = time.Sunday

// TimeFormats default time formats will be parsed as
var TimeFormats = []string{
	"2006", "2006-1", "2006-1-2", "2006-1-2 15", "2006-1-2 15:4", "2006-1-2 15:4:5", "1-2",
	"15:4:5", "15:4", "15",
	"15:4:5 Jan 2, 2006 MST", "2006-01-02 15:04:05.999999999 -0700 MST", "2006-01-02T15:04:05-07:00",
	"2006.1.2", "2006.1.2 15:04:05", "2006.01.02", "2006.01.02 15:04:05",
	"1/2/2006", "1/2/2006 15:4:5", "2006/01/02", "2006/01/02 15:04:05",
	time.ANSIC, time.UnixDate, time.RubyDate, time.RFC822, time.RFC822Z, time.RFC850,
	time.RFC1123, time.RFC1123Z, time.RFC3339, time.RFC3339Nano,
	time.Kitchen, time.Stamp, time.StampMilli, time.StampMicro, time.StampNano,
}

// Config configuration for now package
type Config struct {
	WeekStartDay time.Weekday
	TimeLocation *time.Location
	TimeFormats  []string
}

// DefaultConfig default config
var DefaultConfig *Config

// New initialize Now based on configuration
func (config *Config) With(t time.Time) *Now {
	return &Now{Time: t, Config: config}
}

// Parse parse string to time based on configuration
func (config *Config) Parse(strs ...string) (time.Time, error) {
	if config.TimeLocation == nil {
		return config.With(time.Now()).Parse(strs...)
	} else {
		return config.With(time.Now().In(config.TimeLocation)).Parse(strs...)
	}
}

// MustParse must parse string to time or will panic
func (config *Config) MustParse(strs ...string) time.Time {
	if config.TimeLocation == nil {
		return config.With(time.Now()).MustParse(strs...)
	} else {
		return config.With(time.Now().In(config.TimeLocation)).MustParse(strs...)
	}
}

// Now now struct
type Now struct {
	time.Time
	*Config
}

// With initialize Now with time
func With(t time.Time) *Now {
	config := DefaultConfig
	if config == nil {
		config = &Config{
			WeekStartDay: WeekStartDay,
			TimeFormats:  TimeFormats,
		}
	}

	return &Now{Time: t, Config: config}
}

// New initialize Now with time
func New(t time.Time) *Now {
	return With(t)
}

// BeginningOfMinute beginning of minute
func BeginningOfMinute() time.Time {
	return With(time.Now()).BeginningOfMinute()
}

// BeginningOfHour beginning of hour
func BeginningOfHour() time.Time {
	return With(time.Now()).BeginningOfHour()
}

// BeginningOfDay beginning of day
func BeginningOfDay() time.Time {
	return With(time.Now()).BeginningOfDay()
}

// BeginningOfWeek beginning of week
func BeginningOfWeek() time.Time {
	return With(time.Now()).BeginningOfWeek()
}

// BeginningOfMonth beginning of month
func BeginningOfMonth() time.Time {
	return With(time.Now()).BeginningOfMonth()
}

// BeginningOfQuarter beginning of quarter
func BeginningOfQuarter() time.Time {
	return With(time.Now()).BeginningOfQuarter()
}

// BeginningOfYear beginning of year
func BeginningOfYear() time.Time {
	return With(time.Now()).BeginningOfYear()
}

// EndOfMinute end of minute
func EndOfMinute() time.Time {
	return With(time.Now()).EndOfMinute()
}

// EndOfHour end of hour
func EndOfHour() time.Time {
	return With(time.Now()).EndOfHour()
}

// EndOfDay end of day
func EndOfDay() time.Time {
	return With(time.Now()).EndOfDay()
}

// EndOfWeek end of week
func EndOfWeek() time.Time {
	return With(time.Now()).EndOfWeek()
}

// EndOfMonth end of month
func EndOfMonth() time.Time {
	return With(time.Now()).EndOfMonth()
}

// EndOfQuarter end of quarter
func EndOfQuarter() time.Time {
	return With(time.Now()).EndOfQuarter()
}

// EndOfYear end of year
func EndOfYear() time.Time {
	return With(time.Now()).EndOfYear()
}

// Monday monday
func Monday() time.Time {
	return With(time.Now()).Monday()
}

// Sunday sunday
func Sunday() time.Time {
	return With(time.Now()).Sunday()
}

// EndOfSunday end of sunday
func EndOfSunday() time.Time {
	return With(time.Now()).EndOfSunday()
}

// Parse parse string to time
func Parse(strs ...string) (time.Time, error) {
	return With(time.Now()).Parse(strs...)
}

// ParseInLocation parse string to time in location
func ParseInLocation(loc *time.Location, strs ...string) (time.Time, error) {
	return With(time.Now().In(loc)).Parse(strs...)
}

// MustParse must parse string to time or will panic
func MustParse(strs ...string) time.Time {
	return With(time.Now()).MustParse(strs...)
}

// MustParseInLocation must parse string to time in location or will panic
func MustParseInLocation(loc *time.Location, strs ...string) time.Time {
	return With(time.Now().In(loc)).MustParse(strs...)
}

// Between check now between the begin, end time or not
func Between(time1, time2 string) bool {
	return With(time.Now()).Between(time1, time2)
}

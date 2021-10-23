package maintenance

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	errInvalidMaintenanceStartFormat = errors.New("invalid maintenance start format: must be hh:mm, between 00:00 and 23:59 inclusively (e.g. 23:00)")
	errInvalidMaintenanceDuration    = errors.New("invalid maintenance duration: must be bigger than 0 (e.g. 30m)")
	errInvalidDayName                = fmt.Errorf("invalid value specified for 'on'. supported values are %s", longDayNames)

	longDayNames = []string{
		"Sunday",
		"Monday",
		"Tuesday",
		"Wednesday",
		"Thursday",
		"Friday",
		"Saturday",
	}
)

// Config allows for the configuration of a maintenance period.
// During this maintenance period, no alerts will be sent.
//
// Uses UTC.
type Config struct {
	Enabled  *bool         `yaml:"enabled"`  // Whether the maintenance period is enabled. Enabled by default if nil.
	Start    string        `yaml:"start"`    // Time at which the maintenance period starts (e.g. 23:00)
	Duration time.Duration `yaml:"duration"` // Duration of the maintenance period (e.g. 4h)

	// Every is a list of days of the week during which maintenance period applies.
	// See longDayNames for list of valid values.
	// Every day if empty.
	Every []string `yaml:"every"`

	durationToStartFromMidnight time.Duration
	timeLocation                *time.Location
}

func GetDefaultConfig() *Config {
	defaultValue := false
	return &Config{
		Enabled: &defaultValue,
	}
}

// IsEnabled returns whether maintenance is enabled or not
func (c Config) IsEnabled() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// ValidateAndSetDefaults validates the maintenance configuration and sets the default values if necessary.
//
// Must be called once in the application's lifecycle before IsUnderMaintenance is called, since it
// also sets durationToStartFromMidnight.
func (c *Config) ValidateAndSetDefaults() error {
	if c == nil || !c.IsEnabled() {
		// Don't waste time validating if maintenance is not enabled.
		return nil
	}
	for _, day := range c.Every {
		isDayValid := false
		for _, longDayName := range longDayNames {
			if day == longDayName {
				isDayValid = true
				break
			}
		}
		if !isDayValid {
			return errInvalidDayName
		}
	}
	var err error
	c.durationToStartFromMidnight, err = hhmmToDuration(c.Start)
	if err != nil {
		return err
	}
	if c.Duration <= 0 || c.Duration >= 24*time.Hour {
		return errInvalidMaintenanceDuration
	}
	return nil
}

// IsUnderMaintenance checks whether the endpoints that Gatus monitors are within the configured maintenance window
func (c Config) IsUnderMaintenance() bool {
	if !c.IsEnabled() {
		return false
	}
	now := time.Now().UTC()
	var dayWhereMaintenancePeriodWouldStart time.Time
	if now.Hour() >= int(c.durationToStartFromMidnight.Hours()) {
		dayWhereMaintenancePeriodWouldStart = now.Truncate(24 * time.Hour)
	} else {
		dayWhereMaintenancePeriodWouldStart = now.Add(-c.Duration).Truncate(24 * time.Hour)
	}
	hasMaintenanceEveryDay := len(c.Every) == 0
	hasMaintenancePeriodScheduledToStartOnThatWeekday := c.hasDay(dayWhereMaintenancePeriodWouldStart.Weekday().String())
	if !hasMaintenanceEveryDay && !hasMaintenancePeriodScheduledToStartOnThatWeekday {
		// The day when the maintenance period would start is not scheduled
		// to have any maintenance, so we can just return false.
		return false
	}
	startOfMaintenancePeriod := dayWhereMaintenancePeriodWouldStart.Add(c.durationToStartFromMidnight)
	endOfMaintenancePeriod := startOfMaintenancePeriod.Add(c.Duration)
	return now.After(startOfMaintenancePeriod) && now.Before(endOfMaintenancePeriod)
}

func (c Config) hasDay(day string) bool {
	for _, d := range c.Every {
		if d == day {
			return true
		}
	}
	return false
}

func hhmmToDuration(s string) (time.Duration, error) {
	if len(s) != 5 {
		return 0, errInvalidMaintenanceStartFormat
	}
	var hours, minutes int
	var err error
	if hours, err = extractNumericalValueFromPotentiallyZeroPaddedString(s[:2]); err != nil {
		return 0, err
	}
	if minutes, err = extractNumericalValueFromPotentiallyZeroPaddedString(s[3:5]); err != nil {
		return 0, err
	}
	duration := (time.Duration(hours) * time.Hour) + (time.Duration(minutes) * time.Minute)
	if hours < 0 || hours > 23 || minutes < 0 || minutes > 59 || duration < 0 || duration >= 24*time.Hour {
		return 0, errInvalidMaintenanceStartFormat
	}
	return duration, nil
}

func extractNumericalValueFromPotentiallyZeroPaddedString(s string) (int, error) {
	return strconv.Atoi(strings.TrimPrefix(s, "0"))
}

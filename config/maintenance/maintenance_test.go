package maintenance

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func TestGetDefaultConfig(t *testing.T) {
	if *GetDefaultConfig().Enabled {
		t.Fatal("expected default config to be disabled by default")
	}
}

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	yes, no := true, false
	scenarios := []struct {
		name          string
		cfg           *Config
		expectedError error
	}{
		{
			name:          "nil",
			cfg:           nil,
			expectedError: nil,
		},
		{
			name: "disabled",
			cfg: &Config{
				Enabled: &no,
			},
			expectedError: nil,
		},
		{
			name: "invalid-day",
			cfg: &Config{
				Every: []string{"invalid-day"},
			},
			expectedError: errInvalidDayName,
		},
		{
			name: "invalid-day",
			cfg: &Config{
				Every: []string{"invalid-day"},
			},
			expectedError: errInvalidDayName,
		},
		{
			name: "invalid-start-format",
			cfg: &Config{
				Start: "0000",
			},
			expectedError: errInvalidMaintenanceStartFormat,
		},
		{
			name: "invalid-start-hours",
			cfg: &Config{
				Start: "25:00",
			},
			expectedError: errInvalidMaintenanceStartFormat,
		},
		{
			name: "invalid-start-minutes",
			cfg: &Config{
				Start: "0:61",
			},
			expectedError: errInvalidMaintenanceStartFormat,
		},
		{
			name: "invalid-start-minutes-non-numerical",
			cfg: &Config{
				Start: "00:zz",
			},
			expectedError: strconv.ErrSyntax,
		},
		{
			name: "invalid-start-hours-non-numerical",
			cfg: &Config{
				Start: "zz:00",
			},
			expectedError: strconv.ErrSyntax,
		},
		{
			name: "invalid-duration",
			cfg: &Config{
				Start:    "23:00",
				Duration: 0,
			},
			expectedError: errInvalidMaintenanceDuration,
		},
		{
			name: "invalid-timezone",
			cfg: &Config{
				Start:    "23:00",
				Duration: time.Hour,
				Timezone: "invalid-timezone",
			},
			expectedError: errInvalidTimezone,
		},
		{
			name: "every-day-at-2300",
			cfg: &Config{
				Start:    "23:00",
				Duration: time.Hour,
			},
			expectedError: nil,
		},
		{
			name: "every-day-explicitly-at-2300",
			cfg: &Config{
				Start:    "23:00",
				Duration: time.Hour,
				Every:    []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"},
			},
			expectedError: nil,
		},
		{
			name: "every-monday-at-0000",
			cfg: &Config{
				Start:    "00:00",
				Duration: 30 * time.Minute,
				Every:    []string{"Monday"},
			},
			expectedError: nil,
		},
		{
			name: "every-friday-and-sunday-at-0000-explicitly-enabled",
			cfg: &Config{
				Enabled:  &yes,
				Start:    "08:00",
				Duration: 8 * time.Hour,
				Every:    []string{"Friday", "Sunday"},
			},
			expectedError: nil,
		},
		{
			name: "timezone-amsterdam",
			cfg: &Config{
				Start:    "23:00",
				Duration: time.Hour,
				Timezone: "Europe/Amsterdam",
			},
			expectedError: nil,
		},
		{
			name: "timezone-cet",
			cfg: &Config{
				Start:    "23:00",
				Duration: time.Hour,
				Timezone: "CET",
			},
			expectedError: nil,
		},
		{
			name: "timezone-etc-plus-5",
			cfg: &Config{
				Start:    "23:00",
				Duration: time.Hour,
				Timezone: "Etc/GMT+5",
			},
			expectedError: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.cfg.ValidateAndSetDefaults()
			if !errors.Is(err, scenario.expectedError) {
				t.Errorf("expected %v, got %v", scenario.expectedError, err)
			}
		})
	}
}

func TestConfig_IsUnderMaintenance(t *testing.T) {
	yes, no := true, false
	now := time.Now().UTC()
	scenarios := []struct {
		name                     string
		cfg                      *Config
		expectedUnderMaintenance bool
	}{
		{
			name: "disabled",
			cfg: &Config{
				Enabled: &no,
			},
			expectedUnderMaintenance: false,
		},
		{
			name: "under-maintenance-explicitly-enabled",
			cfg: &Config{
				Enabled:  &yes,
				Start:    fmt.Sprintf("%02d:00", now.Hour()),
				Duration: 2 * time.Hour,
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "under-maintenance-starting-now-for-2h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", now.Hour()),
				Duration: 2 * time.Hour,
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "under-maintenance-starting-now-for-8h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", now.Hour()),
				Duration: 8 * time.Hour,
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "under-maintenance-starting-now-for-8h-explicit-days",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", now.Hour()),
				Duration: 8 * time.Hour,
				Every:    []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"},
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "under-maintenance-starting-now-for-23h-explicit-days",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", now.Hour()),
				Duration: 23 * time.Hour,
				Every:    []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"},
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "under-maintenance-starting-4h-ago-for-8h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", normalizeHour(now.Hour()-4)),
				Duration: 8 * time.Hour,
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "under-maintenance-starting-22h-ago-for-23h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", normalizeHour(now.Hour()-22)),
				Duration: 23 * time.Hour,
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "under-maintenance-starting-22h-ago-for-24h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", normalizeHour(now.Hour()-22)),
				Duration: 24 * time.Hour,
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "under-maintenance-amsterdam-timezone-starting-now-for-2h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", inTimezone(now, "Europe/Amsterdam", t).Hour()),
				Duration: 2 * time.Hour,
				Timezone: "Europe/Amsterdam",
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "under-maintenance-perth-timezone-starting-now-for-2h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", inTimezone(now, "Australia/Perth", t).Hour()),
				Duration: 2 * time.Hour,
				Timezone: "Australia/Perth",
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "not-under-maintenance-los-angeles-timezone-starting-now-for-2h-today",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", now.Hour()),
				Duration: 2 * time.Hour,
				Timezone: "America/Los_Angeles",
				Every:    []string{now.Weekday().String()},
			},
			expectedUnderMaintenance: false,
		},
		{
			name: "under-maintenance-utc-timezone-starting-now-for-2h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", now.Hour()),
				Duration: 2 * time.Hour,
				Timezone: "UTC",
			},
			expectedUnderMaintenance: true,
		},
		{
			name: "not-under-maintenance-starting-4h-ago-for-3h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", normalizeHour(now.Hour()-4)),
				Duration: 3 * time.Hour,
			},
			expectedUnderMaintenance: false,
		},
		{
			name: "not-under-maintenance-starting-5h-ago-for-1h",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", normalizeHour(now.Hour()-5)),
				Duration: time.Hour,
			},
			expectedUnderMaintenance: false,
		},
		{
			name: "not-under-maintenance-today",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", now.Hour()),
				Duration: time.Hour,
				Every:    []string{now.Add(48 * time.Hour).Weekday().String()},
			},
			expectedUnderMaintenance: false,
		},
		{
			name: "not-under-maintenance-today-with-24h-duration",
			cfg: &Config{
				Start:    fmt.Sprintf("%02d:00", now.Hour()),
				Duration: 24 * time.Hour,
				Every:    []string{now.Add(48 * time.Hour).Weekday().String()},
			},
			expectedUnderMaintenance: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Log(scenario.cfg.Start)
			t.Log(now)
			if err := scenario.cfg.ValidateAndSetDefaults(); err != nil {
				t.Fatal("validation shouldn't have returned an error, got", err)
			}
			isUnderMaintenance := scenario.cfg.IsUnderMaintenance()
			if isUnderMaintenance != scenario.expectedUnderMaintenance {
				t.Errorf("expectedUnderMaintenance %v, got %v", scenario.expectedUnderMaintenance, isUnderMaintenance)
				t.Logf("start=%v; duration=%v; now=%v", scenario.cfg.Start, scenario.cfg.Duration, time.Now().UTC())
			}
		})
	}
}

func normalizeHour(hour int) int {
	if hour < 0 {
		return hour + 24
	}
	return hour
}

func inTimezone(passedTime time.Time, timezone string, t *testing.T) time.Time {
	timezoneLocation, err := time.LoadLocation(timezone)
	if err != nil {
		t.Fatalf("timezone %s did not load", timezone)
	}
	return passedTime.In(timezoneLocation)
}

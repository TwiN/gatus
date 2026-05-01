package ui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// CustomDuration is a time.Duration that supports YAML unmarshaling from
// human-readable strings like "30d", "24h", "7d", "1h30m", etc.
type CustomDuration time.Duration

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (d *CustomDuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	parsed, err := ParseHumanDuration(s)
	if err != nil {
		return err
	}
	*d = CustomDuration(parsed)
	return nil
}

// Duration returns the underlying time.Duration.
func (d CustomDuration) Duration() time.Duration {
	return time.Duration(d)
}

// ParseHumanDuration parses a duration string that may include days (e.g., "30d", "7d", "24h").
// It also supports standard Go duration formats like "1h30m".
func ParseHumanDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration string")
	}
	// Try standard Go duration first
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}
	// Try to parse as "<number>d" (days)
	if strings.HasSuffix(s, "d") {
		numStr := strings.TrimSuffix(s, "d")
		value, err := strconv.Atoi(numStr)
		if err != nil {
			return 0, fmt.Errorf("invalid duration format: %s", s)
		}
		if value <= 0 {
			return 0, fmt.Errorf("duration must be positive: %s", s)
		}
		return time.Duration(value) * 24 * time.Hour, nil
	}
	return 0, fmt.Errorf("invalid duration format: %s (e.g., 30d, 24h, 1h30m)", s)
}

// Config is the UI configuration for endpoint.Endpoint
type Config struct {
	// HideConditions whether to hide the condition results on the UI
	HideConditions bool `yaml:"hide-conditions"`

	// HideHostname whether to hide the hostname in the Result
	HideHostname bool `yaml:"hide-hostname"`

	// HideURL whether to ensure the URL is not displayed in the results. Useful if the URL contains a token.
	HideURL bool `yaml:"hide-url"`

	// HidePort whether to hide the port in the Result
	HidePort bool `yaml:"hide-port"`

	// HideErrors whether to hide the errors in the Result
	HideErrors bool `yaml:"hide-errors"`

	// DontResolveFailedConditions whether to resolve failed conditions in the Result for display in the UI
	DontResolveFailedConditions bool `yaml:"dont-resolve-failed-conditions"`

	// ResolveSuccessfulConditions whether to resolve successful conditions in the Result for display in the UI
	ResolveSuccessfulConditions bool `yaml:"resolve-successful-conditions"`

	// Badge is the configuration for the badges generated
	Badge *Badge `yaml:"badge"`

	// Period defines a fixed time window for the "Recent Checks" graph and uptime display.
	// When set, the "Recent Checks" section and uptime badges will use this period
	// instead of the default behavior.
	// Supported formats: "1h", "24h", "7d", "14d", "30d", "60d", "90d"
	// Minimum: 1h, Maximum: 90d
	// Default: 0 (uses default behavior based on endpoint interval)
	Period CustomDuration `yaml:"period,omitempty"`
}

type Badge struct {
	ResponseTime *ResponseTime `yaml:"response-time"`
}

type ResponseTime struct {
	Thresholds []int `yaml:"thresholds"`
}

var (
	ErrInvalidBadgeResponseTimeConfig = errors.New("invalid response time badge configuration: expected parameter 'response-time' to have 5 ascending numerical values")
	ErrInvalidPeriod                  = errors.New("invalid period configuration: period must be between 1h and 90d")
)

// ValidateAndSetDefaults validates the UI configuration and sets the default values
func (config *Config) ValidateAndSetDefaults() error {
	if config.Badge != nil {
		if len(config.Badge.ResponseTime.Thresholds) != 5 {
			return ErrInvalidBadgeResponseTimeConfig
		}
		for i := 4; i > 0; i-- {
			if config.Badge.ResponseTime.Thresholds[i] < config.Badge.ResponseTime.Thresholds[i-1] {
				return ErrInvalidBadgeResponseTimeConfig
			}
		}
	} else {
		config.Badge = GetDefaultConfig().Badge
	}
	if config.Period != 0 {
		d := config.Period.Duration()
		if d < time.Hour || d > 90*24*time.Hour {
			return ErrInvalidPeriod
		}
	}
	return nil
}

// GetDefaultConfig retrieves the default UI configuration
func GetDefaultConfig() *Config {
	return &Config{
		HideHostname:                false,
		HideURL:                     false,
		HidePort:                    false,
		HideErrors:                  false,
		DontResolveFailedConditions: false,
		ResolveSuccessfulConditions: false,
		HideConditions:              false,
		Badge: &Badge{
			ResponseTime: &ResponseTime{
				Thresholds: []int{50, 200, 300, 500, 750},
			},
		},
	}
}

// PeriodDurationString returns the period as a duration string suitable for API calls
// (e.g., "30d", "7d", "24h", "1h"). Returns empty string if period is not set.
func (config *Config) PeriodDurationString() string {
	if config.Period == 0 {
		return ""
	}
	hours := config.Period.Duration().Hours()
	if hours >= 24 && int(hours)%24 == 0 {
		return fmt.Sprintf("%dd", int(hours)/24)
	}
	return fmt.Sprintf("%dh", int(hours))
}

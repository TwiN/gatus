package ui

import "errors"

// Config is the UI configuration for core.Endpoint
type Config struct {
	// HideHostname whether to hide the hostname in the Result
	HideHostname bool `yaml:"hide-hostname"`
	// DontResolveFailedConditions whether to resolve failed conditions in the Result for display in the UI
	DontResolveFailedConditions bool `yaml:"dont-resolve-failed-conditions"`
	ResponseTimerBadgeAwesome   int  `yaml:"response-timer-badge-awesome"`
	ResponseTimerBadgeGreat     int  `yaml:"response-timer-badge-great"`
	ResponseTimerBadgeGood      int  `yaml:"response-timer-badge-good"`
	ResponseTimerBadgePassable  int  `yaml:"response-timer-badge-passable"`
	ResponseTimerBadgeBad       int  `yaml:"response-timer-badge-bad"`
}

var (
	ErrInvalidUiBadgeTimeConfig = errors.New("invalid endpoint ui configuration - invalid badgetime configuration: You need to set all responsetime settings and they have to be in an ascending value range")
)

func (config *Config) Validate() error {

	if config.ResponseTimerBadgeBad >
		config.ResponseTimerBadgePassable {
		if config.ResponseTimerBadgePassable >
			config.ResponseTimerBadgeGood {
			if config.ResponseTimerBadgeGood >
				config.ResponseTimerBadgeGreat {
				if config.ResponseTimerBadgeGreat >
					config.ResponseTimerBadgeAwesome {
					return nil
				}
			}
		}
	}

	return ErrInvalidUiBadgeTimeConfig
}

func (config *Config) DefaultValues() {

	config.HideHostname = false
	config.DontResolveFailedConditions = false
	config.ResponseTimerBadgeAwesome = 50
	config.ResponseTimerBadgeGreat = 200
	config.ResponseTimerBadgeGood = 300
	config.ResponseTimerBadgePassable = 500
	config.ResponseTimerBadgeBad = 750

}

// GetDefaultConfig retrieves the default UI configuration
func GetDefaultConfig() *Config {
	return &Config{
		HideHostname:                false,
		DontResolveFailedConditions: false,
		ResponseTimerBadgeAwesome:   50,
		ResponseTimerBadgeGreat:     200,
		ResponseTimerBadgeGood:      300,
		ResponseTimerBadgePassable:  500,
		ResponseTimerBadgeBad:       750,
	}
}

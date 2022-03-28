package ui

import "errors"

// Config is the UI configuration for core.Endpoint
type Config struct {
	// HideHostname whether to hide the hostname in the Result
	HideHostname bool `yaml:"hide-hostname"`
	// DontResolveFailedConditions whether to resolve failed conditions in the Result for display in the UI
	DontResolveFailedConditions bool   `yaml:"dont-resolve-failed-conditions"`
	Badge                       *Badge `yaml:"badge"`
}

type Badge struct {
	Responsetime *Thresholds `yaml:"response-time"`
}

type Thresholds struct {
	Thresholds []int `yaml:"thresholds"`
}

var (
	ErrInvalidUiBadgeTimeConfig = errors.New("invalid endpoint ui configuration - invalid badgetime configuration: You need to set all responsetime settings and they have to be in an ascending value range")
)

func (config *Config) Validate() error {

	if len(config.Badge.Responsetime.Thresholds) == 5 {
		if config.Badge.Responsetime.Thresholds[4] >
			config.Badge.Responsetime.Thresholds[3] {
			if config.Badge.Responsetime.Thresholds[3] >
				config.Badge.Responsetime.Thresholds[2] {
				if config.Badge.Responsetime.Thresholds[2] >
					config.Badge.Responsetime.Thresholds[1] {
					if config.Badge.Responsetime.Thresholds[1] >
						config.Badge.Responsetime.Thresholds[0] {
						return nil
					}
				}
			}
		}
	}

	return ErrInvalidUiBadgeTimeConfig
}

// GetDefaultConfig retrieves the default UI configuration
func GetDefaultConfig() *Config {
	return &Config{
		HideHostname:                false,
		DontResolveFailedConditions: false,
		Badge: &Badge{
			Responsetime: &Thresholds{
				Thresholds: []int{50, 200, 300, 500, 75},
			},
		},
	}
}

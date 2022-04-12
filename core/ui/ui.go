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
	ResponseTime []int `yaml:"response-time"`
}

var (
	ErrInvalidBadgeResponseTimeConfig = errors.New("invalid response time badge configuration: expected parameter 'response-time' to have 5 ascending numerical values")
)

func (config *Config) ValidateAndSetDefaults() error {

	if config.Badge != nil {
		if config.Badge.ResponseTime != nil {
			if len(config.Badge.ResponseTime) != 5 {
				return ErrInvalidBadgeResponseTimeConfig
			}
			for i := 4; i > 0; i-- {
				if config.Badge.ResponseTime[i] < config.Badge.ResponseTime[i-1] {
					return ErrInvalidBadgeResponseTimeConfig
				}
			}
		}
		config.Badge.ResponseTime = GetDefaultConfig().Badge.ResponseTime
	}

	config.Badge = GetDefaultConfig().Badge
	return nil
}

// GetDefaultConfig retrieves the default UI configuration
func GetDefaultConfig() *Config {
	return &Config{
		HideHostname:                false,
		DontResolveFailedConditions: false,
		Badge: &Badge{
			ResponseTime: []int{50, 200, 300, 500, 750},
		},
	}
}

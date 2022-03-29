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
	ResponseTime *Thresholds `yaml:"response-time"`
}

type Thresholds struct {
	Thresholds []int `yaml:"thresholds"`
}

var (
	ErrInvalidUiBadgeTimeConfig = errors.New("invalid endpoint ui configuration - invalid badgetime configuration: You need to set all responsetime settings and they have to be in an ascending value range")
)

func (config *Config) Validate() error {

	if len(config.Badge.ResponseTime.Thresholds) != 5 {
		return ErrInvalidUiBadgeTimeConfig
	}
	for i := 4; i > 0; i-- {
		if config.Badge.ResponseTime.Thresholds[i] < config.Badge.ResponseTime.Thresholds[i-1] {
			return ErrInvalidUiBadgeTimeConfig
		}
	}
	return nil

}

// GetDefaultConfig retrieves the default UI configuration
func GetDefaultConfig() *Config {
	return &Config{
		HideHostname:                false,
		DontResolveFailedConditions: false,
		Badge: &Badge{
			ResponseTime: &Thresholds{
				Thresholds: []int{50, 200, 300, 500, 750},
			},
		},
	}
}

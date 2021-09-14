package ui

// Config is the UI configuration for services
type Config struct {
	// HideHostname whether to hide the hostname in the Result
	HideHostname bool `yaml:"hide-hostname"`
	// DontResolveFailedConditions whether to resolve failed conditions in the Result for display in the UI
	DontResolveFailedConditions bool `yaml:"dont-resolve-failed-conditions"`
}

// GetDefaultConfig retrieves the default UI configuration
func GetDefaultConfig() *Config {
	return &Config{
		HideHostname:                false,
		DontResolveFailedConditions: false,
	}
}

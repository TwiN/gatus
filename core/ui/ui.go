package ui

// Config is the UI configuration for core.Endpoint
type Config struct {
	// HideHostname whether to hide the hostname in the Result
	HideHostname bool `yaml:"hide-hostname"`
	// HideURL whether to hide the URL in the Result
	HideURL bool `yaml:"hide-url"`
	// DontResolveFailedConditions whether to resolve failed conditions in the Result for display in the UI
	DontResolveFailedConditions bool `yaml:"dont-resolve-failed-conditions"`
}

// GetDefaultConfig retrieves the default UI configuration
func GetDefaultConfig() *Config {
	return &Config{
		HideHostname:                false,
		HideURL:                     false,
		DontResolveFailedConditions: false,
	}
}

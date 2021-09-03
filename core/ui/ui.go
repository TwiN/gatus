package ui

// Config is the UI configuration for services
type Config struct {
	HideHostname bool `yaml:"hide-hostname"` // Whether to hide the hostname in the Result
}

// GetDefaultConfig retrieves the default UI configuration
func GetDefaultConfig() *Config {
	return &Config{
		HideHostname: false,
	}
}

package distributed

type Config struct {
	Enabled *bool `yaml:"enabled"` // Whether the maintenance period is enabled. Enabled by default if nil.
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

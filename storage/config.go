package storage

// Config is the configuration for alerting providers
type Config struct {
	// File is the path of the file to use for persistence
	// If blank, persistence is disabled.
	File string `yaml:"file"`
}

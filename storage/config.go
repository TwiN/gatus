package storage

// Config is the configuration for alerting providers
type Config struct {
	// File is the path of the file to use when using file.Store
	File string `yaml:"file"`
}

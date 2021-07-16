package storage

// Config is the configuration for storage
type Config struct {
	// File is the path of the file to use for persistence
	// If blank, persistence is disabled
	File string `yaml:"file"`

	// Type of store
	// If blank, uses the default in-memory store
	Type Type `yaml:"type"`
}

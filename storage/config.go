package storage

import "errors"

var (
	ErrSQLStorageRequiresFile = errors.New("sql storage requires a non-empty file to be defined")
)

// Config is the configuration for storage
type Config struct {
	// File is the path of the file to use for persistence
	// If blank, persistence is disabled
	//
	// XXX: Rename to path for v4.0.0
	File string `yaml:"file"`

	// Type of store
	// If blank, uses the default in-memory store
	Type Type `yaml:"type"`
}

// ValidateAndSetDefaults validates the configuration and sets the default values (if applicable)
func (c *Config) ValidateAndSetDefaults() error {
	if (c.Type == TypePostgres || c.Type == TypeSQLite) && len(c.File) == 0 {
		return ErrSQLStorageRequiresFile
	}
	return nil
}

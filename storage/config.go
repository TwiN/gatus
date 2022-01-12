package storage

import (
	"errors"
	"log"
)

var (
	ErrSQLStorageRequiresPath          = errors.New("sql storage requires a non-empty path to be defined")
	ErrMemoryStorageDoesNotSupportPath = errors.New("memory storage does not support persistence, use sqlite if you want persistence on file")
	ErrCannotSetBothFileAndPath        = errors.New("file has been deprecated in favor of path: you cannot set both of them")
)

// Config is the configuration for storage
type Config struct {
	// Path is the path used by the store to achieve persistence
	// If blank, persistence is disabled.
	// Note that not all Type support persistence
	Path string `yaml:"path"`

	// File is the path of the file to use for persistence
	// If blank, persistence is disabled
	//
	// Deprecated
	File string `yaml:"file"`

	// Type of store
	// If blank, uses the default in-memory store
	Type Type `yaml:"type"`
}

// ValidateAndSetDefaults validates the configuration and sets the default values (if applicable)
func (c *Config) ValidateAndSetDefaults() error {
	if len(c.File) > 0 && len(c.Path) > 0 { // XXX: Remove for v4.0.0
		return ErrCannotSetBothFileAndPath
	} else if len(c.File) > 0 { // XXX: Remove for v4.0.0
		log.Println("WARNING: Your configuration is using 'storage.file', which is deprecated in favor of 'storage.path'")
		log.Println("WARNING: storage.file will be completely removed in v4.0.0, so please update your configuration")
		log.Println("WARNING: See https://github.com/TwiN/gatus/issues/197")
		c.Path = c.File
	}
	if c.Type == "" {
		c.Type = TypeMemory
	}
	if (c.Type == TypePostgres || c.Type == TypeSQLite) && len(c.Path) == 0 {
		return ErrSQLStorageRequiresPath
	}
	if c.Type == TypeMemory && len(c.Path) > 0 {
		log.Println("WARNING: Your configuration is using a storage of type memory with persistence, which has been deprecated")
		log.Println("WARNING: As of v4.0.0, the default storage type (memory) will not support persistence.")
		log.Println("WARNING: If you want persistence, use 'storage.type: sqlite' instead of 'storage.type: memory'")
		log.Println("WARNING: See https://github.com/TwiN/gatus/issues/198")
		// XXX: Uncomment the following line for v4.0.0
		//return ErrMemoryStorageDoesNotSupportPath
	}
	return nil
}

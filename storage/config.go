package storage

import (
	"errors"
	"time"
)

const (
	DefaultMaximumNumberOfResults = 100
	DefaultMaximumNumberOfEvents  = 50
	DefaultUptimeRetention        = 30 * 24 * time.Hour
)

var (
	ErrSQLStorageRequiresPath          = errors.New("sql storage requires a non-empty path to be defined")
	ErrMemoryStorageDoesNotSupportPath = errors.New("memory storage does not support persistence, use sqlite if you want persistence on file")
)

// Config is the configuration for storage
type Config struct {
	// Path is the path used by the store to achieve persistence
	// If blank, persistence is disabled.
	// Note that not all Type support persistence
	Path string `yaml:"path"`

	// Type of store
	// If blank, uses the default in-memory store
	Type Type `yaml:"type"`

	// Caching is whether to enable caching.
	// This is used to drastically decrease read latency by pre-emptively caching writes
	// as they happen, also known as the write-through caching strategy.
	// Does not apply if Config.Type is not TypePostgres or TypeSQLite.
	Caching bool `yaml:"caching,omitempty"`

	// MaximumNumberOfResults is the number of results each endpoint should be able to provide
	MaximumNumberOfResults int `yaml:"maximum-number-of-results,omitempty"`

	// MaximumNumberOfEvents is the number of events each endpoint should be able to provide
	MaximumNumberOfEvents int `yaml:"maximum-number-of-events,omitempty"`

	// UptimeRetention is the minimum duration for which uptime data is kept.
	// Defaults to 30 days. Increase it (e.g. to 8760h for a full year) to support long-range
	// uptime badges such as the 365d badge; note that a higher value increases storage usage.
	UptimeRetention time.Duration `yaml:"uptime-retention,omitempty"`
}

// ValidateAndSetDefaults validates the configuration and sets the default values (if applicable)
func (c *Config) ValidateAndSetDefaults() error {
	if c.Type == "" {
		c.Type = TypeMemory
	}
	if (c.Type == TypePostgres || c.Type == TypeSQLite) && len(c.Path) == 0 {
		return ErrSQLStorageRequiresPath
	}
	if c.Type == TypeMemory && len(c.Path) > 0 {
		return ErrMemoryStorageDoesNotSupportPath
	}
	if c.MaximumNumberOfResults <= 0 {
		c.MaximumNumberOfResults = DefaultMaximumNumberOfResults
	}
	if c.MaximumNumberOfEvents <= 0 {
		c.MaximumNumberOfEvents = DefaultMaximumNumberOfEvents
	}
	if c.UptimeRetention <= 0 {
		c.UptimeRetention = DefaultUptimeRetention
	}
	return nil
}

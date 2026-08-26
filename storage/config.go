package storage

import (
	"errors"
	"time"
)

const (
	DefaultMaximumNumberOfResults = 100
	DefaultMaximumNumberOfEvents  = 50

	// DefaultFlushInterval is the flush-interval used when none is configured
	DefaultFlushInterval = 10 * time.Minute

	// MinimumFlushInterval is the lowest allowed flush-interval
	MinimumFlushInterval = time.Minute
)

var (
	ErrSQLStorageRequiresPath               = errors.New("sql storage requires a non-empty path to be defined")
	ErrMemoryStorageDoesNotSupportPath      = errors.New("memory storage does not support persistence, use sqlite if you want persistence on file")
	ErrBufferedStorageRequiresSQLite        = errors.New("buffered storage is only supported if the storage type is sqlite")
	ErrFlushIntervalRequiresBufferedStorage = errors.New("flush-interval is only supported if buffered is set to true")
	ErrFlushIntervalTooSmall                = errors.New("flush-interval must be at least 1m")
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

	// Buffered is whether to keep the live database in memory and only persist it to Path
	// on a timer (see FlushInterval) as well as on graceful shutdown. Spares flash storage
	// at the cost of losing up to FlushInterval of history on an unclean shutdown.
	// Only applies if Config.Type is TypeSQLite.
	Buffered bool `yaml:"buffered,omitempty"`

	// FlushInterval is how often the in-memory database is persisted to Path.
	// Defaults to DefaultFlushInterval, cannot be lower than MinimumFlushInterval.
	// Only applies if Buffered is true.
	FlushInterval time.Duration `yaml:"flush-interval,omitempty"`

	// MaximumNumberOfResults is the number of results each endpoint should be able to provide
	MaximumNumberOfResults int `yaml:"maximum-number-of-results,omitempty"`

	// MaximumNumberOfEvents is the number of events each endpoint should be able to provide
	MaximumNumberOfEvents int `yaml:"maximum-number-of-events,omitempty"`
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
	if c.Buffered && c.Type != TypeSQLite {
		return ErrBufferedStorageRequiresSQLite
	}
	if c.FlushInterval != 0 && !c.Buffered {
		return ErrFlushIntervalRequiresBufferedStorage
	}
	if c.Buffered {
		if c.FlushInterval == 0 {
			c.FlushInterval = DefaultFlushInterval
		} else if c.FlushInterval < MinimumFlushInterval {
			return ErrFlushIntervalTooSmall
		}
	}
	if c.MaximumNumberOfResults <= 0 {
		c.MaximumNumberOfResults = DefaultMaximumNumberOfResults
	}
	if c.MaximumNumberOfEvents <= 0 {
		c.MaximumNumberOfEvents = DefaultMaximumNumberOfEvents
	}
	return nil
}

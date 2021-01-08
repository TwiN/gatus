package storage

import "time"

// Config is the cache persistence configuration for Gatus
type Config struct {
	FilePath string         `yaml:"file-path"`
	Interval *time.Duration `yaml:"interval,omitempty"`
}

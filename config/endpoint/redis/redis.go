package redis

import (
	"errors"
)

var (
	// ErrEndpointWithoutSSHUsername is the error with which Gatus will panic if an endpoint with SSH monitoring is configured without a user.
	ErrEndpointWithoutRedisPassword = errors.New("you must specify a password for each REDIS endpoint")
)

type Config struct {
	Password string `yaml:"password,omitempty"`
	SSL      bool   `yaml:"ssl,omitempty"`
	DB       int    `yaml:"db,omitempty"`
}

// Validate the SSH configuration
func (cfg *Config) Validate() error {
	// If there's no password, return an error
	if len(cfg.Password) == 0 {
		return ErrEndpointWithoutRedisPassword
	}
	return nil
}

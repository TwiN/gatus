package ssh

import (
	"errors"
)

var (
	// ErrEndpointWithoutSSHUsername is the error with which Gatus will panic if an endpoint with SSH monitoring is configured without a user.
	ErrEndpointWithoutSSHUsername = errors.New("you must specify a username for each SSH endpoint")

	// ErrEndpointWithoutSSHPassword is the error with which Gatus will panic if an endpoint with SSH monitoring is configured without a password.
	ErrEndpointWithoutSSHPassword = errors.New("you must specify a password for each SSH endpoint")
)

type Config struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// Validate the SSH configuration
func (cfg *Config) Validate() error {
	// If there's no username and password, this endpoint can still check the SSH banner, so the endpoint is still valid
	if len(cfg.Username) == 0 && len(cfg.Password) == 0 {
		return nil
	}
	if len(cfg.Username) == 0 {
		return ErrEndpointWithoutSSHUsername
	}
	if len(cfg.Password) == 0 {
		return ErrEndpointWithoutSSHPassword
	}
	return nil
}

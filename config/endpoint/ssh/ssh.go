package ssh

import (
	"errors"
)

var (
	// ErrEndpointWithoutSSHUsername is the error with which Gatus will panic if an endpoint with SSH monitoring is configured without a user.
	ErrEndpointWithoutSSHUsername = errors.New("you must specify a username for each SSH endpoint")

	// ErrEndpointWithoutSSHAuth is the error with which Gatus will panic if an endpoint with SSH monitoring is configured without a password or private key.
	ErrEndpointWithoutSSHAuth = errors.New("you must specify a password or private-key for each SSH endpoint")
)

type Config struct {
	Username   string `yaml:"username,omitempty"`
	Password   string `yaml:"password,omitempty"`
	PrivateKey string `yaml:"private-key,omitempty"`
}

// Validate the SSH configuration
func (cfg *Config) Validate() error {
	// If there's no username, password, or private key, this endpoint can still check the SSH banner, so the endpoint is still valid
	if len(cfg.Username) == 0 && len(cfg.Password) == 0 && len(cfg.PrivateKey) == 0 {
		return nil
	}
	// If any authentication method is provided (password or private key), a username is required
	if len(cfg.Username) == 0 {
		return ErrEndpointWithoutSSHUsername
	}
	// If a username is provided, require at least a password or a private key
	if len(cfg.Password) == 0 && len(cfg.PrivateKey) == 0 {
		return ErrEndpointWithoutSSHAuth
	}
	return nil
}

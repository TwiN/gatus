package sftp

import (
	"errors"
)

var (
	// ErrEndpointWithoutSFTPUsername is the error with which Gatus will panic if an endpoint with SFTP monitoring is configured without a user.
	ErrEndpointWithoutSFTPUsername = errors.New("you must specify a username for each SFTP endpoint")

	// ErrEndpointWithoutSFTPAuth is the error with which Gatus will panic if an endpoint with SFTP monitoring is configured without a password or private key.
	ErrEndpointWithoutSFTPAuth = errors.New("you must specify a password or private-key for each SFTP endpoint")
)

type Config struct {
	Username   string `yaml:"username,omitempty"`
	Password   string `yaml:"password,omitempty"`
	PrivateKey string `yaml:"private-key,omitempty"`
	Path       string `yaml:"path,omitempty"`
}

// Validate the SFTP configuration
func (cfg *Config) Validate() error {
	// If there's no username, password, or private key, this endpoint can still check the SFTP connection, so the endpoint is still valid
	if len(cfg.Username) == 0 && len(cfg.Password) == 0 && len(cfg.PrivateKey) == 0 {
		return nil
	}
	// If any authentication method is provided (password or private key), a username is required
	if len(cfg.Username) == 0 {
		return ErrEndpointWithoutSFTPUsername
	}
	// If a username is provided, require at least a password or a private key
	if len(cfg.Password) == 0 && len(cfg.PrivateKey) == 0 {
		return ErrEndpointWithoutSFTPAuth
	}
	return nil
}

package core

import (
	"errors"
)

var (
	// ErrEndpointWithoutSSHUsername is the error with which Gatus will panic if an endpoint with SSH monitoring is configured without a user.
	ErrEndpointWithoutSSHUsername = errors.New("you must specify a username for each SSH endpoint")

	// ErrEndpointWithoutSSHPassword is the error with which Gatus will panic if an endpoint with SSH monitoring is configured without a password.
	ErrEndpointWithoutSSHPassword = errors.New("you must specify a password for each SSH endpoint")
)

type SSH struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// validateAndSetDefaults validates the endpoint
func (s *SSH) validate() error {
	if len(s.Username) == 0 {
		return ErrEndpointWithoutSSHUsername
	}
	if len(s.Password) == 0 {
		return ErrEndpointWithoutSSHPassword
	}
	return nil
}

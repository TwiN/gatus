package ssh

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"
)

var (
	// ErrEndpointWithoutSSHUsername is the error with which Gatus will panic if an endpoint with SSH monitoring is configured without a user.
	ErrEndpointWithoutSSHUsername = errors.New("you must specify a username for each SSH endpoint")

	// ErrEndpointWithoutSSHAuth is the error with which Gatus will panic if an endpoint with SSH monitoring is configured without a password or private key.
	ErrEndpointWithoutSSHAuth = errors.New("you must specify a password or private-key for each SSH endpoint")

	// ErrEndpointWithHostKeyWithoutSSHAuth is the error with which Gatus will panic if an endpoint with SSH monitoring is configured
	// with a host key but without credentials, because host keys are only verified when authenticating.
	ErrEndpointWithHostKeyWithoutSSHAuth = errors.New("ssh.host-key requires a username and a password or private-key, because the host key cannot be verified when only the SSH banner is checked")
)

type Config struct {
	Username   string `yaml:"username,omitempty"`
	Password   string `yaml:"password,omitempty"`
	PrivateKey string `yaml:"private-key,omitempty"`

	// HostKey is the expected public key of the server, in the authorized_keys format (e.g. "ssh-ed25519 AAAA...").
	// If specified, the connection fails unless the server presents this key. If not, the host key is not verified.
	HostKey string `yaml:"host-key,omitempty"`
}

// Validate the SSH configuration
func (cfg *Config) Validate() error {
	// If there's no username, password, or private key, this endpoint can still check the SSH banner, so the endpoint is still valid
	if len(cfg.Username) == 0 && len(cfg.Password) == 0 && len(cfg.PrivateKey) == 0 {
		if len(cfg.HostKey) != 0 {
			return ErrEndpointWithHostKeyWithoutSSHAuth
		}
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
	if len(cfg.HostKey) != 0 {
		if _, _, _, _, err := ssh.ParseAuthorizedKey([]byte(cfg.HostKey)); err != nil {
			return fmt.Errorf("invalid ssh.host-key: %w", err)
		}
	}
	return nil
}

package security

import (
	"github.com/gorilla/mux"
)

// Config is the security configuration for Gatus
type Config struct {
	Basic *BasicConfig `yaml:"basic,omitempty"`
	OIDC  *OIDCConfig  `yaml:"oidc,omitempty"`
}

// IsValid returns whether the security configuration is valid or not
func (c *Config) IsValid() bool {
	return (c.Basic != nil && c.Basic.isValid()) || (c.OIDC != nil && c.OIDC.isValid())
}

// RegisterHandlers registers all handlers required based on the security configuration
func (c *Config) RegisterHandlers(router *mux.Router) error {
	if c.OIDC != nil {
		if err := c.OIDC.initialize(); err != nil {
			return err
		}
		router.HandleFunc("/login", c.OIDC.loginHandler)
		router.HandleFunc("/authorization-code/callback", c.OIDC.callbackHandler)
	}
	return nil
}

// BasicConfig is the configuration for Basic authentication
type BasicConfig struct {
	// Username is the name which will need to be used for a successful authentication
	Username string `yaml:"username"`

	// PasswordSha512Hash is the SHA512 hash of the password which will need to be used for a successful authentication
	PasswordSha512Hash string `yaml:"password-sha512"`
}

// isValid returns whether the basic security configuration is valid or not
func (c *BasicConfig) isValid() bool {
	return len(c.Username) > 0 && len(c.PasswordSha512Hash) == 128
}

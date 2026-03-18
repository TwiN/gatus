package security

import (
	"errors"
	"slices"
)

var (
	ErrAPITokensEmpty = errors.New("security.api.tokens must not contain empty tokens")
)

// APIConfig is the configuration for API token authentication
type APIConfig struct {
	// Tokens is a list of valid API tokens for authentication
	Tokens []string `yaml:"tokens"`
}

// Validate validates the APIConfig
func (c *APIConfig) Validate() error {
	if c == nil {
		return nil
	}
	// Ensure no empty tokens are configured
	for _, token := range c.Tokens {
		if len(token) == 0 {
			return ErrAPITokensEmpty
		}
	}
	return nil
}

// IsValid checks if a given token is valid
func (c *APIConfig) IsValid(token string) bool {
	if c == nil || len(c.Tokens) == 0 || len(token) == 0 {
		return false
	}
	return slices.Contains(c.Tokens, token)
}

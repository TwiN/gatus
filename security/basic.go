package security

// BasicConfig is the configuration for Basic authentication
type BasicConfig struct {
	// Username is the name which will need to be used for a successful authentication
	Username string `yaml:"username"`

	// PasswordBcryptHashBase64Encoded is the base64 encoded string of the Bcrypt hash of the password to use to
	// authenticate using basic auth.
	PasswordBcryptHashBase64Encoded string `yaml:"password-bcrypt-base64"`

	// APITokens is a list of bearer tokens that can be used to access the API instead of basic auth
	APITokens []string `yaml:"api-tokens"`
}

// isValid returns whether the basic security configuration is valid or not
func (c *BasicConfig) isValid() bool {
	return len(c.Username) > 0 && len(c.PasswordBcryptHashBase64Encoded) > 0 && validateTokens(c.APITokens)
}

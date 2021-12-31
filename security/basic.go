package security

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

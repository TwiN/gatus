package security

// Config is the security configuration for Gatus
type Config struct {
	Basic *BasicConfig `yaml:"basic"`
}

// IsValid returns whether the security configuration is valid or not
func (c *Config) IsValid() bool {
	return c.Basic != nil && c.Basic.IsValid()
}

// BasicConfig is the configuration for Basic authentication
type BasicConfig struct {
	// Username is the name which will need to be used for a successful authentication
	Username string `yaml:"username"`

	// PasswordSha512Hash is the SHA512 hash of the password which will need to be used for a successful authentication
	PasswordSha512Hash string `yaml:"password-sha512"`

	// Clear Text Password
	Password string `yaml:"password"`
}

// IsValid returns whether the basic security configuration is valid or not
func (c *BasicConfig) IsValid() bool {
	return len(c.Username) > 0 && len(c.PasswordSha512Hash) == 128
}

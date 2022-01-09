package security

import "log"

// BasicConfig is the configuration for Basic authentication
type BasicConfig struct {
	// Username is the name which will need to be used for a successful authentication
	Username string `yaml:"username"`

	// PasswordSha512Hash is the SHA512 hash of the password which will need to be used for a successful authentication
	// XXX: Remove this on v4.0.0
	// Deprecated: Use PasswordBcryptHashBase64Encoded instead
	PasswordSha512Hash string `yaml:"password-sha512"`

	// PasswordBcryptHashBase64Encoded is the base64 encoded string of the Bcrypt hash of the password to use to
	// authenticate using basic auth.
	PasswordBcryptHashBase64Encoded string `yaml:"password-bcrypt-base64"`
}

// isValid returns whether the basic security configuration is valid or not
func (c *BasicConfig) isValid() bool {
	if len(c.PasswordSha512Hash) > 0 {
		log.Println("WARNING: security.basic.password-sha512 has been deprecated in favor of security.basic.password-bcrypt-base64")
	}
	return len(c.Username) > 0 && (len(c.PasswordSha512Hash) == 128 || len(c.PasswordBcryptHashBase64Encoded) > 0)
}

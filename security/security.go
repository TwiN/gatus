package security

type Config struct {
	Basic *BasicConfig `yaml:"basic"`
}

func (c *Config) IsValid() bool {
	return c.Basic != nil && c.Basic.IsValid()
}

type BasicConfig struct {
	Username           string `yaml:"username"`
	PasswordSha512Hash string `yaml:"password-sha512"`
}

func (c *BasicConfig) IsValid() bool {
	return len(c.Username) > 0 && len(c.PasswordSha512Hash) == 128
}

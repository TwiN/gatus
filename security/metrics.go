package security

// MetricsConfig is the configuration for protecting the metrics endpoint
type MetricsConfig struct {
	// Protected is whether the metrics endpoint requires Basic or OIDC authentication. Implied if Tokens is set
	Protected bool `yaml:"protected"`

	// Tokens is a list of bearer tokens that can be used to access the metrics endpoint
	Tokens []string `yaml:"tokens"`
}

// isProtected returns whether the metrics endpoint requires authentication
func (c *MetricsConfig) isProtected() bool {
	return c.Protected || len(c.Tokens) > 0
}

// isValid returns whether the metrics security configuration is valid or not
func (c *MetricsConfig) isValid(hasProvider bool) bool {
	return validateTokens(c.Tokens) && (!c.Protected || hasProvider || len(c.Tokens) > 0)
}

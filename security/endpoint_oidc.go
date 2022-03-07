package security

// EndpointOIDCConfig is the configuration for endpoint OIDC authentication
type EndpointOIDCConfig struct {
	IssuerURL       string   `yaml:"issuer-url"`   // e.g. https://dev-12345678.okta.com
	ClientID        string   `yaml:"client-id"`
	ClientSecret    string   `yaml:"client-secret"`
	Scopes          []string `yaml:"scopes"`           // e.g. ["openid"]
}

func (c *EndpointOIDCConfig) IsValid() bool {
	return len(c.IssuerURL) > 0 && len(c.ClientID) > 0 && len(c.ClientSecret) > 0 && len(c.Scopes) > 0
}

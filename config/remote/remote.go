package remote

import (
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/logr"
)

// NOTICE: This is an experimental alpha feature and may be updated/removed in future versions.
// For more information, see https://github.com/TwiN/gatus/issues/64

type Config struct {
	// Instances is a list of remote instances to retrieve endpoint statuses from.
	Instances []Instance `yaml:"instances,omitempty"`

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client,omitempty"`
}

type Instance struct {
	EndpointPrefix string `yaml:"endpoint-prefix"`
	URL            string `yaml:"url"`
}

func (c *Config) ValidateAndSetDefaults() error {
	if c.ClientConfig == nil {
		c.ClientConfig = client.GetDefaultConfig()
	} else {
		if err := c.ClientConfig.ValidateAndSetDefaults(); err != nil {
			return err
		}
	}
	if len(c.Instances) > 0 {
		logr.Warn("WARNING: Your configuration is using 'remote', which is in alpha and may be updated/removed in future versions.")
		logr.Warn("WARNING: See https://github.com/TwiN/gatus/issues/64 for more information")
	}
	return nil
}

package remote

import (
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/logr"
)

// NOTICE: This is an experimental alpha feature and may be updated/removed in future versions.
// For more information, see https://github.com/TwiN/gatus/issues/64

type Config struct {
	// Instances is a list of remote instances to retrieve endpoint statuses from.
	Instances []*Instance `yaml:"instances,omitempty"`

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client,omitempty"`
}

type Instance struct {
	EndpointPrefix string `yaml:"endpoint-prefix"`
	URL            string `yaml:"url"`
	IncludeRemote  *bool  `yaml:"include-remote,omitempty"`
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

	foundDefaultIncludeRemote := false
	defaultIncludeRemote := true
	for _, instance := range c.Instances {
		if instance.IncludeRemote == nil {
			foundDefaultIncludeRemote = true
			instance.IncludeRemote = &defaultIncludeRemote
		}
	}

	if foundDefaultIncludeRemote {
		logr.Warn("WARNING: There is a new remote instance setting `include-remote` which may be used (when `false`) to avoid duplicate entries.")
		logr.Warn("WARNING: The default value is `true` for backwards-compatibility, but please set a value explicitly to avoid this message.")
		logr.Warn("WARNING: See https://github.com/TwiN/gatus/pull/1721 for more information")
	}
	return nil
}

package k8s

import "github.com/TwinProduction/gatus/core"

// Config for Kubernetes auto-discovery
type Config struct {
	// AutoDiscover to discover services to monitor
	AutoDiscover bool `yaml:"auto-discover"`

	// ServiceTemplate Template for auto discovered services
	ServiceTemplate *core.Service `yaml:"service-template"`

	// ExcludeSuffix is a slice of service suffixes that should be ignored
	ExcludeSuffix []string `yaml:"exclude-suffix"`

	// ClusterMode to authenticate with kubernetes
	ClusterMode ClusterMode `yaml:"cluster-mode"`

	// Namespaces from which to discover services
	Namespaces []*NamespaceConfig `yaml:"namespaces"`
}

// NamespaceConfig level config
type NamespaceConfig struct {
	// Name of namespace
	Name string `yaml:"name"`

	// ServiceSuffix to append to service name
	ServiceSuffix string `yaml:"service-suffix"`

	// HealthAPI URI to append to service to reach health check API
	HealthAPI string `yaml:"health-api"`
}

// ClusterMode is the mode to use to authenticate to Kubernetes
type ClusterMode string

const (
	ClusterModeIn  ClusterMode = "in"
	ClusterModeOut ClusterMode = "out"
)

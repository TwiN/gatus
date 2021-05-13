package k8s

import "github.com/Meldiron/gatus/core"

// Config for Kubernetes auto-discovery
type Config struct {
	// AutoDiscover to discover services to monitor
	AutoDiscover bool `yaml:"auto-discover"`

	// ClusterMode is the mode to use to authenticate with Kubernetes
	ClusterMode ClusterMode `yaml:"cluster-mode"`

	// ServiceTemplate is the template for auto discovered services
	ServiceTemplate *core.Service `yaml:"service-template"`

	// ExcludedServiceSuffixes is a list of service suffixes that should be ignored
	ExcludedServiceSuffixes []string `yaml:"excluded-service-suffixes"`

	// Namespaces is a list of configurations for the namespaces from which services will be discovered
	Namespaces []*NamespaceConfig `yaml:"namespaces"`
}

// NamespaceConfig level config
type NamespaceConfig struct {
	// Name of the namespace
	Name string `yaml:"name"`

	// ExcludedServices is a list of services to exclude from the auto discovery
	ExcludedServices []string `yaml:"excluded-services"`

	// HostnameSuffix is a suffix to append to each service name before calling TargetPath
	HostnameSuffix string `yaml:"hostname-suffix"`

	// TargetPath Path to append after the HostnameSuffix
	TargetPath string `yaml:"target-path"`
}

// ClusterMode is the mode to use to authenticate to Kubernetes
type ClusterMode string

const (
	ClusterModeIn   ClusterMode = "in"
	ClusterModeOut  ClusterMode = "out"
	ClusterModeMock ClusterMode = "mock"
)

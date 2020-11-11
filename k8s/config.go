package k8s

import "github.com/TwinProduction/gatus/core"

//Config for Kubernetes auto-discovery
type Config struct {
	//AutoDiscover to discover services to monitor
	AutoDiscover bool `yaml:"auto-discover"`

	//ServiceTemplate Template for auto disocovered services
	ServiceTemplate core.Service `yaml:"service-template"`

	//ExcludeSuffix Ignore services with this suffix
	ExcludeSuffix []string `yaml:"exclude-suffix"`

	//ClusterMode to authenticate with kubernetes
	ClusterMode string `yaml:"cluster-mode"`

	//Namespaces from which to discover services
	Namespaces []Namespace `yaml:"namespaces"`
}

//Namespace level config
type Namespace struct {
	//Name of namespace
	Name string `yaml:"name"`
	//ServiceSuffix to append to service name
	ServiceSuffix string `yaml:"service-suffix"`
	//HealthAPI URI to append to service to reach health check API
	HealthAPI string `yaml:"health-api"`
}

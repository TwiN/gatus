package k8s

import (
	"fmt"
	"strings"

	"github.com/TwinProduction/gatus/core"
)

// DiscoverServices return discovered services
func DiscoverServices(kubernetesConfig *Config) ([]*core.Service, error) {
	client, err := NewClient(kubernetesConfig.ClusterMode)
	if err != nil {
		return nil, err
	}
	services := make([]*core.Service, 0)
	for _, ns := range kubernetesConfig.Namespaces {
		kubernetesServices, err := GetKubernetesServices(client, ns.Name)
		if err != nil {
			return nil, err
		}
	skipExcluded:
		for _, service := range kubernetesServices {
			for _, excludedServiceSuffix := range kubernetesConfig.ExcludedServiceSuffixes {
				if strings.HasSuffix(service.Name, excludedServiceSuffix) {
					continue skipExcluded
				}
			}
			for _, excludedService := range ns.ExcludedServices {
				if service.Name == excludedService {
					continue skipExcluded
				}
			}
			var url string
			if strings.HasPrefix(ns.TargetPath, "/") {
				url = fmt.Sprintf("http://%s%s%s", service.Name, ns.HostnameSuffix, ns.TargetPath)
			} else {
				url = fmt.Sprintf("http://%s%s/%s", service.Name, ns.HostnameSuffix, ns.TargetPath)
			}
			services = append(services, &core.Service{
				Name:       service.Name,
				URL:        url,
				Interval:   kubernetesConfig.ServiceTemplate.Interval,
				Conditions: kubernetesConfig.ServiceTemplate.Conditions,
			})
		}
	}
	return services, nil
}

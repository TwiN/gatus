package k8s

import (
	"fmt"
	"strings"

	"github.com/TwinProduction/gatus/core"
)

// DiscoverServices return discovered services
func DiscoverServices(kubernetesConfig *Config) ([]*core.Service, error) {
	client := NewClient(kubernetesConfig.ClusterMode)
	services := make([]*core.Service, 0)
	for _, ns := range kubernetesConfig.Namespaces {
		kubernetesServices, err := GetKubernetesServices(client, ns.Name)
		if err != nil {
			return nil, err
		}
		for _, s := range kubernetesServices {
			if isExcluded(kubernetesConfig.ExcludeSuffix, s.Name) {
				continue
			}
			services = append(services, &core.Service{
				Name:       s.Name,
				URL:        fmt.Sprintf("http://%s%s/%s", s.Name, ns.ServiceSuffix, ns.HealthAPI),
				Interval:   kubernetesConfig.ServiceTemplate.Interval,
				Conditions: kubernetesConfig.ServiceTemplate.Conditions,
			})
		}
	}
	return services, nil
}

// TODO: don't uselessly allocate new things here, just move this inside the DiscoverServices function
func isExcluded(excludeList []string, name string) bool {
	for _, x := range excludeList {
		if strings.HasSuffix(name, x) {
			return true
		}
	}
	return false
}

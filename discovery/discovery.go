package discovery

import (
	"fmt"
	"strings"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/k8s"
)

//GetServices return discovered service
func GetServices(kubernetesConfig *k8s.Config) []*core.Service {
	client := k8s.NewClient(kubernetesConfig.ClusterMode)
	svcs := make([]*core.Service, 0)

	for _, ns := range kubernetesConfig.Namespaces {
		services := k8s.GetServices(client, ns.Name)
		for _, s := range services {
			if exclude(kubernetesConfig.ExcludeSuffix, s.Name) {
				continue
			}
			svc := core.Service{
				Name:       s.Name,
				URL:        fmt.Sprintf("http://%s%s/%s", s.Name, ns.ServiceSuffix, ns.HealthAPI),
				Interval:   kubernetesConfig.ServiceTemplate.Interval,
				Conditions: kubernetesConfig.ServiceTemplate.Conditions,
			}
			svcs = append(svcs, &svc)
		}
	}
	return svcs
}

func exclude(excludeList []string, name string) bool {
	for _, x := range excludeList {
		if strings.HasSuffix(name, x) {
			return true
		}
	}
	return false
}

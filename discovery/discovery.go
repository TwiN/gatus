package discovery

import (
	"fmt"
	"strings"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/k8s"
)

//GetServices return discovered service
func GetServices(cfg *config.Config) []*core.Service {
	client := k8s.NewClient(cfg.Kubernetes.ClusterMode)
	svcs := make([]*core.Service, 0)

	for _, ns := range cfg.Kubernetes.Namespaces {
		services := k8s.GetServices(client, ns.Name)

		for _, s := range services {
			if exclude(cfg.Kubernetes.ExcludeSuffix, s.Name) {
				continue
			}
			svc := core.Service{Name: s.Name,
				URL:        fmt.Sprintf("http://%s%s/%s", s.Name, ns.ServiceSuffix, ns.HealthAPI),
				Interval:   cfg.Kubernetes.ServiceTemplate.Interval,
				Conditions: cfg.Kubernetes.ServiceTemplate.Conditions,
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

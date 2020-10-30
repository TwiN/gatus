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
	client := k8s.NewClient(cfg.K8sClusterMode)
	services := k8s.GetServices(client, "services")
	svcs := make([]*core.Service, 0)
	for _, s := range services {
		if exclude(cfg.ExcludeSuffix, s.Name) {
			continue
		}
		svc := core.Service{Name: s.Name, URL: fmt.Sprintf("http://%s%s/health", s.Name, cfg.K8SServiceSuffix), Interval: cfg.K8SServiceConfig.Interval, Conditions: cfg.K8SServiceConfig.Conditions}
		svcs = append(svcs, &svc)
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

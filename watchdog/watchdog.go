package watchdog

import (
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/metric"
	"log"
	"sync"
	"time"
)

var (
	serviceResults = make(map[string][]*core.Result)
	rwLock         sync.RWMutex
)

func GetServiceResults() *map[string][]*core.Result {
	return &serviceResults
}

func Monitor() {
	for _, service := range config.Get().Services {
		go func(service *core.Service) {
			for {
				log.Printf("[watchdog][Monitor] Monitoring serviceName=%s", service.Name)
				result := service.EvaluateConditions()
				metric.PublishMetricsForService(service, result)
				rwLock.Lock()
				serviceResults[service.Name] = append(serviceResults[service.Name], result)
				if len(serviceResults[service.Name]) > 10 {
					serviceResults[service.Name] = serviceResults[service.Name][1:]
				}
				rwLock.Unlock()
				log.Printf(
					"[watchdog][Monitor] Finished monitoring serviceName=%s; errors=%d; requestDuration=%s",
					service.Name,
					len(result.Errors),
					result.Duration.Round(time.Millisecond),
				)
				log.Printf("[watchdog][Monitor] Waiting interval=%s before monitoring serviceName=%s", service.Interval, service.Name)
				time.Sleep(service.Interval)
			}
		}(service)
	}
}

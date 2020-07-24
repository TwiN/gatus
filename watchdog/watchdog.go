package watchdog

import (
	"fmt"
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

func Monitor(cfg *config.Config) {
	for _, service := range cfg.Services {
		go monitor(service)
		// To prevent multiple requests from running at the same time
		time.Sleep(1111 * time.Millisecond)
	}
}

func monitor(service *core.Service) {
	for {
		// By placing the lock here, we prevent multiple services from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		rwLock.Lock()
		log.Printf("[watchdog][Monitor] Monitoring serviceName=%s", service.Name)
		result := service.EvaluateConditions()
		metric.PublishMetricsForService(service, result)
		serviceResults[service.Name] = append(serviceResults[service.Name], result)
		if len(serviceResults[service.Name]) > 20 {
			serviceResults[service.Name] = serviceResults[service.Name][1:]
		}
		rwLock.Unlock()
		var extra string
		if !result.Success {
			extra = fmt.Sprintf("responseBody=%s", result.Body)
		}
		log.Printf(
			"[watchdog][Monitor] Finished monitoring serviceName=%s; errors=%d; requestDuration=%s; %s",
			service.Name,
			len(result.Errors),
			result.Duration.Round(time.Millisecond),
			extra,
		)
		log.Printf("[watchdog][Monitor] Waiting interval=%s before monitoring serviceName=%s", service.Interval, service.Name)
		time.Sleep(service.Interval)
	}
}

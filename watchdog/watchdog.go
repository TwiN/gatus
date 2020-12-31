package watchdog

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/metric"
	"github.com/TwinProduction/gatus/storage"
)

var (
	store = storage.NewInMemoryStore()

	// monitoringMutex is used to prevent multiple services from being evaluated at the same time.
	// Without this, conditions using response time may become inaccurate.
	monitoringMutex sync.Mutex
)

// GetJSONEncodedServiceStatuses returns a list of core.ServiceStatus for each services encoded using json.Marshal.
// The reason why the encoding is done here is because we use a mutex to prevent concurrent map access.
func GetJSONEncodedServiceStatuses() ([]byte, error) {
	serviceStatuses := store.GetAll()
	data, err := json.Marshal(serviceStatuses)
	return data, err
}

// GetUptimeByServiceGroupAndName returns the uptime of a service based on its group and name
func GetUptimeByServiceGroupAndName(group, name string) *core.Uptime {
	key := fmt.Sprintf("%s_%s", group, name)
	serviceStatuses := store.GetAll()
	serviceStatus, exists := serviceStatuses[key]
	if !exists {
		return nil
	}
	return serviceStatus.Uptime
}

// Monitor loops over each services and starts a goroutine to monitor each services separately
func Monitor(cfg *config.Config) {
	for _, service := range cfg.Services {
		go monitor(service)
		// To prevent multiple requests from running at the same time
		time.Sleep(1111 * time.Millisecond)
	}
}

// monitor monitors a single service in a loop
func monitor(service *core.Service) {
	cfg := config.Get()
	for {
		if !cfg.DisableMonitoringLock {
			// By placing the lock here, we prevent multiple services from being monitored at the exact same time, which
			// could cause performance issues and return inaccurate results
			monitoringMutex.Lock()
		}
		if cfg.Debug {
			log.Printf("[watchdog][monitor] Monitoring serviceName=%s", service.Name)
		}
		result := service.EvaluateHealth()
		metric.PublishMetricsForService(service, result)
		UpdateServiceStatuses(service, result)
		var extra string
		if !result.Success {
			extra = fmt.Sprintf("responseBody=%s", result.Body)
		}
		log.Printf(
			"[watchdog][monitor] Monitored serviceName=%s; success=%v; errors=%d; requestDuration=%s; %s",
			service.Name,
			result.Success,
			len(result.Errors),
			result.Duration.Round(time.Millisecond),
			extra,
		)
		HandleAlerting(service, result)
		if cfg.Debug {
			log.Printf("[watchdog][monitor] Waiting for interval=%s before monitoring serviceName=%s again", service.Interval, service.Name)
		}
		if !cfg.DisableMonitoringLock {
			monitoringMutex.Unlock()
		}
		time.Sleep(service.Interval)
	}
}

// UpdateServiceStatuses updates the slice of service statuses
func UpdateServiceStatuses(service *core.Service, result *core.Result) {
	store.Insert(service, result)
}

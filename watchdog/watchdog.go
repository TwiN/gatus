package watchdog

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/TwinProduction/gatus/v3/alerting"
	"github.com/TwinProduction/gatus/v3/config"
	"github.com/TwinProduction/gatus/v3/config/maintenance"
	"github.com/TwinProduction/gatus/v3/core"
	"github.com/TwinProduction/gatus/v3/metric"
	"github.com/TwinProduction/gatus/v3/storage"
)

var (
	// monitoringMutex is used to prevent multiple services from being evaluated at the same time.
	// Without this, conditions using response time may become inaccurate.
	monitoringMutex sync.Mutex

	ctx        context.Context
	cancelFunc context.CancelFunc
)

// Monitor loops over each services and starts a goroutine to monitor each services separately
func Monitor(cfg *config.Config) {
	ctx, cancelFunc = context.WithCancel(context.Background())
	for _, service := range cfg.Services {
		if service.IsEnabled() {
			// To prevent multiple requests from running at the same time, we'll wait for a little before each iteration
			time.Sleep(1111 * time.Millisecond)
			go monitor(service, cfg.Alerting, cfg.Maintenance, cfg.DisableMonitoringLock, cfg.Metrics, cfg.Debug, ctx)
		}
	}
}

// monitor monitors a single service in a loop
func monitor(service *core.Service, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, disableMonitoringLock, enabledMetrics, debug bool, ctx context.Context) {
	// Run it immediately on start
	execute(service, alertingConfig, maintenanceConfig, disableMonitoringLock, enabledMetrics, debug)
	// Loop for the next executions
	for {
		select {
		case <-ctx.Done():
			log.Printf("[watchdog][monitor] Canceling current execution of group=%s; service=%s", service.Group, service.Name)
			return
		case <-time.After(service.Interval):
			execute(service, alertingConfig, maintenanceConfig, disableMonitoringLock, enabledMetrics, debug)
		}
	}
}

func execute(service *core.Service, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, disableMonitoringLock, enabledMetrics, debug bool) {
	if !disableMonitoringLock {
		// By placing the lock here, we prevent multiple services from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		monitoringMutex.Lock()
	}
	if debug {
		log.Printf("[watchdog][execute] Monitoring group=%s; service=%s", service.Group, service.Name)
	}
	result := service.EvaluateHealth()
	if enabledMetrics {
		metric.PublishMetricsForService(service, result)
	}
	UpdateServiceStatuses(service, result)
	log.Printf(
		"[watchdog][execute] Monitored group=%s; service=%s; success=%v; errors=%d; duration=%s",
		service.Group,
		service.Name,
		result.Success,
		len(result.Errors),
		result.Duration.Round(time.Millisecond),
	)
	if !maintenanceConfig.IsUnderMaintenance() {
		HandleAlerting(service, result, alertingConfig, debug)
	} else if debug {
		log.Println("[watchdog][execute] Not handling alerting because currently in the maintenance window")
	}
	if debug {
		log.Printf("[watchdog][execute] Waiting for interval=%s before monitoring group=%s service=%s again", service.Interval, service.Group, service.Name)
	}
	if !disableMonitoringLock {
		monitoringMutex.Unlock()
	}
}

// UpdateServiceStatuses updates the slice of service statuses
func UpdateServiceStatuses(service *core.Service, result *core.Result) {
	if err := storage.Get().Insert(service, result); err != nil {
		log.Println("[watchdog][UpdateServiceStatuses] Failed to insert data in storage:", err.Error())
	}
}

// Shutdown stops monitoring all services
func Shutdown() {
	cancelFunc()
}

package watchdog

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/TwiN/gatus/v3/alerting"
	"github.com/TwiN/gatus/v3/config"
	"github.com/TwiN/gatus/v3/config/maintenance"
	"github.com/TwiN/gatus/v3/core"
	"github.com/TwiN/gatus/v3/metric"
	"github.com/TwiN/gatus/v3/storage/store"
)

var (
	// monitoringMutex is used to prevent multiple endpoint from being evaluated at the same time.
	// Without this, conditions using response time may become inaccurate.
	monitoringMutex sync.Mutex

	ctx        context.Context
	cancelFunc context.CancelFunc
)

// Monitor loops over each endpoint and starts a goroutine to monitor each endpoint separately
func Monitor(cfg *config.Config) {
	ctx, cancelFunc = context.WithCancel(context.Background())
	for _, endpoint := range cfg.Endpoints {
		if endpoint.IsEnabled() {
			// To prevent multiple requests from running at the same time, we'll wait for a little before each iteration
			time.Sleep(777 * time.Millisecond)
			go monitor(endpoint, cfg.Alerting, cfg.Maintenance, cfg.DisableMonitoringLock, cfg.Metrics, cfg.Debug, ctx)
		}
	}
}

// monitor a single endpoint in a loop
func monitor(endpoint *core.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, disableMonitoringLock, enabledMetrics, debug bool, ctx context.Context) {
	// Run it immediately on start
	execute(endpoint, alertingConfig, maintenanceConfig, disableMonitoringLock, enabledMetrics, debug)
	// Loop for the next executions
	for {
		select {
		case <-ctx.Done():
			log.Printf("[watchdog][monitor] Canceling current execution of group=%s; endpoint=%s", endpoint.Group, endpoint.Name)
			return
		case <-time.After(endpoint.Interval):
			execute(endpoint, alertingConfig, maintenanceConfig, disableMonitoringLock, enabledMetrics, debug)
		}
	}
}

func execute(endpoint *core.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, disableMonitoringLock, enabledMetrics, debug bool) {
	if !disableMonitoringLock {
		// By placing the lock here, we prevent multiple endpoints from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		monitoringMutex.Lock()
	}
	if debug {
		log.Printf("[watchdog][execute] Monitoring group=%s; endpoint=%s", endpoint.Group, endpoint.Name)
	}
	result := endpoint.EvaluateHealth()
	if enabledMetrics {
		metric.PublishMetricsForEndpoint(endpoint, result)
	}
	UpdateEndpointStatuses(endpoint, result)
	log.Printf(
		"[watchdog][execute] Monitored group=%s; endpoint=%s; success=%v; errors=%d; duration=%s",
		endpoint.Group,
		endpoint.Name,
		result.Success,
		len(result.Errors),
		result.Duration.Round(time.Millisecond),
	)
	if !maintenanceConfig.IsUnderMaintenance() {
		HandleAlerting(endpoint, result, alertingConfig, debug)
	} else if debug {
		log.Println("[watchdog][execute] Not handling alerting because currently in the maintenance window")
	}
	if debug {
		log.Printf("[watchdog][execute] Waiting for interval=%s before monitoring group=%s endpoint=%s again", endpoint.Interval, endpoint.Group, endpoint.Name)
	}
	if !disableMonitoringLock {
		monitoringMutex.Unlock()
	}
}

// UpdateEndpointStatuses updates the slice of endpoint statuses
func UpdateEndpointStatuses(endpoint *core.Endpoint, result *core.Result) {
	if err := store.Get().Insert(endpoint, result); err != nil {
		log.Println("[watchdog][UpdateEndpointStatuses] Failed to insert data in storage:", err.Error())
	}
}

// Shutdown stops monitoring all endpoints
func Shutdown() {
	cancelFunc()
}

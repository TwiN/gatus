package watchdog

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/connectivity"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
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
			go monitor(endpoint, cfg.Alerting, cfg.Maintenance, cfg.Connectivity, cfg.DisableMonitoringLock, cfg.Metrics, cfg.Debug, ctx)
		}
	}
}

// monitor a single endpoint in a loop
func monitor(endpoint *core.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock, enabledMetrics, debug bool, ctx context.Context) {
	// Run it immediately on start
	execute(endpoint, alertingConfig, maintenanceConfig, connectivityConfig, disableMonitoringLock, enabledMetrics, debug)
	// Loop for the next executions
	for {
		select {
		case <-ctx.Done():
			log.Printf("[watchdog][monitor] Canceling current execution of group=%s; endpoint=%s", endpoint.Group, endpoint.Name)
			return
		case <-time.After(endpoint.Interval):
			execute(endpoint, alertingConfig, maintenanceConfig, connectivityConfig, disableMonitoringLock, enabledMetrics, debug)
		}
	}
}

func execute(endpoint *core.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock, enabledMetrics, debug bool) {
	if !disableMonitoringLock {
		// By placing the lock here, we prevent multiple endpoints from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		monitoringMutex.Lock()
		defer monitoringMutex.Unlock()
	}
	// If there's a connectivity checker configured, check if Gatus has internet connectivity
	if connectivityConfig != nil && connectivityConfig.Checker != nil && !connectivityConfig.Checker.IsConnected() {
		log.Println("[watchdog][execute] No connectivity; skipping execution")
		return
	}
	if debug {
		log.Printf("[watchdog][execute] Monitoring group=%s; endpoint=%s", endpoint.Group, endpoint.Name)
	}
	result := endpoint.EvaluateHealth()
	if enabledMetrics {
		metrics.PublishMetricsForEndpoint(endpoint, result)
	}
	UpdateEndpointStatuses(endpoint, result)
	if debug && !result.Success {
		log.Printf("[watchdog][execute] Monitored group=%s; endpoint=%s; success=%v; errors=%d; duration=%s; body=%s", endpoint.Group, endpoint.Name, result.Success, len(result.Errors), result.Duration.Round(time.Millisecond), result.Body)
	} else {
		log.Printf("[watchdog][execute] Monitored group=%s; endpoint=%s; success=%v; errors=%d; duration=%s", endpoint.Group, endpoint.Name, result.Success, len(result.Errors), result.Duration.Round(time.Millisecond))
	}
	if !maintenanceConfig.IsUnderMaintenance() {
		// TODO: Consider moving this after the monitoring lock is unlocked? I mean, how much noise can a single alerting provider cause...
		HandleAlerting(endpoint, result, alertingConfig, debug)
	} else if debug {
		log.Println("[watchdog][execute] Not handling alerting because currently in the maintenance window")
	}
	if debug {
		log.Printf("[watchdog][execute] Waiting for interval=%s before monitoring group=%s endpoint=%s again", endpoint.Interval, endpoint.Group, endpoint.Name)
	}
}

// UpdateEndpointStatuses updates the slice of endpoint statuses
func UpdateEndpointStatuses(endpoint *core.Endpoint, result *core.Result) {
	if err := store.Get().Insert(endpoint, result); err != nil {
		log.Println("[watchdog][UpdateEndpointStatuses] Failed to insert data in storage:", err.Error())
	}
}

// Shutdown stops monitoring all endpoints
func Shutdown(cfg *config.Config) {
	// Disable all the old HTTP connections
	for _, endpoint := range cfg.Endpoints {
		endpoint.Close()
	}
	cancelFunc()
}

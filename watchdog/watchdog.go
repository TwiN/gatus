package watchdog

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/connectivity"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/result"
	"github.com/TwiN/gatus/v5/config/maintenance"
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
func monitor(ep *endpoint.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock, enabledMetrics, debug bool, ctx context.Context) {
	// Run it immediately on start
	execute(ep, alertingConfig, maintenanceConfig, connectivityConfig, disableMonitoringLock, enabledMetrics, debug)
	// Loop for the next executions
	for {
		select {
		case <-ctx.Done():
			log.Printf("[watchdog.monitor] Canceling current execution of group=%s; endpoint=%s", ep.Group, ep.Name)
			return
		case <-time.After(ep.Interval):
			execute(ep, alertingConfig, maintenanceConfig, connectivityConfig, disableMonitoringLock, enabledMetrics, debug)
		}
	}
	// Just in case somebody wandered all the way to here and wonders, "what about ExternalEndpoints?"
	// Alerting is checked every time an external endpoint is pushed to Gatus, so they're not monitored
	// periodically like they are for normal endpoints.
}

func execute(ep *endpoint.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock, enabledMetrics, debug bool) {
	if !disableMonitoringLock {
		// By placing the lock here, we prevent multiple endpoints from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		monitoringMutex.Lock()
		defer monitoringMutex.Unlock()
	}
	// If there's a connectivity checker configured, check if Gatus has internet connectivity
	if connectivityConfig != nil && connectivityConfig.Checker != nil && !connectivityConfig.Checker.IsConnected() {
		log.Println("[watchdog.execute] No connectivity; skipping execution")
		return
	}
	if debug {
		log.Printf("[watchdog.execute] Monitoring group=%s; endpoint=%s", ep.Group, ep.Name)
	}
	r := ep.EvaluateHealth()
	if enabledMetrics {
		metrics.PublishMetricsForEndpoint(ep, r)
	}
	UpdateEndpointStatuses(ep, r)
	if debug && !r.Success {
		log.Printf("[watchdog.execute] Monitored group=%s; endpoint=%s; success=%v; errors=%d; duration=%s; body=%s", ep.Group, ep.Name, r.Success, len(r.Errors), r.Duration.Round(time.Millisecond), r.Body)
	} else {
		log.Printf("[watchdog.execute] Monitored group=%s; endpoint=%s; success=%v; errors=%d; duration=%s", ep.Group, ep.Name, r.Success, len(r.Errors), r.Duration.Round(time.Millisecond))
	}
	if !maintenanceConfig.IsUnderMaintenance() {
		// TODO: Consider moving this after the monitoring lock is unlocked? I mean, how much noise can a single alerting provider cause...
		HandleAlerting(ep, r, alertingConfig, debug)
	} else if debug {
		log.Println("[watchdog.execute] Not handling alerting because currently in the maintenance window")
	}
	if debug {
		log.Printf("[watchdog.execute] Waiting for interval=%s before monitoring group=%s endpoint=%s again", ep.Interval, ep.Group, ep.Name)
	}
}

// UpdateEndpointStatuses updates the slice of endpoint statuses
func UpdateEndpointStatuses(ep *endpoint.Endpoint, result *result.Result) {
	if err := store.Get().Insert(ep, result); err != nil {
		log.Println("[watchdog.UpdateEndpointStatuses] Failed to insert result in storage:", err.Error())
	}
}

// Shutdown stops monitoring all endpoints
func Shutdown(cfg *config.Config) {
	// Disable all the old HTTP connections
	for _, ep := range cfg.Endpoints {
		ep.Close()
	}
	cancelFunc()
}

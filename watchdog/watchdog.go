package watchdog

import (
	"context"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/connectivity"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/logr"
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
			go monitor(endpoint, cfg.Alerting, cfg.Maintenance, cfg.Connectivity, cfg.DisableMonitoringLock, cfg.Metrics, ctx)
		}
	}
}

// monitor a single endpoint in a loop
func monitor(ep *endpoint.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock bool, enabledMetrics bool, ctx context.Context) {
	// Run it immediately on start
	execute(ep, alertingConfig, maintenanceConfig, connectivityConfig, disableMonitoringLock, enabledMetrics)
	// Loop for the next executions
	ticker := time.NewTicker(ep.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logr.Warnf("[watchdog.monitor] Canceling current execution of group=%s; endpoint=%s; key=%s", ep.Group, ep.Name, ep.Key())
			return
		case <-ticker.C:
			execute(ep, alertingConfig, maintenanceConfig, connectivityConfig, disableMonitoringLock, enabledMetrics)
		}
	}
	// Just in case somebody wandered all the way to here and wonders, "what about ExternalEndpoints?"
	// Alerting is checked every time an external endpoint is pushed to Gatus, so they're not monitored
	// periodically like they are for normal endpoints.
}

func execute(ep *endpoint.Endpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock bool, enabledMetrics bool) {
	if !disableMonitoringLock {
		// By placing the lock here, we prevent multiple endpoints from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		monitoringMutex.Lock()
		defer monitoringMutex.Unlock()
	}
	// If there's a connectivity checker configured, check if Gatus has internet connectivity
	if connectivityConfig != nil && connectivityConfig.Checker != nil && !connectivityConfig.Checker.IsConnected() {
		logr.Infof("[watchdog.execute] No connectivity; skipping execution")
		return
	}
	logr.Debugf("[watchdog.execute] Monitoring group=%s; endpoint=%s; key=%s", ep.Group, ep.Name, ep.Key())
	result := ep.EvaluateHealth()
	if enabledMetrics {
		metrics.PublishMetricsForEndpoint(ep, result)
	}
	UpdateEndpointStatuses(ep, result)
	if logr.GetThreshold() == logr.LevelDebug && !result.Success {
		logr.Debugf("[watchdog.execute] Monitored group=%s; endpoint=%s; key=%s; success=%v; errors=%d; duration=%s; body=%s", ep.Group, ep.Name, ep.Key(), result.Success, len(result.Errors), result.Duration.Round(time.Millisecond), result.Body)
	} else {
		logr.Infof("[watchdog.execute] Monitored group=%s; endpoint=%s; key=%s; success=%v; errors=%d; duration=%s", ep.Group, ep.Name, ep.Key(), result.Success, len(result.Errors), result.Duration.Round(time.Millisecond))
	}
	inEndpointMaintenanceWindow := false
	for _, maintenanceWindow := range ep.MaintenanceWindows {
		if maintenanceWindow.IsUnderMaintenance() {
			logr.Debug("[watchdog.execute] Under endpoint maintenance window")
			inEndpointMaintenanceWindow = true
		}
	}
	if !maintenanceConfig.IsUnderMaintenance() && !inEndpointMaintenanceWindow {
		// TODO: Consider moving this after the monitoring lock is unlocked? I mean, how much noise can a single alerting provider cause...
		HandleAlerting(ep, result, alertingConfig)
	} else {
		logr.Debug("[watchdog.execute] Not handling alerting because currently in the maintenance window")
	}
	logr.Debugf("[watchdog.execute] Waiting for interval=%s before monitoring group=%s endpoint=%s (key=%s) again", ep.Interval, ep.Group, ep.Name, ep.Key())
}

// UpdateEndpointStatuses updates the slice of endpoint statuses
func UpdateEndpointStatuses(ep *endpoint.Endpoint, result *endpoint.Result) {
	if err := store.Get().Insert(ep, result); err != nil {
		logr.Errorf("[watchdog.UpdateEndpointStatuses] Failed to insert result in storage: %s", err.Error())
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

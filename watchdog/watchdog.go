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
	for _, externalEndpoint := range cfg.ExternalEndpoints {
		// Check if the external endpoint is enabled and is using heartbeat
		// If the external endpoint does not use heartbeat, then it does not need to be monitored periodically, because
		// alerting is checked every time an external endpoint is pushed to Gatus, unlike normal endpoints.
		if externalEndpoint.IsEnabled() && externalEndpoint.Heartbeat.Interval > 0 {
			go monitorExternalEndpointHeartbeat(externalEndpoint, cfg.Alerting, cfg.Maintenance, cfg.Connectivity, cfg.DisableMonitoringLock, cfg.Metrics, ctx)
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

func monitorExternalEndpointHeartbeat(ee *endpoint.ExternalEndpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock bool, enabledMetrics bool, ctx context.Context) {
	ticker := time.NewTicker(ee.Heartbeat.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logr.Warnf("[watchdog.monitorExternalEndpointHeartbeat] Canceling current execution of group=%s; endpoint=%s; key=%s", ee.Group, ee.Name, ee.Key())
			return
		case <-ticker.C:
			executeExternalEndpointHeartbeat(ee, alertingConfig, maintenanceConfig, connectivityConfig, disableMonitoringLock, enabledMetrics)
		}
	}
}

func executeExternalEndpointHeartbeat(ee *endpoint.ExternalEndpoint, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock bool, enabledMetrics bool) {
	if !disableMonitoringLock {
		// By placing the lock here, we prevent multiple endpoints from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		monitoringMutex.Lock()
		defer monitoringMutex.Unlock()
	}
	// If there's a connectivity checker configured, check if Gatus has internet connectivity
	if connectivityConfig != nil && connectivityConfig.Checker != nil && !connectivityConfig.Checker.IsConnected() {
		logr.Infof("[watchdog.monitorExternalEndpointHeartbeat] No connectivity; skipping execution")
		return
	}
	logr.Debugf("[watchdog.monitorExternalEndpointHeartbeat] Checking heartbeat for group=%s; endpoint=%s; key=%s", ee.Group, ee.Name, ee.Key())
	convertedEndpoint := ee.ToEndpoint()
	hasReceivedResultWithinHeartbeatInterval, err := store.Get().HasEndpointStatusNewerThan(ee.Key(), time.Now().Add(-ee.Heartbeat.Interval))
	if err != nil {
		logr.Errorf("[watchdog.monitorExternalEndpointHeartbeat] Failed to check if endpoint has received a result within the heartbeat interval: %s", err.Error())
		return
	}
	if hasReceivedResultWithinHeartbeatInterval {
		// If we received a result within the heartbeat interval, we don't want to create a successful result, so we
		// skip the rest. We don't have to worry about alerting or metrics, because if the previous heartbeat failed
		// while this one succeeds, it implies that there was a new result pushed, and that result being pushed
		// should've resolved the alert.
		logr.Infof("[watchdog.monitorExternalEndpointHeartbeat] Checked heartbeat for group=%s; endpoint=%s; key=%s; success=%v; errors=%d", ee.Group, ee.Name, ee.Key(), hasReceivedResultWithinHeartbeatInterval, 0)
		return
	}
	// All code after this point assumes the heartbeat failed
	result := &endpoint.Result{
		Timestamp: time.Now(),
		Success:   false,
		Errors:    []string{"heartbeat: no update received within " + ee.Heartbeat.Interval.String()},
	}
	if enabledMetrics {
		metrics.PublishMetricsForEndpoint(convertedEndpoint, result)
	}
	UpdateEndpointStatuses(convertedEndpoint, result)
	logr.Infof("[watchdog.monitorExternalEndpointHeartbeat] Checked heartbeat for group=%s; endpoint=%s; key=%s; success=%v; errors=%d; duration=%s", ee.Group, ee.Name, ee.Key(), result.Success, len(result.Errors), result.Duration.Round(time.Millisecond))
	inEndpointMaintenanceWindow := false
	for _, maintenanceWindow := range ee.MaintenanceWindows {
		if maintenanceWindow.IsUnderMaintenance() {
			logr.Debug("[watchdog.monitorExternalEndpointHeartbeat] Under endpoint maintenance window")
			inEndpointMaintenanceWindow = true
		}
	}
	if !maintenanceConfig.IsUnderMaintenance() && !inEndpointMaintenanceWindow {
		HandleAlerting(convertedEndpoint, result, alertingConfig)
		// Sync the failure/success counters back to the external endpoint
		ee.NumberOfSuccessesInARow = convertedEndpoint.NumberOfSuccessesInARow
		ee.NumberOfFailuresInARow = convertedEndpoint.NumberOfFailuresInARow
	} else {
		logr.Debug("[watchdog.monitorExternalEndpointHeartbeat] Not handling alerting because currently in the maintenance window")
	}
	logr.Debugf("[watchdog.monitorExternalEndpointHeartbeat] Waiting for interval=%s before checking heartbeat for group=%s endpoint=%s (key=%s) again", ee.Heartbeat.Interval, ee.Group, ee.Name, ee.Key())
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

package watchdog

import (
	"context"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/logr"
)

// monitorEndpoint a single endpoint in a loop
func monitorEndpoint(ep *endpoint.Endpoint, cfg *config.Config, extraLabels []string, ctx context.Context) {
	// Run it immediately on start
	executeEndpoint(ep, cfg, extraLabels)
	// Loop for the next executions
	ticker := time.NewTicker(ep.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logr.Warnf("[watchdog.monitorEndpoint] Canceling current execution of group=%s; endpoint=%s; key=%s", ep.Group, ep.Name, ep.Key())
			return
		case <-ticker.C:
			executeEndpoint(ep, cfg, extraLabels)
		}
	}
	// Just in case somebody wandered all the way to here and wonders, "what about ExternalEndpoints?"
	// Alerting is checked every time an external endpoint is pushed to Gatus, so they're not monitored
	// periodically like they are for normal endpoints.
}

func executeEndpoint(ep *endpoint.Endpoint, cfg *config.Config, extraLabels []string) {
	// Acquire semaphore to limit concurrent endpoint monitoring
	if err := monitoringSemaphore.Acquire(ctx, 1); err != nil {
		// Only fails if context is cancelled (during shutdown)
		logr.Debugf("[watchdog.executeEndpoint] Context cancelled, skipping execution: %s", err.Error())
		return
	}
	defer monitoringSemaphore.Release(1)
	// If there's a connectivity checker configured, check if Gatus has internet connectivity
	if cfg.Connectivity != nil && cfg.Connectivity.Checker != nil && !cfg.Connectivity.Checker.IsConnected() {
		logr.Infof("[watchdog.executeEndpoint] No connectivity; skipping execution")
		return
	}
	logr.Debugf("[watchdog.executeEndpoint] Monitoring group=%s; endpoint=%s; key=%s", ep.Group, ep.Name, ep.Key())
	result := ep.EvaluateHealth()
	if cfg.Metrics {
		metrics.PublishMetricsForEndpoint(ep, result, extraLabels)
	}
	UpdateEndpointStatus(ep, result)
	if logr.GetThreshold() == logr.LevelDebug && !result.Success {
		logr.Debugf("[watchdog.executeEndpoint] Monitored group=%s; endpoint=%s; key=%s; success=%v; errors=%d; duration=%s; body=%s", ep.Group, ep.Name, ep.Key(), result.Success, len(result.Errors), result.Duration.Round(time.Millisecond), result.Body)
	} else {
		logr.Infof("[watchdog.executeEndpoint] Monitored group=%s; endpoint=%s; key=%s; success=%v; errors=%d; duration=%s", ep.Group, ep.Name, ep.Key(), result.Success, len(result.Errors), result.Duration.Round(time.Millisecond))
	}
	inEndpointMaintenanceWindow := false
	for _, maintenanceWindow := range ep.MaintenanceWindows {
		if maintenanceWindow.IsUnderMaintenance() {
			logr.Debug("[watchdog.executeEndpoint] Under endpoint maintenance window")
			inEndpointMaintenanceWindow = true
		}
	}
	if !cfg.Maintenance.IsUnderMaintenance() && !inEndpointMaintenanceWindow {
		HandleAlerting(ep, result, cfg.Alerting)
	} else {
		logr.Debug("[watchdog.executeEndpoint] Not handling alerting because currently in the maintenance window")
	}
	logr.Debugf("[watchdog.executeEndpoint] Waiting for interval=%s before monitoring group=%s endpoint=%s (key=%s) again", ep.Interval, ep.Group, ep.Name, ep.Key())
}

// UpdateEndpointStatus persists the endpoint result in the storage
func UpdateEndpointStatus(ep *endpoint.Endpoint, result *endpoint.Result) {
	if err := store.Get().InsertEndpointResult(ep, result); err != nil {
		logr.Errorf("[watchdog.UpdateEndpointStatus] Failed to insert result in storage: %s", err.Error())
	}
}

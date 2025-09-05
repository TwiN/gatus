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

func monitorExternalEndpointHeartbeat(ee *endpoint.ExternalEndpoint, cfg *config.Config, extraLabels []string, ctx context.Context) {
	ticker := time.NewTicker(ee.Heartbeat.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logr.Warnf("[watchdog.monitorExternalEndpointHeartbeat] Canceling current execution of group=%s; endpoint=%s; key=%s", ee.Group, ee.Name, ee.Key())
			return
		case <-ticker.C:
			executeExternalEndpointHeartbeat(ee, cfg, extraLabels)
		}
	}
}

func executeExternalEndpointHeartbeat(ee *endpoint.ExternalEndpoint, cfg *config.Config, extraLabels []string) {
	// Acquire semaphore to limit concurrent external endpoint monitoring
	if err := monitoringSemaphore.Acquire(ctx, 1); err != nil {
		// Only fails if context is cancelled (during shutdown)
		logr.Debugf("[watchdog.executeExternalEndpointHeartbeat] Context cancelled, skipping execution: %s", err.Error())
		return
	}
	defer monitoringSemaphore.Release(1)
	// If there's a connectivity checker configured, check if Gatus has internet connectivity
	if cfg.Connectivity != nil && cfg.Connectivity.Checker != nil && !cfg.Connectivity.Checker.IsConnected() {
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
	if cfg.Metrics {
		metrics.PublishMetricsForEndpoint(convertedEndpoint, result, extraLabels)
	}
	UpdateEndpointStatus(convertedEndpoint, result)
	logr.Infof("[watchdog.monitorExternalEndpointHeartbeat] Checked heartbeat for group=%s; endpoint=%s; key=%s; success=%v; errors=%d; duration=%s", ee.Group, ee.Name, ee.Key(), result.Success, len(result.Errors), result.Duration.Round(time.Millisecond))
	inEndpointMaintenanceWindow := false
	for _, maintenanceWindow := range ee.MaintenanceWindows {
		if maintenanceWindow.IsUnderMaintenance() {
			logr.Debug("[watchdog.monitorExternalEndpointHeartbeat] Under endpoint maintenance window")
			inEndpointMaintenanceWindow = true
		}
	}
	if !cfg.Maintenance.IsUnderMaintenance() && !inEndpointMaintenanceWindow {
		HandleAlerting(convertedEndpoint, result, cfg.Alerting)
		// Sync the failure/success counters back to the external endpoint
		ee.NumberOfSuccessesInARow = convertedEndpoint.NumberOfSuccessesInARow
		ee.NumberOfFailuresInARow = convertedEndpoint.NumberOfFailuresInARow
	} else {
		logr.Debug("[watchdog.monitorExternalEndpointHeartbeat] Not handling alerting because currently in the maintenance window")
	}
	logr.Debugf("[watchdog.monitorExternalEndpointHeartbeat] Waiting for interval=%s before checking heartbeat for group=%s endpoint=%s (key=%s) again", ee.Heartbeat.Interval, ee.Group, ee.Name, ee.Key())
}

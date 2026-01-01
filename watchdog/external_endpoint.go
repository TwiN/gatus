package watchdog

import (
	"context"
	"log/slog"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
)

func monitorExternalEndpointHeartbeat(ee *endpoint.ExternalEndpoint, cfg *config.Config, extraLabels []string, ctx context.Context) {
	ticker := time.NewTicker(ee.Heartbeat.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Warn("Canceling current external endpoint execution", "group", ee.Group, "name", ee.Name, "key", ee.Key())
			return
		case <-ticker.C:
			executeExternalEndpointHeartbeat(ee, cfg, extraLabels)
		}
	}
}

func executeExternalEndpointHeartbeat(ee *endpoint.ExternalEndpoint, cfg *config.Config, extraLabels []string) {
	logger := slog.With(ee.GetLogAttribute())

	// Acquire semaphore to limit concurrent external endpoint monitoring
	if err := monitoringSemaphore.Acquire(ctx, 1); err != nil {
		// Only fails if context is cancelled (during shutdown)
		logger.Debug("Context cancelled; skipping execution", "error", err.Error())
		return
	}
	defer monitoringSemaphore.Release(1)
	// If there's a connectivity checker configured, check if Gatus has internet connectivity
	if cfg.Connectivity != nil && cfg.Connectivity.Checker != nil && !cfg.Connectivity.Checker.IsConnected() {
		logger.Info("No connectivity, skipping execution")
		return
	}
	logger.Debug("Monitoring start")
	convertedEndpoint := ee.ToEndpoint()
	hasReceivedResultWithinHeartbeatInterval, err := store.Get().HasEndpointStatusNewerThan(ee.Key(), time.Now().Add(-ee.Heartbeat.Interval))
	if err != nil {
		logger.Error("Failed to check if external endpoint has received a result within the heartbeat interval", "error", err.Error())
		logger.Error("Monitoring error", "error", err.Error())
		return
	}
	if hasReceivedResultWithinHeartbeatInterval {
		// If we received a result within the heartbeat interval, we don't want to create a successful result, so we
		// skip the rest. We don't have to worry about alerting or metrics, because if the previous heartbeat failed
		// while this one succeeds, it implies that there was a new result pushed, and that result being pushed
		// should've resolved the alert.
		logger.Info("Monitoring success, heartbeat received within interval", "success", true, "error_count", 0)
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
	logger.Info("Monitoring done", result.GetLogAttribute())
	inEndpointMaintenanceWindow := false
	for _, maintenanceWindow := range ee.MaintenanceWindows {
		if maintenanceWindow.IsUnderMaintenance() {
			logger.Debug("Under external endpoint maintenance window")
			inEndpointMaintenanceWindow = true
		}
	}
	if !cfg.Maintenance.IsUnderMaintenance() && !inEndpointMaintenanceWindow {
		HandleAlerting(convertedEndpoint, result, cfg.Alerting)
		// Sync the failure/success counters back to the external endpoint
		ee.NumberOfSuccessesInARow = convertedEndpoint.NumberOfSuccessesInARow
		ee.NumberOfFailuresInARow = convertedEndpoint.NumberOfFailuresInARow
	} else {
		logger.Debug("Not handling alerting due to active maintenance window")
	}
}

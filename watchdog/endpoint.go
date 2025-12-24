package watchdog

import (
	"context"
	"log/slog"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/logging"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
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
			slog.Warn("Canceling current execution", "group", ep.Group, "endpoint", ep.Name, "key", ep.Key())
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
		slog.Debug("Context cancelled, skipping execution", "error", err.Error())
		return
	}
	defer monitoringSemaphore.Release(1)
	// If there's a connectivity checker configured, check if Gatus has internet connectivity
	if cfg.Connectivity != nil && cfg.Connectivity.Checker != nil && !cfg.Connectivity.Checker.IsConnected() {
		slog.Info("No connectivity; skipping execution")
		return
	}
	slog.Debug("Monitoring", "group", ep.Group, "endpoint", ep.Name, "key", ep.Key())
	result := ep.EvaluateHealth()
	if cfg.Metrics {
		metrics.PublishMetricsForEndpoint(ep, result, extraLabels)
	}
	UpdateEndpointStatus(ep, result)
	if logging.Level() == slog.LevelDebug && !result.Success { // TODO: Check if it is possible to get the configured level directly from slog
		slog.Debug("Monitored", "group", ep.Group, "endpoint", ep.Name, "key", ep.Key(), "success", result.Success, "errors", len(result.Errors), "duration", result.Duration.Round(time.Millisecond), "body", result.Body)
	} else {
		slog.Info("Monitored", "group", ep.Group, "endpoint", ep.Name, "key", ep.Key(), "success", result.Success, "errors", len(result.Errors), "duration", result.Duration.Round(time.Millisecond))
	}
	inEndpointMaintenanceWindow := false
	for _, maintenanceWindow := range ep.MaintenanceWindows {
		if maintenanceWindow.IsUnderMaintenance() {
			slog.Debug("Under endpoint maintenance window")
			inEndpointMaintenanceWindow = true
		}
	}
	if !cfg.Maintenance.IsUnderMaintenance() && !inEndpointMaintenanceWindow {
		HandleAlerting(ep, result, cfg.Alerting)
	} else {
		slog.Debug("Not handling alerting because currently in the maintenance window")
	}
	slog.Debug("Waiting for next execution", "group", ep.Group, "endpoint", ep.Name, "key", ep.Key(), "interval", ep.Interval.String())
}

// UpdateEndpointStatus persists the endpoint result in the storage
func UpdateEndpointStatus(ep *endpoint.Endpoint, result *endpoint.Result) {
	if err := store.Get().InsertEndpointResult(ep, result); err != nil {
		slog.Error("Failed to insert result in storage", "error", err.Error())
	}
}

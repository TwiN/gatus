package watchdog

import (
	"context"
	"log/slog"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
)

// monitorSuite monitors a suite by executing it at regular intervals
func monitorSuite(s *suite.Suite, cfg *config.Config, extraLabels []string, ctx context.Context) {
	// Execute immediately on start
	executeSuite(s, cfg, extraLabels)
	// Set up ticker for periodic execution
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Warn("Canceling monitoring for suite", "name", s.Name)
			return
		case <-ticker.C:
			executeSuite(s, cfg, extraLabels)
		}
	}
}

// executeSuite executes a suite with proper concurrency control
func executeSuite(s *suite.Suite, cfg *config.Config, extraLabels []string) {
	// Acquire semaphore to limit concurrent suite monitoring
	if err := monitoringSemaphore.Acquire(ctx, 1); err != nil {
		// Only fails if context is cancelled (during shutdown)
		slog.Debug("Context cancelled, skipping execution", "suite", s.Name, "error", err.Error())
		return
	}
	defer monitoringSemaphore.Release(1)
	// Check connectivity if configured
	if cfg.Connectivity != nil && cfg.Connectivity.Checker != nil && !cfg.Connectivity.Checker.IsConnected() {
		slog.Info("No connectivity; skipping suite execution", "name", s.Name)
		return
	}
	slog.Debug("Monitoring suite", "group", s.Group, "name", s.Name, "key", s.Key())
	// Execute the suite using its Execute method
	result := s.Execute()
	// Publish metrics for the suite execution
	if cfg.Metrics {
		metrics.PublishMetricsForSuite(s, result, extraLabels)
	}
	// Store result
	UpdateSuiteStatus(s, result)
	// Handle alerting for suite endpoints
	for i, ep := range s.Endpoints {
		if i < len(result.EndpointResults) {
			epResult := result.EndpointResults[i]
			// Handle alerting if configured and not under maintenance
			if cfg.Alerting != nil && !cfg.Maintenance.IsUnderMaintenance() {
				// Check if endpoint is under maintenance
				inEndpointMaintenanceWindow := false
				for _, maintenanceWindow := range ep.MaintenanceWindows {
					if maintenanceWindow.IsUnderMaintenance() {
						slog.Debug("Endpoint under maintenance window", "suite", s.Name, "endpoint", ep.Name)
						inEndpointMaintenanceWindow = true
						break
					}
				}
				if !inEndpointMaintenanceWindow {
					HandleAlerting(ep, epResult, cfg.Alerting)
				}
			}
		}
	}
	slog.Info("Completed suite execution", slog.Group("details",
		slog.String("name", s.Name),
		slog.Bool("success", result.Success),
		slog.Int("errors", len(result.Errors)),
		slog.Duration("duration", result.Duration),
		slog.Int("endpoints_executed", len(result.EndpointResults)),
		slog.Int("total_endpoints", len(s.Endpoints)),
	))
}

// UpdateSuiteStatus persists the suite result in the database
func UpdateSuiteStatus(s *suite.Suite, result *suite.Result) {
	if err := store.Get().InsertSuiteResult(s, result); err != nil {
		slog.Error("Failed to insert suite result", "suite", s.Name, "error", err.Error())
	}
}

package watchdog

import (
	"context"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/logr"
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
			logr.Warnf("[watchdog.monitorSuite] Canceling monitoring for suite=%s", s.Name)
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
		logr.Debugf("[watchdog.executeSuite] Context cancelled, skipping execution: %s", err.Error())
		return
	}
	defer monitoringSemaphore.Release(1)
	// Check connectivity if configured
	if cfg.Connectivity != nil && cfg.Connectivity.Checker != nil && !cfg.Connectivity.Checker.IsConnected() {
		logr.Infof("[watchdog.executeSuite] No connectivity; skipping suite=%s", s.Name)
		return
	}
	logr.Debugf("[watchdog.executeSuite] Monitoring group=%s; suite=%s; key=%s", s.Group, s.Name, s.Key())
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
						logr.Debug("[watchdog.executeSuite] Endpoint under maintenance window")
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
	logr.Infof("[watchdog.executeSuite] Completed suite=%s; success=%v; errors=%d; duration=%v; endpoints_executed=%d/%d", s.Name, result.Success, len(result.Errors), result.Duration, len(result.EndpointResults), len(s.Endpoints))
}

// UpdateSuiteStatus persists the suite result in the database
func UpdateSuiteStatus(s *suite.Suite, result *suite.Result) {
	if err := store.Get().InsertSuiteResult(s, result); err != nil {
		logr.Errorf("[watchdog.executeSuite] Failed to insert suite result for suite=%s: %v", s.Name, err)
	}
}

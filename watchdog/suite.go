package watchdog

import (
	"context"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/config/connectivity"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/logr"
)

// monitorSuite monitors a suite by executing it at regular intervals
func monitorSuite(s *suite.Suite, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock bool, enabledMetrics bool, extraLabels []string, ctx context.Context) {
	// Execute immediately on start
	executeSuite(s, alertingConfig, maintenanceConfig, connectivityConfig, disableMonitoringLock, enabledMetrics, extraLabels)
	// Set up ticker for periodic execution
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logr.Warnf("[watchdog.monitorSuite] Canceling monitoring for suite=%s", s.Name)
			return
		case <-ticker.C:
			executeSuite(s, alertingConfig, maintenanceConfig, connectivityConfig, disableMonitoringLock, enabledMetrics, extraLabels)
		}
	}
}

// executeSuite executes a suite with proper locking
func executeSuite(s *suite.Suite, alertingConfig *alerting.Config, maintenanceConfig *maintenance.Config, connectivityConfig *connectivity.Config, disableMonitoringLock bool, enabledMetrics bool, extraLabels []string) {
	if !disableMonitoringLock {
		// Use the same monitoring lock to prevent concurrent executions
		monitoringMutex.Lock()
		defer monitoringMutex.Unlock()
	}
	// Check connectivity if configured
	if connectivityConfig != nil && connectivityConfig.Checker != nil && !connectivityConfig.Checker.IsConnected() {
		logr.Infof("[watchdog.executeSuite] No connectivity; skipping suite=%s", s.Name)
		return
	}
	logr.Debugf("[watchdog.executeSuite] Monitoring group=%s; suite=%s; key=%s", s.Group, s.Name, s.Key())
	// Execute the suite using its Execute method
	result := s.Execute()
	// Publish metrics for the suite execution
	if enabledMetrics {
		metrics.PublishMetricsForSuite(s, result, extraLabels)
	}
	// Store individual endpoint results and handle alerting
	for i, ep := range s.Endpoints {
		if i < len(result.EndpointResults) {
			epResult := result.EndpointResults[i]
			// Store the endpoint result
			UpdateEndpointStatus(ep, epResult)
			// Handle alerting if configured and not under maintenance
			if alertingConfig != nil && !maintenanceConfig.IsUnderMaintenance() {
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
					HandleAlerting(ep, epResult, alertingConfig)
				}
			}
		}
	}
	logr.Infof("[watchdog.executeSuite] Completed suite=%s; success=%v; errors=%d; duration=%v; endpoints_executed=%d/%d", s.Name, result.Success, len(result.Errors), result.Duration, len(result.EndpointResults), len(s.Endpoints))
	// Store result in database
	UpdateSuiteStatus(s, result)
}

// UpdateSuiteStatus persists the suite result in the database
func UpdateSuiteStatus(s *suite.Suite, result *suite.Result) {
	if err := store.Get().InsertSuiteResult(s, result); err != nil {
		logr.Errorf("[watchdog.executeSuite] Failed to insert suite result for suite=%s: %v", s.Name, err)
	}
}

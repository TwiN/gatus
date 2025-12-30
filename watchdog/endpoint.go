package watchdog

import (
	"context"
	"fmt"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/state"
	"github.com/TwiN/gatus/v5/metrics"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/logr"
)

type maintenanceStatus int

const (
	noMaintenance maintenanceStatus = iota
	endpointMaintenance
	globalMaintenance
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
	maintenanceState := GetMaintenanceStatus(ep, cfg)
	if maintenanceState != noMaintenance && !result.Success {
		result.State = state.DefaultMaintenanceStateName
	}
	// TODO#227 Evaluate result.Success based on set states' healthiness configuration once that config option is implemented
	if cfg.Metrics {
		metrics.PublishMetricsForEndpoint(ep, result, extraLabels)
	}
	UpdateEndpointStatus(ep, result)
	if logr.GetThreshold() == logr.LevelDebug && !result.Success {
		logr.Debugf("[watchdog.executeEndpoint] Monitored group=%s; endpoint=%s; key=%s; success=%v; errors=%d; duration=%s; body=%s", ep.Group, ep.Name, ep.Key(), result.Success, len(result.Errors), result.Duration.Round(time.Millisecond), result.Body)
	} else {
		logr.Infof("[watchdog.executeEndpoint] Monitored group=%s; endpoint=%s; key=%s; success=%v; errors=%d; duration=%s", ep.Group, ep.Name, ep.Key(), result.Success, len(result.Errors), result.Duration.Round(time.Millisecond))
	}
	if maintenanceState == noMaintenance {
		HandleAlerting(ep, result, cfg.Alerting)
	} else {
		logr.Debug(fmt.Sprintf("[watchdog.executeEndpoint] Not handling alerting because currently in %s maintenance window", GetMaintenanceStatusName(maintenanceState))) // TODO#227 Not sure if fmt.Sprintf is a good idea here since is not used elsewhere
	}
	logr.Debugf("[watchdog.executeEndpoint] Waiting for interval=%s before monitoring group=%s endpoint=%s (key=%s) again", ep.Interval, ep.Group, ep.Name, ep.Key())
}

func GetMaintenanceStatus(ep *endpoint.Endpoint, cfg *config.Config) maintenanceStatus {
	if cfg.Maintenance.IsUnderMaintenance() {
		return globalMaintenance
	}
	for _, maintenanceWindow := range ep.MaintenanceWindows {
		if maintenanceWindow.IsUnderMaintenance() {
			return endpointMaintenance
		}
	}
	return noMaintenance
}

func GetMaintenanceStatusName(status maintenanceStatus) string {
	switch status {
	case globalMaintenance:
		return "global"
	case endpointMaintenance:
		return "endpoint"
	default:
		return "no"
	}
}

// UpdateEndpointStatus persists the endpoint result in the storage
func UpdateEndpointStatus(ep *endpoint.Endpoint, result *endpoint.Result) {
	if err := store.Get().InsertEndpointResult(ep, result); err != nil {
		logr.Errorf("[watchdog.UpdateEndpointStatus] Failed to insert result in storage: %s", err.Error())
	}
}

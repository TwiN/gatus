package watchdog

import (
	"context"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/config"
)

var (
	// monitoringMutex is used to prevent multiple endpoint from being evaluated at the same time.
	// Without this, conditions using response time may become inaccurate.
	monitoringMutex sync.Mutex

	ctx        context.Context
	cancelFunc context.CancelFunc
)

// Monitor loops over each endpoint and starts a goroutine to monitorEndpoint each endpoint separately
func Monitor(cfg *config.Config) {
	ctx, cancelFunc = context.WithCancel(context.Background())
	extraLabels := cfg.GetUniqueExtraMetricLabels()
	for _, endpoint := range cfg.Endpoints {
		if endpoint.IsEnabled() {
			// To prevent multiple requests from running at the same time, we'll wait for a little before each iteration
			time.Sleep(222 * time.Millisecond)
			go monitorEndpoint(endpoint, cfg.Alerting, cfg.Maintenance, cfg.Connectivity, cfg.DisableMonitoringLock, cfg.Metrics, extraLabels, ctx)
		}
	}
	for _, externalEndpoint := range cfg.ExternalEndpoints {
		// Check if the external endpoint is enabled and is using heartbeat
		// If the external endpoint does not use heartbeat, then it does not need to be monitored periodically, because
		// alerting is checked every time an external endpoint is pushed to Gatus, unlike normal endpoints.
		if externalEndpoint.IsEnabled() && externalEndpoint.Heartbeat.Interval > 0 {
			go monitorExternalEndpointHeartbeat(externalEndpoint, cfg.Alerting, cfg.Maintenance, cfg.Connectivity, cfg.DisableMonitoringLock, cfg.Metrics, ctx, extraLabels)
		}
	}
	for _, suite := range cfg.Suites {
		if suite.IsEnabled() {
			time.Sleep(222 * time.Millisecond)
			go monitorSuite(suite, cfg.Alerting, cfg.Maintenance, cfg.Connectivity, cfg.DisableMonitoringLock, cfg.Metrics, extraLabels, ctx)
		}
	}
}

// Shutdown stops monitoring all endpoints
func Shutdown(cfg *config.Config) {
	// Stop in-flight HTTP connections
	for _, ep := range cfg.Endpoints {
		ep.Close()
	}
	for _, s := range cfg.Suites {
		for _, ep := range s.Endpoints {
			ep.Close()
		}
	}
	cancelFunc()
}

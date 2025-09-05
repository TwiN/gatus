package watchdog

import (
	"context"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"golang.org/x/sync/semaphore"
)

const (
	// UnlimitedConcurrencyWeight is the semaphore weight used when concurrency is set to 0 (unlimited).
	// This provides a practical upper limit while allowing very high concurrency for large deployments.
	UnlimitedConcurrencyWeight = 10000
)

var (
	// monitoringSemaphore is used to limit the number of endpoints/suites that can be evaluated concurrently.
	// Without this, conditions using response time may become inaccurate.
	monitoringSemaphore *semaphore.Weighted

	ctx        context.Context
	cancelFunc context.CancelFunc
)

// Monitor loops over each endpoint and starts a goroutine to monitor each endpoint separately
func Monitor(cfg *config.Config) {
	ctx, cancelFunc = context.WithCancel(context.Background())
	// Initialize semaphore based on concurrency configuration
	if cfg.Concurrency == 0 {
		// Unlimited concurrency - use a very high limit
		monitoringSemaphore = semaphore.NewWeighted(UnlimitedConcurrencyWeight)
	} else {
		// Limited concurrency based on configuration
		monitoringSemaphore = semaphore.NewWeighted(int64(cfg.Concurrency))
	}
	extraLabels := cfg.GetUniqueExtraMetricLabels()
	for _, endpoint := range cfg.Endpoints {
		if endpoint.IsEnabled() {
			// To prevent multiple requests from running at the same time, we'll wait for a little before each iteration
			time.Sleep(222 * time.Millisecond)
			go monitorEndpoint(endpoint, cfg, extraLabels, ctx)
		}
	}
	for _, externalEndpoint := range cfg.ExternalEndpoints {
		// Check if the external endpoint is enabled and is using heartbeat
		// If the external endpoint does not use heartbeat, then it does not need to be monitored periodically, because
		// alerting is checked every time an external endpoint is pushed to Gatus, unlike normal endpoints.
		if externalEndpoint.IsEnabled() && externalEndpoint.Heartbeat.Interval > 0 {
			go monitorExternalEndpointHeartbeat(externalEndpoint, cfg, extraLabels, ctx)
		}
	}
	for _, suite := range cfg.Suites {
		if suite.IsEnabled() {
			time.Sleep(222 * time.Millisecond)
			go monitorSuite(suite, cfg, extraLabels, ctx)
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

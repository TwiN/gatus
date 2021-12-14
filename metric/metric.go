package metric

import (
	"strconv"

	"github.com/TwiN/gatus/v3/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// This will be initialized once PublishMetricsForEndpoint.
	// The reason why we're doing this is that if metrics are disabled, we don't want to initialize it unnecessarily.
	resultCount *prometheus.CounterVec = nil
)

// PublishMetricsForEndpoint publishes metrics for the given endpoint and its result.
// These metrics will be exposed at /metrics if the metrics are enabled
func PublishMetricsForEndpoint(endpoint *core.Endpoint, result *core.Result) {
	if resultCount == nil {
		resultCount = promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "gatus_results_total",
			Help: "Number of results per endpoint",
		}, []string{"key", "group", "name", "success"})
	}
	resultCount.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, strconv.FormatBool(result.Success)).Inc()
}

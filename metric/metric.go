package metric

import (
	"strconv"

	"github.com/TwiN/gatus/v3/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	namespace string
	// This will be initialized once PublishMetricsForEndpoint.
	// The reason why we're doing this is that if metrics are disabled, we don't want to initialize it unnecessarily.
	resultCount         *prometheus.CounterVec
	resultSuccessGauge  *prometheus.GaugeVec
	resultDurationGauge *prometheus.GaugeVec
)

func ensurePrometheusMetrics() {
	if namespace == "" {
		namespace = "gatus"

		resultCount = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_total",
			Help:      "Number of results per endpoint",
		}, []string{"key", "group", "name", "success"})

		resultSuccessGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_success",
			Help:      "Displays whether or not the watchdog result was a success",
		}, []string{"key", "group", "name"})

		resultDurationGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_duration_seconds",
			Help:      "Returns how long the watchdog took to complete in seconds",
		}, []string{"key", "group", "name"})
	}
}

// PublishMetricsForEndpoint publishes metrics for the given endpoint and its result.
// These metrics will be exposed at /metrics if the metrics are enabled
func PublishMetricsForEndpoint(endpoint *core.Endpoint, result *core.Result) {
	ensurePrometheusMetrics()

	resultCount.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, strconv.FormatBool(result.Success)).Inc()
	resultSuccessGauge.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name).Set(map[bool]float64{true: 1, false: 0}[result.Success])
	resultDurationGauge.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name).Set(float64(result.Duration.Seconds()))
}

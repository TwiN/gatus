package metric

import (
	"strconv"

	"github.com/TwiN/gatus/v3/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Check if the metric is initialized
	initializedMetrics bool

	// The prefix of the metrics
	namespace string

	// This will be initialized once PublishMetricsForEndpoint.
	// The reason why we're doing this is that if metrics are disabled, we don't want to initialize it unnecessarily.
	resultTotal                              *prometheus.CounterVec
	resultDurationSeconds                    *prometheus.GaugeVec
	resultConnectedTotal                     *prometheus.CounterVec
	resultDNSReturnCodeTotal                 *prometheus.CounterVec
	resultHTTPStatusCodeTotal                *prometheus.CounterVec
	resultSSLLastChainExpiryTimestampSeconds *prometheus.GaugeVec
)

func ensurePrometheusMetrics() {
	if !initializedMetrics {
		initializedMetrics = true
		namespace = "gatus"

		resultTotal = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_total",
			Help:      "Number of results per endpoint",
		}, []string{"key", "group", "name", "type", "success"})

		resultDurationSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_duration_seconds",
			Help:      "Returns how long the watchdog took to complete in seconds",
		}, []string{"key", "group", "name", "type"})

		resultConnectedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_connected_total",
			Help:      "Number of results connected per endpoint",
		}, []string{"key", "group", "name", "type"})

		resultDNSReturnCodeTotal = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_dns_return_code_total",
			Help:      "Number of results DNS return code",
		}, []string{"key", "group", "name", "type", "code"})

		resultHTTPStatusCodeTotal = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_http_status_code_total",
			Help:      "Number of results HTTP status code",
		}, []string{"key", "group", "name", "type", "code"})

		resultSSLLastChainExpiryTimestampSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_ssl_last_chain_expiry_timestamp_seconds",
			Help:      "Number of results last SSL chain expiry in timestamp seconds",
		}, []string{"key", "group", "name", "type"})
	}
}

// PublishMetricsForEndpoint publishes metrics for the given endpoint and its result.
// These metrics will be exposed at /metrics if the metrics are enabled
func PublishMetricsForEndpoint(endpoint *core.Endpoint, result *core.Result) {
	ensurePrometheusMetrics()

	endpointType := endpoint.Type()

	resultTotal.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType), strconv.FormatBool(result.Success)).Inc()
	resultDurationSeconds.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType)).Set(float64(result.Duration.Seconds()))

	if result.Connected {
		resultConnectedTotal.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType)).Inc()
	}
	if result.DNSRCode != "" {
		resultDNSReturnCodeTotal.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType), result.DNSRCode).Inc()
	}
	if result.HTTPStatus != 0 {
		resultHTTPStatusCodeTotal.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType), strconv.Itoa(result.HTTPStatus)).Inc()
	}
	if result.CertificateExpiration != 0 {
		resultSSLLastChainExpiryTimestampSeconds.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType)).Set(float64(result.CertificateExpiration.Seconds()))
	}
}

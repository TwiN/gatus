package metrics

import (
	"strconv"

	"github.com/TwiN/gatus/v5/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "gatus" // The prefix of the metrics

var (
	initializedMetrics bool // Whether the metrics have been initialized

	resultTotal                        *prometheus.CounterVec
	resultDurationSeconds              *prometheus.GaugeVec
	resultConnectedTotal               *prometheus.CounterVec
	resultCodeTotal                    *prometheus.CounterVec
	resultCertificateExpirationSeconds *prometheus.GaugeVec
)

func initializePrometheusMetrics() {
	resultTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "results_total",
		Help:      "Number of results per endpoint",
	}, []string{"key", "group", "name", "type", "success"})
	resultDurationSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "results_duration_seconds",
		Help:      "Duration of the request in seconds",
	}, []string{"key", "group", "name", "type"})
	resultConnectedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "results_connected_total",
		Help:      "Total number of results in which a connection was successfully established",
	}, []string{"key", "group", "name", "type"})
	resultCodeTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "results_code_total",
		Help:      "Total number of results by code",
	}, []string{"key", "group", "name", "type", "code"})
	resultCertificateExpirationSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "results_certificate_expiration_seconds",
		Help:      "Number of seconds until the certificate expires",
	}, []string{"key", "group", "name", "type"})
}

// PublishMetricsForEndpoint publishes metrics for the given endpoint and its result.
// These metrics will be exposed at /metrics if the metrics are enabled
func PublishMetricsForEndpoint(endpoint *core.Endpoint, result *core.Result) {
	if !initializedMetrics {
		initializePrometheusMetrics()
		initializedMetrics = true
	}
	endpointType := endpoint.Type()
	resultTotal.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType), strconv.FormatBool(result.Success)).Inc()
	resultDurationSeconds.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType)).Set(result.Duration.Seconds())
	if result.Connected {
		resultConnectedTotal.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType)).Inc()
	}
	if result.DNSRCode != "" {
		resultCodeTotal.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType), result.DNSRCode).Inc()
	}
	if result.HTTPStatus != 0 {
		resultCodeTotal.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType), strconv.Itoa(result.HTTPStatus)).Inc()
	}
	if result.CertificateExpiration != 0 {
		resultCertificateExpirationSeconds.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, string(endpointType)).Set(result.CertificateExpiration.Seconds())
	}
}

package metrics

import (
	"strconv"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const namespace = "gatus" // The prefix of the metrics

var (
	resultTotal                        *prometheus.CounterVec
	resultDurationSeconds              *prometheus.GaugeVec
	resultConnectedTotal               *prometheus.CounterVec
	resultCodeTotal                    *prometheus.CounterVec
	resultCertificateExpirationSeconds *prometheus.GaugeVec
	resultEndpointSuccess              *prometheus.GaugeVec
)

func InitializePrometheusMetrics(cfg *config.Config) {
	labels := cfg.GetMetricLabels()
	resultTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "results_total",
		Help:      "Number of results per endpoint",
	}, append([]string{"key", "group", "name", "type", "success"}, labels...))
	resultDurationSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "results_duration_seconds",
		Help:      "Duration of the request in seconds",
	}, append([]string{"key", "group", "name", "type"}, labels...))
	resultConnectedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "results_connected_total",
		Help:      "Total number of results in which a connection was successfully established",
	}, append([]string{"key", "group", "name", "type"}, labels...))
	resultCodeTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "results_code_total",
		Help:      "Total number of results by code",
	}, append([]string{"key", "group", "name", "type", "code"}, labels...))
	resultCertificateExpirationSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "results_certificate_expiration_seconds",
		Help:      "Number of seconds until the certificate expires",
	}, append([]string{"key", "group", "name", "type"}, labels...))
	resultEndpointSuccess = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "results_endpoint_success",
		Help:      "Displays whether or not the endpoint was a success",
	}, append([]string{"key", "group", "name", "type"}, labels...))
}

// PublishMetricsForEndpoint publishes metrics for the given endpoint and its result.
// These metrics will be exposed at /metrics if the metrics are enabled
func PublishMetricsForEndpoint(ep *endpoint.Endpoint, result *endpoint.Result, labels []string) {
	labelValues := []string{}
	for _, label := range labels {
		if value, ok := ep.Labels[label]; ok {
			labelValues = append(labelValues, value)
		} else {
			labelValues = append(labelValues, "")
		}
	}

	endpointType := ep.Type()
	resultTotal.WithLabelValues(append([]string{ep.Key(), ep.Group, ep.Name, string(endpointType), strconv.FormatBool(result.Success)}, labelValues...)...).Inc()
	resultDurationSeconds.WithLabelValues(append([]string{ep.Key(), ep.Group, ep.Name, string(endpointType)}, labelValues...)...).Set(result.Duration.Seconds())
	if result.Connected {
		resultConnectedTotal.WithLabelValues(append([]string{ep.Key(), ep.Group, ep.Name, string(endpointType)}, labelValues...)...).Inc()
	}
	if result.DNSRCode != "" {
		resultCodeTotal.WithLabelValues(append([]string{ep.Key(), ep.Group, ep.Name, string(endpointType), result.DNSRCode}, labelValues...)...).Inc()
	}
	if result.HTTPStatus != 0 {
		resultCodeTotal.WithLabelValues(append([]string{ep.Key(), ep.Group, ep.Name, string(endpointType), strconv.Itoa(result.HTTPStatus)}, labelValues...)...).Inc()
	}
	if result.CertificateExpiration != 0 {
		resultCertificateExpirationSeconds.WithLabelValues(append([]string{ep.Key(), ep.Group, ep.Name, string(endpointType)}, labelValues...)...).Set(result.CertificateExpiration.Seconds())
	}
	if result.Success {
		resultEndpointSuccess.WithLabelValues(append([]string{ep.Key(), ep.Group, ep.Name, string(endpointType)}, labelValues...)...).Set(1)
	} else {
		resultEndpointSuccess.WithLabelValues(append([]string{ep.Key(), ep.Group, ep.Name, string(endpointType)}, labelValues...)...).Set(0)
	}
}

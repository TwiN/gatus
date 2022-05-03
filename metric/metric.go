package metric

import (
	"strconv"
	"time"

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
	resultCount         *prometheus.CounterVec
	resultSuccessGauge  *prometheus.GaugeVec
	resultDurationGauge *prometheus.GaugeVec

	resultDNSReturnCode *prometheus.CounterVec

	resultTCPConnected *prometheus.CounterVec

	resultStartTLSConnected                          *prometheus.CounterVec
	resultStartTLSSSLLastChainExpiryTimestampSeconds *prometheus.GaugeVec

	resultTLSConnected                          *prometheus.CounterVec
	resultTLSSSLLastChainExpiryTimestampSeconds *prometheus.GaugeVec

	resultHTTPConnected                          *prometheus.CounterVec
	resultHTTPStatusCode                         *prometheus.CounterVec
	resultHTTPSSLLastChainExpiryTimestampSeconds *prometheus.GaugeVec
)

func ensurePrometheusMetrics() {
	if !initializedMetrics {
		initializedMetrics = true
		namespace = "gatus"

		resultCount = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_total",
			Help:      "Number of results per endpoint",
		}, []string{"key", "group", "name", "url", "success"})

		resultSuccessGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_success",
			Help:      "Displays whether or not the watchdog result was a success",
		}, []string{"key", "group", "name", "url"})

		resultDurationGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_duration_seconds",
			Help:      "Returns how long the watchdog took to complete in seconds",
		}, []string{"key", "group", "name", "url"})

		resultDNSReturnCode = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_dns_return_code",
			Help:      "Response DNS return code",
		}, []string{"key", "group", "name", "url", "code"})

		resultTCPConnected = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_tcp_connected",
			Help:      "Response TCP connected count",
		}, []string{"key", "group", "name", "url"})

		resultStartTLSConnected = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_starttls_connected",
			Help:      "Response StartTLS connected count",
		}, []string{"key", "group", "name", "url"})

		resultStartTLSSSLLastChainExpiryTimestampSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_starttls_ssl_last_chain_expiry_timestamp_seconds",
			Help:      "Returns last SSL chain expiry in timestamp seconds",
		}, []string{"key", "group", "name", "url"})

		resultTLSConnected = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_tls_connected",
			Help:      "Response TLS connected count",
		}, []string{"key", "group", "name", "url"})

		resultTLSSSLLastChainExpiryTimestampSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_tls_ssl_last_chain_expiry_timestamp_seconds",
			Help:      "Returns last SSL chain expiry in timestamp seconds",
		}, []string{"key", "group", "name", "url"})

		resultHTTPConnected = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_http_connected",
			Help:      "Response HTTP connected count",
		}, []string{"key", "group", "name", "url"})

		resultHTTPStatusCode = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_http_status_code",
			Help:      "Response HTTP status code",
		}, []string{"key", "group", "name", "url", "code"})

		resultHTTPSSLLastChainExpiryTimestampSeconds = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_http_ssl_last_chain_expiry_timestamp_seconds",
			Help:      "Returns last SSL chain expiry in timestamp seconds",
		}, []string{"key", "group", "name", "url"})
	}
}

// PublishMetricsForEndpoint publishes metrics for the given endpoint and its result.
// These metrics will be exposed at /metrics if the metrics are enabled
func PublishMetricsForEndpoint(endpoint *core.Endpoint, result *core.Result) {
	ensurePrometheusMetrics()

	resultCount.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL, strconv.FormatBool(result.Success)).Inc()
	resultSuccessGauge.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Set(map[bool]float64{true: 1, false: 0}[result.Success])
	resultDurationGauge.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Set(float64(result.Duration.Seconds()))

	switch endpointType := endpoint.Type(); endpointType {
	case core.EndpointTypeDNS:
		resultDNSReturnCode.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL, result.DNSRCode).Inc()
	case core.EndpointTypeTCP:
		if result.Connected {
			resultTCPConnected.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Inc()
		}
	case core.EndpointTypeICMP:
	case core.EndpointTypeSTARTTLS:
		if result.Connected {
			resultStartTLSConnected.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Inc()
		}
		if result.CertificateExpiration != 0 {
			resultStartTLSSSLLastChainExpiryTimestampSeconds.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Set(float64(result.CertificateExpiration / time.Second))
		}
	case core.EndpointTypeTLS:
		if result.Connected {
			resultTLSConnected.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Inc()
		}
		if result.CertificateExpiration != 0 {
			resultTLSSSLLastChainExpiryTimestampSeconds.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Set(float64(result.CertificateExpiration / time.Second))
		}
	case core.EndpointTypeHTTP:
		if result.Connected {
			resultHTTPConnected.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Inc()
		}
		if result.HTTPStatus != 0 {
			resultHTTPStatusCode.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL, strconv.Itoa(result.HTTPStatus)).Inc()
		}
		if result.CertificateExpiration != 0 {
			resultHTTPSSLLastChainExpiryTimestampSeconds.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Set(float64(result.CertificateExpiration / time.Second))
		}
	}
}

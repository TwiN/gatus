package metric

import (
	"strconv"
	"strings"
	"time"

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

	resultHTTPStatusCode                         *prometheus.CounterVec
	resultHTTPSSL                                *prometheus.GaugeVec
	resultHTTPSSLLastChainExpiryTimestampSeconds *prometheus.GaugeVec
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

		resultHTTPStatusCode = promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "results_http_status_code",
			Help:      "Response HTTP status code",
		}, []string{"key", "group", "name", "url", "code"})

		resultHTTPSSL = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "results_http_ssl",
			Help:      "Indicates if SSL was used for the final redirect",
		}, []string{"key", "group", "name", "url"})

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

	resultCount.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, strconv.FormatBool(result.Success)).Inc()
	resultSuccessGauge.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name).Set(map[bool]float64{true: 1, false: 0}[result.Success])
	resultDurationGauge.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name).Set(float64(result.Duration.Seconds()))

	switch {
	// DNS
	case endpoint.DNS != nil:
	// TCP
	case strings.HasPrefix(endpoint.URL, "tcp://"):
	// ICMP
	case strings.HasPrefix(endpoint.URL, "icmp://"):
	// STARTTLS
	case strings.HasPrefix(endpoint.URL, "starttls://"):
	// TLS
	case strings.HasPrefix(endpoint.URL, "tls://"):
	// HTTP
	default:
		resultHTTPStatusCode.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL, strconv.Itoa(result.HTTPStatus)).Inc()
		if result.CertificateExpiration != 0 {
			resultHTTPSSL.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Set(1)
			resultHTTPSSLLastChainExpiryTimestampSeconds.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Set(float64(result.CertificateExpiration / time.Second))
		} else {
			resultHTTPSSL.WithLabelValues(endpoint.Key(), endpoint.Group, endpoint.Name, endpoint.URL).Set(0)
		}
	}
}

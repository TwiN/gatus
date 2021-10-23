package metric

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/TwiN/gatus/v3/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	gauges = map[string]*prometheus.GaugeVec{}
	rwLock sync.RWMutex
)

// PublishMetricsForEndpoint publishes metrics for the given endpoint and its result.
// These metrics will be exposed at /metrics if the metrics are enabled
func PublishMetricsForEndpoint(endpoint *core.Endpoint, result *core.Result) {
	rwLock.Lock()
	gauge, exists := gauges[fmt.Sprintf("%s_%s", endpoint.Name, endpoint.URL)]
	if !exists {
		gauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
			Subsystem: "gatus",
			Name:      "tasks",
			// TODO: remove the "service" key in v4.0.0, as it is only kept for backward compatibility
			ConstLabels: prometheus.Labels{"service": endpoint.Name, "endpoint": endpoint.Name, "url": endpoint.URL},
		}, []string{"status", "success"})
		gauges[fmt.Sprintf("%s_%s", endpoint.Name, endpoint.URL)] = gauge
	}
	rwLock.Unlock()
	gauge.WithLabelValues(strconv.Itoa(result.HTTPStatus), strconv.FormatBool(result.Success)).Inc()
}

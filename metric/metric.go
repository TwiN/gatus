package metric

import (
	"fmt"
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"strconv"
	"sync"
)

var (
	gauges = map[string]*prometheus.GaugeVec{}
	rwLock sync.RWMutex
)

// PublishMetricsForService publishes metrics for the given service and its result.
// These metrics will be exposed at /metrics if the metrics are enabled
func PublishMetricsForService(service *core.Service, result *core.Result) {
	if config.Get().Metrics {
		rwLock.Lock()
		gauge, exists := gauges[fmt.Sprintf("%s_%s", service.Name, service.Url)]
		if !exists {
			gauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
				Subsystem:   "gatus",
				Name:        "tasks",
				ConstLabels: prometheus.Labels{"service": service.Name, "url": service.Url},
			}, []string{"status", "success"})
			gauges[fmt.Sprintf("%s_%s", service.Name, service.Url)] = gauge
		}
		rwLock.Unlock()
		gauge.WithLabelValues(strconv.Itoa(result.HttpStatus), strconv.FormatBool(result.Success)).Inc()
	}
}

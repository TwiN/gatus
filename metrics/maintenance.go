package metrics

import (
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/prometheus/client_golang/prometheus"
)

// endpointMaintenanceCollector evaluates maintenance at scrape time, independently
// of endpoint results. This also keeps external endpoints and long check intervals
// up to date when a maintenance window starts or ends.
type endpointMaintenanceCollector struct {
	desc        *prometheus.Desc
	config      *config.Config
	extraLabels []string
}

func newEndpointMaintenanceCollector(cfg *config.Config, extraLabels []string) *endpointMaintenanceCollector {
	return &endpointMaintenanceCollector{
		desc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "results_endpoint_maintenance"),
			"Whether the endpoint is currently under maintenance (1) or not (0)",
			append([]string{"key", "group", "name", "type"}, extraLabels...), nil,
		),
		config:      cfg,
		extraLabels: extraLabels,
	}
}

func (c *endpointMaintenanceCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.desc
}

func (c *endpointMaintenanceCollector) Collect(ch chan<- prometheus.Metric) {
	globalMaintenance := c.config.Maintenance != nil && c.config.Maintenance.IsUnderMaintenance()
	for _, ep := range c.config.Endpoints {
		if ep.IsEnabled() {
			c.collectEndpoint(ch, ep, ep.MaintenanceWindows, globalMaintenance)
		}
	}
	for _, ep := range c.config.ExternalEndpoints {
		if ep.IsEnabled() {
			// Copy only static identity fields, not the counters updated by the watchdog.
			converted := &endpoint.Endpoint{Name: ep.Name, Group: ep.Group}
			c.collectEndpoint(ch, converted, ep.MaintenanceWindows, globalMaintenance)
		}
	}
}

func (c *endpointMaintenanceCollector) collectEndpoint(ch chan<- prometheus.Metric, ep *endpoint.Endpoint, windows []*maintenance.Config, globalMaintenance bool) {
	value := 0.0
	if globalMaintenance {
		value = 1
	} else {
		for _, window := range windows {
			if window.IsUnderMaintenance() {
				value = 1
				break
			}
		}
	}
	labelValues := []string{ep.Key(), ep.Group, ep.Name, string(ep.Type())}
	for _, label := range c.extraLabels {
		labelValues = append(labelValues, ep.ExtraLabels[label])
	}
	ch <- prometheus.MustNewConstMetric(c.desc, prometheus.GaugeValue, value, labelValues...)
}

package metrics

import (
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/maintenance"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func maintenanceWindow(t *testing.T, start string, duration time.Duration) *maintenance.Config {
	t.Helper()
	window := &maintenance.Config{Start: start, Duration: duration}
	if err := window.ValidateAndSetDefaults(); err != nil {
		t.Fatal(err)
	}
	return window
}

func assertMaintenanceMetric(t *testing.T, reg *prometheus.Registry, samples string) {
	t.Helper()
	expected := "# HELP gatus_results_endpoint_maintenance Whether the endpoint is currently under maintenance (1) or not (0)\n" +
		"# TYPE gatus_results_endpoint_maintenance gauge\n" + samples
	if err := testutil.GatherAndCompare(reg, strings.NewReader(expected), "gatus_results_endpoint_maintenance"); err != nil {
		t.Fatal(err)
	}
}

func TestEndpointMaintenanceWindows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		t.Cleanup(UnregisterPrometheusMetrics)
		// synctest starts at midnight UTC. Evaluate well inside the window.
		time.Sleep(90 * time.Minute)
		active := maintenanceWindow(t, "01:00", time.Hour)
		inactive := maintenanceWindow(t, "03:00", time.Hour)
		disabled := maintenance.GetDefaultConfig()
		cases := []struct {
			name    string
			global  *maintenance.Config
			windows []*maintenance.Config
			want    int
		}{
			{name: "unconfigured"},
			{name: "disabled global", global: disabled},
			{name: "inactive global", global: inactive},
			{name: "active global", global: active, want: 1},
			{name: "active endpoint", windows: []*maintenance.Config{active}, want: 1},
			{name: "inactive endpoint", windows: []*maintenance.Config{inactive}},
			{name: "disabled endpoint", windows: []*maintenance.Config{disabled}},
			{name: "any endpoint window", windows: []*maintenance.Config{inactive, active}, want: 1},
			{name: "global or endpoint", global: inactive, windows: []*maintenance.Config{active}, want: 1},
			{name: "endpoint cannot disable global", global: active, windows: []*maintenance.Config{disabled}, want: 1},
		}
		for _, tc := range cases {
			t.Log(tc.name)
			cfg := &config.Config{
				Maintenance: tc.global,
				Endpoints:   []*endpoint.Endpoint{{Name: "api", Group: "prod", URL: "https://example.org", MaintenanceWindows: tc.windows}},
			}
			reg := prometheus.NewRegistry()
			InitializePrometheusMetrics(cfg, reg)
			assertMaintenanceMetric(t, reg, fmt.Sprintf("gatus_results_endpoint_maintenance{group=\"prod\",key=\"prod_api\",name=\"api\",type=\"HTTP\"} %d\n", tc.want))
		}
	})
}

func TestEndpointMaintenanceUpdatesOnScrape(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cfg := &config.Config{Endpoints: []*endpoint.Endpoint{{
			Name: "api", URL: "https://example.org",
			MaintenanceWindows: []*maintenance.Config{maintenanceWindow(t, "01:00", time.Hour)},
		}}}
		reg := prometheus.NewRegistry()
		InitializePrometheusMetrics(cfg, reg)
		t.Cleanup(UnregisterPrometheusMetrics)
		// Scrape before the first result and before publishing each subsequent result.
		// Maintenance must track wall time independently of the last result.
		for _, want := range []int{0, 1, 0} {
			assertMaintenanceMetric(t, reg, fmt.Sprintf("gatus_results_endpoint_maintenance{group=\"\",key=\"_api\",name=\"api\",type=\"HTTP\"} %d\n", want))
			// A failed result during maintenance must still be recorded as a failure.
			PublishMetricsForEndpoint(cfg.Endpoints[0], &endpoint.Result{Success: false}, nil)
			if value := testutil.ToFloat64(resultEndpointSuccess.WithLabelValues("_api", "", "api", "HTTP")); value != 0 {
				t.Fatalf("maintenance changed the endpoint success metric: %v", value)
			}
			time.Sleep(90 * time.Minute)
		}
	})
}

func TestEndpointMaintenanceExternalAndExtraLabels(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		time.Sleep(90 * time.Minute)
		disabled := false
		cfg := &config.Config{
			Endpoints: []*endpoint.Endpoint{
				{Name: "api", URL: "https://example.org", ExtraLabels: map[string]string{"team": "backend"}},
				{Name: "disabled", URL: "https://example.org", Enabled: &disabled},
			},
			ExternalEndpoints: []*endpoint.ExternalEndpoint{
				{Name: "backup", MaintenanceWindows: []*maintenance.Config{maintenanceWindow(t, "01:00", time.Hour)}},
				{Name: "disabled-backup", Enabled: &disabled},
			},
		}
		reg := prometheus.NewRegistry()
		InitializePrometheusMetrics(cfg, reg)
		t.Cleanup(UnregisterPrometheusMetrics)
		assertMaintenanceMetric(t, reg, `gatus_results_endpoint_maintenance{group="",key="_api",name="api",team="backend",type="HTTP"} 0
gatus_results_endpoint_maintenance{group="",key="_backup",name="backup",team="",type="UNKNOWN"} 1
`)
	})
}

func TestEndpointMaintenanceReload(t *testing.T) {
	reg := prometheus.NewRegistry()
	InitializePrometheusMetrics(&config.Config{Endpoints: []*endpoint.Endpoint{{Name: "old", URL: "https://example.org"}}}, reg)
	assertMaintenanceMetric(t, reg, "gatus_results_endpoint_maintenance{group=\"\",key=\"_old\",name=\"old\",type=\"HTTP\"} 0\n")
	InitializePrometheusMetrics(&config.Config{Endpoints: []*endpoint.Endpoint{{Name: "new", URL: "https://example.org"}}}, reg)
	t.Cleanup(UnregisterPrometheusMetrics)
	assertMaintenanceMetric(t, reg, "gatus_results_endpoint_maintenance{group=\"\",key=\"_new\",name=\"new\",type=\"HTTP\"} 0\n")
	UnregisterPrometheusMetrics()
	count, err := testutil.GatherAndCount(reg, "gatus_results_endpoint_maintenance")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("maintenance metric remains registered after shutdown: %d", count)
	}
}

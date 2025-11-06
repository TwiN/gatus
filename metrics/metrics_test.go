package metrics

import (
	"bytes"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/dns"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// TestInitializePrometheusMetrics tests metrics initialization with extraLabels.
// Note: Because of the global Prometheus registry, this test can only safely verify one label set per process.
// If the function is called with a different set of labels for the same metric, a panic will occur.
func TestInitializePrometheusMetrics(t *testing.T) {
	cfgWithExtras := &config.Config{
		Endpoints: []*endpoint.Endpoint{
			{
				Name:  "TestEP",
				Group: "G",
				URL:   "http://x/",
				ExtraLabels: map[string]string{
					"foo":   "foo-val",
					"hello": "world-val",
				},
			},
		},
	}
	reg := prometheus.NewRegistry()
	InitializePrometheusMetrics(cfgWithExtras, reg)
	// Metrics variables should be non-nil
	if resultTotal == nil {
		t.Error("resultTotal metric not initialized")
	}
	if resultDurationSeconds == nil {
		t.Error("resultDurationSeconds metric not initialized")
	}
	if resultConnectedTotal == nil {
		t.Error("resultConnectedTotal metric not initialized")
	}
	if resultCodeTotal == nil {
		t.Error("resultCodeTotal metric not initialized")
	}
	if resultCertificateExpirationSeconds == nil {
		t.Error("resultCertificateExpirationSeconds metric not initialized")
	}
	if resultDomainExpirationSeconds == nil {
		t.Error("resultDomainExpirationSeconds metric not initialized")
	}
	if resultEndpointSuccess == nil {
		t.Error("resultEndpointSuccess metric not initialized")
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("resultTotal.WithLabelValues panicked: %v", r)
		}
	}()
	_ = resultTotal.WithLabelValues("k", "g", "n", "ty", "true", "fval", "hval")
}

// TestPublishMetricsForEndpoint_withExtraLabels ensures extraLabels are included in the exported metrics.
func TestPublishMetricsForEndpoint_withExtraLabels(t *testing.T) {
	// Only test one label set per process due to Prometheus registry limits.
	reg := prometheus.NewRegistry()
	cfg := &config.Config{
		Endpoints: []*endpoint.Endpoint{
			{
				Name: "ep-extra",
				URL:  "https://sample.com",
				ExtraLabels: map[string]string{
					"foo": "my-foo",
					"bar": "my-bar",
				},
			},
		},
	}
	InitializePrometheusMetrics(cfg, reg)

	ep := &endpoint.Endpoint{
		Name:  "ep-extra",
		Group: "g1",
		URL:   "https://sample.com",
		ExtraLabels: map[string]string{
			"foo": "my-foo",
			"bar": "my-bar",
		},
	}
	result := &endpoint.Result{
		HTTPStatus: 200,
		Connected:  true,
		Duration:   2340 * time.Millisecond,
		Success:    true,
	}
	// Get labels in sorted order as per GetUniqueExtraMetricLabels
	extraLabels := cfg.GetUniqueExtraMetricLabels()
	PublishMetricsForEndpoint(ep, result, extraLabels)

	expected := `
# HELP gatus_results_total Number of results per endpoint
# TYPE gatus_results_total counter
gatus_results_total{bar="my-bar",foo="my-foo",group="g1",key="g1_ep-extra",name="ep-extra",success="true",type="HTTP"} 1
`
	err := testutil.GatherAndCompare(reg, bytes.NewBufferString(expected), "gatus_results_total")
	if err != nil {
		t.Error("metrics export does not include extraLabels as expected:", err)
	}
}

func TestPublishMetricsForEndpoint(t *testing.T) {
	reg := prometheus.NewRegistry()
	InitializePrometheusMetrics(&config.Config{}, reg)

	httpEndpoint := &endpoint.Endpoint{Name: "http-ep-name", Group: "http-ep-group", URL: "https://example.org"}
	PublishMetricsForEndpoint(httpEndpoint, &endpoint.Result{
		HTTPStatus: 200,
		Connected:  true,
		Duration:   123 * time.Millisecond,
		ConditionResults: []*endpoint.ConditionResult{
			{Condition: "[STATUS] == 200", Success: true},
			{Condition: "[CERTIFICATE_EXPIRATION] > 48h", Success: true},
			{Condition: "[DOMAIN_EXPIRATION] > 24h", Success: true},
		},
		Success:               true,
		CertificateExpiration: 49 * time.Hour,
		DomainExpiration:      25 * time.Hour,
	}, []string{})
	err := testutil.GatherAndCompare(reg, bytes.NewBufferString(`
# HELP gatus_results_code_total Total number of results by code
# TYPE gatus_results_code_total counter
gatus_results_code_total{code="200",group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 1
# HELP gatus_results_connected_total Total number of results in which a connection was successfully established
# TYPE gatus_results_connected_total counter
gatus_results_connected_total{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 1
# HELP gatus_results_duration_seconds Duration of the request in seconds
# TYPE gatus_results_duration_seconds gauge
gatus_results_duration_seconds{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 0.123
# HELP gatus_results_total Number of results per endpoint
# TYPE gatus_results_total counter
gatus_results_total{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",success="true",type="HTTP"} 1
# HELP gatus_results_certificate_expiration_seconds Number of seconds until the certificate expires
# TYPE gatus_results_certificate_expiration_seconds gauge
gatus_results_certificate_expiration_seconds{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 176400
# HELP gatus_results_domain_expiration_seconds Number of seconds until the domain expires
# TYPE gatus_results_domain_expiration_seconds gauge
gatus_results_domain_expiration_seconds{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 90000
# HELP gatus_results_endpoint_success Displays whether or not the endpoint was a success
# TYPE gatus_results_endpoint_success gauge
gatus_results_endpoint_success{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 1
`), "gatus_results_code_total", "gatus_results_connected_total", "gatus_results_duration_seconds", "gatus_results_total", "gatus_results_certificate_expiration_seconds", "gatus_results_endpoint_success")
	if err != nil {
		t.Errorf("Expected no errors but got: %v", err)
	}
	PublishMetricsForEndpoint(httpEndpoint, &endpoint.Result{
		HTTPStatus: 200,
		Connected:  true,
		Duration:   125 * time.Millisecond,
		ConditionResults: []*endpoint.ConditionResult{
			{Condition: "[STATUS] == 200", Success: true},
			{Condition: "[CERTIFICATE_EXPIRATION] > 47h", Success: false},
			{Condition: "[DOMAIN_EXPIRATION] > 24h", Success: true},
		},
		Success:               false,
		CertificateExpiration: 47 * time.Hour,
		DomainExpiration:      24 * time.Hour,
	}, []string{})
	err = testutil.GatherAndCompare(reg, bytes.NewBufferString(`
# HELP gatus_results_code_total Total number of results by code
# TYPE gatus_results_code_total counter
gatus_results_code_total{code="200",group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 2
# HELP gatus_results_connected_total Total number of results in which a connection was successfully established
# TYPE gatus_results_connected_total counter
gatus_results_connected_total{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 2
# HELP gatus_results_duration_seconds Duration of the request in seconds
# TYPE gatus_results_duration_seconds gauge
gatus_results_duration_seconds{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 0.125
# HELP gatus_results_total Number of results per endpoint
# TYPE gatus_results_total counter
gatus_results_total{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",success="false",type="HTTP"} 1
gatus_results_total{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",success="true",type="HTTP"} 1
# HELP gatus_results_certificate_expiration_seconds Number of seconds until the certificate expires
# TYPE gatus_results_certificate_expiration_seconds gauge
gatus_results_certificate_expiration_seconds{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 169200
# HELP gatus_results_domain_expiration_seconds Number of seconds until the domain expires
# TYPE gatus_results_domain_expiration_seconds gauge
gatus_results_domain_expiration_seconds{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 86400
# HELP gatus_results_endpoint_success Displays whether or not the endpoint was a success
# TYPE gatus_results_endpoint_success gauge
gatus_results_endpoint_success{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 0
`), "gatus_results_code_total", "gatus_results_connected_total", "gatus_results_duration_seconds", "gatus_results_total", "gatus_results_certificate_expiration_seconds", "gatus_results_endpoint_success")
	if err != nil {
		t.Errorf("Expected no errors but got: %v", err)
	}
	dnsEndpoint := &endpoint.Endpoint{
		Name: "dns-ep-name", Group: "dns-ep-group", URL: "8.8.8.8", DNSConfig: &dns.Config{
			QueryType: "A",
			QueryName: "example.com.",
		},
	}
	PublishMetricsForEndpoint(dnsEndpoint, &endpoint.Result{
		DNSRCode:  "NOERROR",
		Connected: true,
		Duration:  50 * time.Millisecond,
		ConditionResults: []*endpoint.ConditionResult{
			{Condition: "[DNS_RCODE] == NOERROR", Success: true},
		},
		Success: true,
	}, []string{})
	err = testutil.GatherAndCompare(reg, bytes.NewBufferString(`
# HELP gatus_results_code_total Total number of results by code
# TYPE gatus_results_code_total counter
gatus_results_code_total{code="200",group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 2
gatus_results_code_total{code="NOERROR",group="dns-ep-group",key="dns-ep-group_dns-ep-name",name="dns-ep-name",type="DNS"} 1
# HELP gatus_results_connected_total Total number of results in which a connection was successfully established
# TYPE gatus_results_connected_total counter
gatus_results_connected_total{group="dns-ep-group",key="dns-ep-group_dns-ep-name",name="dns-ep-name",type="DNS"} 1
gatus_results_connected_total{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 2
# HELP gatus_results_duration_seconds Duration of the request in seconds
# TYPE gatus_results_duration_seconds gauge
gatus_results_duration_seconds{group="dns-ep-group",key="dns-ep-group_dns-ep-name",name="dns-ep-name",type="DNS"} 0.05
gatus_results_duration_seconds{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 0.125
# HELP gatus_results_total Number of results per endpoint
# TYPE gatus_results_total counter
gatus_results_total{group="dns-ep-group",key="dns-ep-group_dns-ep-name",name="dns-ep-name",success="true",type="DNS"} 1
gatus_results_total{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",success="false",type="HTTP"} 1
gatus_results_total{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",success="true",type="HTTP"} 1
# HELP gatus_results_certificate_expiration_seconds Number of seconds until the certificate expires
# TYPE gatus_results_certificate_expiration_seconds gauge
gatus_results_certificate_expiration_seconds{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 169200
# HELP gatus_results_endpoint_success Displays whether or not the endpoint was a success
# TYPE gatus_results_endpoint_success gauge
gatus_results_endpoint_success{group="dns-ep-group",key="dns-ep-group_dns-ep-name",name="dns-ep-name",type="DNS"} 1
gatus_results_endpoint_success{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 0
`), "gatus_results_code_total", "gatus_results_connected_total", "gatus_results_duration_seconds", "gatus_results_total", "gatus_results_certificate_expiration_seconds", "gatus_results_endpoint_success")
	if err != nil {
		t.Errorf("Expected no errors but got: %v", err)
	}
}

func TestPublishMetricsForSuite(t *testing.T) {
	reg := prometheus.NewRegistry()
	InitializePrometheusMetrics(&config.Config{}, reg)

	testSuite := &suite.Suite{
		Name:  "test-suite",
		Group: "test-group",
	}
	// Test successful suite execution
	successResult := &suite.Result{
		Success:  true,
		Duration: 5 * time.Second,
		Name:     "test-suite",
		Group:    "test-group",
	}
	PublishMetricsForSuite(testSuite, successResult, []string{})

	err := testutil.GatherAndCompare(reg, bytes.NewBufferString(`
# HELP gatus_suite_results_duration_seconds Duration of suite execution in seconds
# TYPE gatus_suite_results_duration_seconds gauge
gatus_suite_results_duration_seconds{group="test-group",key="test-group_test-suite",name="test-suite"} 5
# HELP gatus_suite_results_success Whether the suite execution was successful (1) or not (0)
# TYPE gatus_suite_results_success gauge
gatus_suite_results_success{group="test-group",key="test-group_test-suite",name="test-suite"} 1
# HELP gatus_suite_results_total Total number of suite executions
# TYPE gatus_suite_results_total counter
gatus_suite_results_total{group="test-group",key="test-group_test-suite",name="test-suite",success="true"} 1
`), "gatus_suite_results_duration_seconds", "gatus_suite_results_success", "gatus_suite_results_total")
	if err != nil {
		t.Errorf("Expected no errors but got: %v", err)
	}

	// Test failed suite execution
	failureResult := &suite.Result{
		Success:  false,
		Duration: 10 * time.Second,
		Name:     "test-suite",
		Group:    "test-group",
	}
	PublishMetricsForSuite(testSuite, failureResult, []string{})

	err = testutil.GatherAndCompare(reg, bytes.NewBufferString(`
# HELP gatus_suite_results_duration_seconds Duration of suite execution in seconds
# TYPE gatus_suite_results_duration_seconds gauge
gatus_suite_results_duration_seconds{group="test-group",key="test-group_test-suite",name="test-suite"} 10
# HELP gatus_suite_results_success Whether the suite execution was successful (1) or not (0)
# TYPE gatus_suite_results_success gauge
gatus_suite_results_success{group="test-group",key="test-group_test-suite",name="test-suite"} 0
# HELP gatus_suite_results_total Total number of suite executions
# TYPE gatus_suite_results_total counter
gatus_suite_results_total{group="test-group",key="test-group_test-suite",name="test-suite",success="false"} 1
gatus_suite_results_total{group="test-group",key="test-group_test-suite",name="test-suite",success="true"} 1
`), "gatus_suite_results_duration_seconds", "gatus_suite_results_success", "gatus_suite_results_total")
	if err != nil {
		t.Errorf("Expected no errors but got: %v", err)
	}
}

func TestPublishMetricsForSuite_NoGroup(t *testing.T) {
	reg := prometheus.NewRegistry()
	InitializePrometheusMetrics(&config.Config{}, reg)

	testSuite := &suite.Suite{
		Name:  "no-group-suite",
		Group: "",
	}
	result := &suite.Result{
		Success:  true,
		Duration: 3 * time.Second,
		Name:     "no-group-suite",
		Group:    "",
	}
	PublishMetricsForSuite(testSuite, result, []string{})

	err := testutil.GatherAndCompare(reg, bytes.NewBufferString(`
# HELP gatus_suite_results_duration_seconds Duration of suite execution in seconds
# TYPE gatus_suite_results_duration_seconds gauge
gatus_suite_results_duration_seconds{group="",key="_no-group-suite",name="no-group-suite"} 3
# HELP gatus_suite_results_success Whether the suite execution was successful (1) or not (0)
# TYPE gatus_suite_results_success gauge
gatus_suite_results_success{group="",key="_no-group-suite",name="no-group-suite"} 1
# HELP gatus_suite_results_total Total number of suite executions
# TYPE gatus_suite_results_total counter
gatus_suite_results_total{group="",key="_no-group-suite",name="no-group-suite",success="true"} 1
`), "gatus_suite_results_duration_seconds", "gatus_suite_results_success", "gatus_suite_results_total")
	if err != nil {
		t.Errorf("Expected no errors but got: %v", err)
	}
}

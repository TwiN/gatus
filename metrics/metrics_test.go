package metrics

import (
	"bytes"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/dns"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestPublishMetricsForEndpoint(t *testing.T) {
	httpEndpoint := &endpoint.Endpoint{Name: "http-ep-name", Group: "http-ep-group", URL: "https://example.org"}
	PublishMetricsForEndpoint(httpEndpoint, &endpoint.Result{
		HTTPStatus: 200,
		Connected:  true,
		Duration:   123 * time.Millisecond,
		ConditionResults: []*endpoint.ConditionResult{
			{Condition: "[STATUS] == 200", Success: true},
			{Condition: "[CERTIFICATE_EXPIRATION] > 48h", Success: true},
		},
		Success:               true,
		CertificateExpiration: 49 * time.Hour,
	})
	err := testutil.GatherAndCompare(prometheus.Gatherers{prometheus.DefaultGatherer}, bytes.NewBufferString(`
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
		},
		Success:               false,
		CertificateExpiration: 47 * time.Hour,
	})
	err = testutil.GatherAndCompare(prometheus.Gatherers{prometheus.DefaultGatherer}, bytes.NewBufferString(`
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
# HELP gatus_results_endpoint_success Displays whether or not the endpoint was a success
# TYPE gatus_results_endpoint_success gauge
gatus_results_endpoint_success{group="http-ep-group",key="http-ep-group_http-ep-name",name="http-ep-name",type="HTTP"} 0
`), "gatus_results_code_total", "gatus_results_connected_total", "gatus_results_duration_seconds", "gatus_results_total", "gatus_results_certificate_expiration_seconds", "gatus_results_endpoint_success")
	if err != nil {
		t.Errorf("Expected no errors but got: %v", err)
	}
	dnsEndpoint := &endpoint.Endpoint{Name: "dns-ep-name", Group: "dns-ep-group", URL: "8.8.8.8", DNSConfig: &dns.Config{
		QueryType: "A",
		QueryName: "example.com.",
	}}
	PublishMetricsForEndpoint(dnsEndpoint, &endpoint.Result{
		DNSRCode:  "NOERROR",
		Connected: true,
		Duration:  50 * time.Millisecond,
		ConditionResults: []*endpoint.ConditionResult{
			{Condition: "[DNS_RCODE] == NOERROR", Success: true},
		},
		Success: true,
	})
	err = testutil.GatherAndCompare(prometheus.Gatherers{prometheus.DefaultGatherer}, bytes.NewBufferString(`
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

package placeholder

import (
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestReplace(t *testing.T) {
	t.Parallel()
	description := "healthcheck failed"
	nestedDescription := "[ENDPOINT_NAME]"
	scenarios := []struct {
		name     string
		template string
		endpoint endpoint.Endpoint
		alert    alert.Alert
		resolved bool
		want     string
	}{
		{
			name:     "triggered",
			template: "[ALERT_TRIGGERED_OR_RESOLVED] [ENDPOINT_GROUP]/[ENDPOINT_NAME]: [ALERT_DESCRIPTION]",
			endpoint: endpoint.Endpoint{Name: "website", Group: "external"},
			alert:    alert.Alert{Description: &description},
			want:     "TRIGGERED external/website: healthcheck failed",
		},
		{
			name:     "endpoint-with-group",
			template: "[ENDPOINT] [ENDPOINT]",
			endpoint: endpoint.Endpoint{Name: "website", Group: "external"},
			want:     "external/website external/website",
		},
		{
			name:     "endpoint-without-group",
			template: "[ENDPOINT]",
			endpoint: endpoint.Endpoint{Name: "website"},
			want:     "website",
		},
		{
			name:     "resolved",
			template: "[ALERT_TRIGGERED_OR_RESOLVED] [ENDPOINT_NAME]",
			endpoint: endpoint.Endpoint{Name: "website"},
			resolved: true,
			want:     "RESOLVED website",
		},
		{
			name:     "empty-values-and-unsupported-result-errors",
			template: "[ENDPOINT_NAME]|[ENDPOINT_GROUP]|[ALERT_DESCRIPTION]|[RESULT_ERRORS]",
			endpoint: endpoint.Endpoint{Name: "website"},
			want:     "website|||[RESULT_ERRORS]",
		},
		{
			name:     "unknown-placeholder-and-static-text",
			template: "<@123> [UNKNOWN] []",
			want:     "<@123> [UNKNOWN] []",
		},
		{
			name:     "values-are-not-expanded",
			template: "[ALERT_DESCRIPTION]",
			endpoint: endpoint.Endpoint{Name: "website"},
			alert:    alert.Alert{Description: &nestedDescription},
			want:     "[ENDPOINT_NAME]",
		},
		{
			name:     "empty-template",
			endpoint: endpoint.Endpoint{Name: "website"},
			want:     "",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Parallel()
			got := Replace(scenario.template, &scenario.endpoint, &scenario.alert, scenario.resolved)
			if got != scenario.want {
				t.Errorf("expected %q, got %q", scenario.want, got)
			}
		})
	}
}

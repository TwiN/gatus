package pagerduty

import (
	"fmt"
	"net/http"

	"github.com/TwinProduction/gatus/alerting/alert"
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/core"
)

const (
	restAPIURL = "https://events.pagerduty.com/v2/enqueue"
)

// AlertProvider is the configuration necessary for sending an alert using PagerDuty
type AlertProvider struct {
	IntegrationKey string `yaml:"integration-key"`

	// DefaultAlert is the default alert configuration to use for services with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.IntegrationKey) == 32
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
//
// relevant: https://developer.pagerduty.com/docs/events-api-v2/trigger-events/
func (provider *AlertProvider) ToCustomAlertProvider(service *core.Service, alert *alert.Alert, _ *core.Result, resolved bool) *custom.AlertProvider {
	var message, eventAction, resolveKey string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", service.Name, alert.GetDescription())
		eventAction = "resolve"
		resolveKey = alert.ResolveKey
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", service.Name, alert.GetDescription())
		eventAction = "trigger"
		resolveKey = ""
	}
	return &custom.AlertProvider{
		URL:    restAPIURL,
		Method: http.MethodPost,
		Body: fmt.Sprintf(`{
  "routing_key": "%s",
  "dedup_key": "%s",
  "event_action": "%s",
  "payload": {
    "summary": "%s",
    "source": "%s",
    "severity": "critical"
  }
}`, provider.IntegrationKey, resolveKey, eventAction, message, service.Name),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

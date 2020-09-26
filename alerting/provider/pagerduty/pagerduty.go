package pagerduty

import (
	"fmt"
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/core"
)

type AlertProvider struct {
	IntegrationKey string `yaml:"integration-key"`
}

func (provider *AlertProvider) IsValid() bool {
	return len(provider.IntegrationKey) == 32
}

// https://developer.pagerduty.com/docs/events-api-v2/trigger-events/
func (provider *AlertProvider) ToCustomAlertProvider(service *core.Service, alert *core.Alert, result *core.Result, resolved bool) *custom.AlertProvider {
	var message, eventAction, resolveKey string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", service.Name, alert.Description)
		eventAction = "resolve"
		resolveKey = alert.ResolveKey
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", service.Name, alert.Description)
		eventAction = "trigger"
		resolveKey = ""
	}
	return &custom.AlertProvider{
		Url:    "https://events.pagerduty.com/v2/enqueue",
		Method: "POST",
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

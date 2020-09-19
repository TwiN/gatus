package alerting

import (
	"fmt"
	"github.com/TwinProduction/gatus/core"
)

type PagerDutyAlertProvider struct {
	IntegrationKey string `yaml:"integration-key"`
}

func (provider *PagerDutyAlertProvider) IsValid() bool {
	return len(provider.IntegrationKey) == 32
}

// https://developer.pagerduty.com/docs/events-api-v2/trigger-events/
func (provider *PagerDutyAlertProvider) ToCustomAlertProvider(eventAction, resolveKey string, service *core.Service, message string) *CustomAlertProvider {
	return &CustomAlertProvider{
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

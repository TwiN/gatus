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

type Integrations struct {
	IntegrationKey string `yaml:"integration-key"`
	Group          string `yaml:"group"`
}

// AlertProvider is the configuration necessary for sending an alert using PagerDuty
type AlertProvider struct {
	IntegrationKey string `yaml:"integration-key"`

	// DefaultAlert is the default alert configuration to use for services with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert"`

	Integrations []Integrations `yaml:"integrations"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	registeredGroups := make(map[string]bool)
	if provider.Integrations != nil {
		for _, integration := range provider.Integrations {
			if isAlreadyRegistered := registeredGroups[integration.Group]; isAlreadyRegistered || integration.Group == "" || len(integration.IntegrationKey) != 32 {
				return false
			}
			registeredGroups[integration.Group] = true
		}
	}
	return len(provider.IntegrationKey) == 32 || provider.Integrations != nil
}

// GetPagerDutyIntegrationKey returns the appropriate pagerduty integration key
func (provider *AlertProvider) GetPagerDutyIntegrationKey(group string) string {
	if provider.Integrations != nil {
		for _, integration := range provider.Integrations {
			if group == integration.Group {
				return integration.IntegrationKey
			}
		}
	}
	if provider.IntegrationKey != "" {
		return provider.IntegrationKey
	}
	return ""
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
}`, provider.GetPagerDutyIntegrationKey(service.Group), resolveKey, eventAction, message, service.Name),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

package pagerduty

import (
	"fmt"
	"net/http"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/alerting/provider/custom"
	"github.com/TwiN/gatus/v3/core"
)

const (
	restAPIURL = "https://events.pagerduty.com/v2/enqueue"
)

// AlertProvider is the configuration necessary for sending an alert using PagerDuty
type AlertProvider struct {
	IntegrationKey string `yaml:"integration-key"`

	// DefaultAlert is the default alert configuration to use for services with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides"`
}

// Override is a case under which the default integration is overridden
type Override struct {
	Group          string `yaml:"group"`
	IntegrationKey string `yaml:"integration-key"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	registeredGroups := make(map[string]bool)
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" || len(override.IntegrationKey) != 32 {
				return false
			}
			registeredGroups[override.Group] = true
		}
	}
	// Either the default integration key has the right length, or there are overrides who are properly configured.
	return len(provider.IntegrationKey) == 32 || len(provider.Overrides) != 0
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
}`, provider.getPagerDutyIntegrationKeyForGroup(service.Group), resolveKey, eventAction, message, service.Name),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}
}

// getPagerDutyIntegrationKeyForGroup returns the appropriate pagerduty integration key for a given group
func (provider *AlertProvider) getPagerDutyIntegrationKeyForGroup(group string) string {
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				return override.IntegrationKey
			}
		}
	}
	if provider.IntegrationKey != "" {
		return provider.IntegrationKey
	}
	return ""
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

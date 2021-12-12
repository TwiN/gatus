package pagerduty

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/client"
	"github.com/TwiN/gatus/v3/core"
)

const (
	restAPIURL = "https://events.pagerduty.com/v2/enqueue"
)

// AlertProvider is the configuration necessary for sending an alert using PagerDuty
type AlertProvider struct {
	IntegrationKey string `yaml:"integration-key"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
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

// Send an alert using the provider
//
// Relevant: https://developer.pagerduty.com/docs/events-api-v2/trigger-events/
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer([]byte(provider.buildRequestBody(endpoint, alert, result, resolved)))
	request, err := http.NewRequest(http.MethodPost, restAPIURL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	if alert.IsSendingOnResolved() {
		if resolved {
			// The alert has been resolved and there's no error, so we can clear the alert's ResolveKey
			alert.ResolveKey = ""
		} else {
			// We need to retrieve the resolve key from the response
			body, err := io.ReadAll(response.Body)
			var payload pagerDutyResponsePayload
			if err = json.Unmarshal(body, &payload); err != nil {
				// Silently fail. We don't want to create tons of alerts just because we failed to parse the body.
				log.Printf("[pagerduty][Send] Ran into error unmarshaling pagerduty response: %s", err.Error())
			} else {
				alert.ResolveKey = payload.DedupKey
			}
		}
	}
	return nil
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) string {
	var message, eventAction, resolveKey string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", endpoint.DisplayName(), alert.GetDescription())
		eventAction = "resolve"
		resolveKey = alert.ResolveKey
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", endpoint.DisplayName(), alert.GetDescription())
		eventAction = "trigger"
		resolveKey = ""
	}
	return fmt.Sprintf(`{
  "routing_key": "%s",
  "dedup_key": "%s",
  "event_action": "%s",
  "payload": {
    "summary": "%s",
    "source": "%s",
    "severity": "critical"
  }
}`, provider.getIntegrationKeyForGroup(endpoint.Group), resolveKey, eventAction, message, endpoint.Name)
}

// getIntegrationKeyForGroup returns the appropriate pagerduty integration key for a given group
func (provider *AlertProvider) getIntegrationKeyForGroup(group string) string {
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				return override.IntegrationKey
			}
		}
	}
	return provider.IntegrationKey
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

type pagerDutyResponsePayload struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	DedupKey string `json:"dedup_key"`
}

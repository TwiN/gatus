package teams

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/core"
)

// AlertProvider is the configuration necessary for sending an alert using Teams
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

// Override is a case under which the default integration is overridden
type Override struct {
	Group      string `yaml:"group"`
	WebhookURL string `yaml:"webhook-url"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	registeredGroups := make(map[string]bool)
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" || len(override.WebhookURL) == 0 {
				return false
			}
			registeredGroups[override.Group] = true
		}
	}
	return len(provider.WebhookURL) > 0
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildRequestBody(endpoint, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, provider.getWebhookURLForGroup(endpoint.Group), buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return err
}

type Body struct {
	Type       string    `json:"@type"`
	Context    string    `json:"@context"`
	ThemeColor string    `json:"themeColor"`
	Title      string    `json:"title"`
	Text       string    `json:"text"`
	Sections   []Section `json:"sections"`
}

type Section struct {
	ActivityTitle string `json:"activityTitle"`
	Text          string `json:"text"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) []byte {
	var message, color string
	if resolved {
		message = fmt.Sprintf("An alert for *%s* has been resolved after passing successfully %d time(s) in a row", endpoint.DisplayName(), alert.SuccessThreshold)
		color = "#36A64F"
	} else {
		message = fmt.Sprintf("An alert for *%s* has been triggered due to having failed %d time(s) in a row", endpoint.DisplayName(), alert.FailureThreshold)
		color = "#DD0000"
	}
	var results string
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "&#x2705;"
		} else {
			prefix = "&#x274C;"
		}
		results += fmt.Sprintf("%s - `%s`<br/>", prefix, conditionResult.Condition)
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = ": " + alertDescription
	}
	body, _ := json.Marshal(Body{
		Type:       "MessageCard",
		Context:    "http://schema.org/extensions",
		ThemeColor: color,
		Title:      "&#x1F6A8; Gatus",
		Text:       message + description,
		Sections: []Section{
			{
				ActivityTitle: "Condition results",
				Text:          results,
			},
		},
	})
	return body
}

// getWebhookURLForGroup returns the appropriate Webhook URL integration to for a given group
func (provider *AlertProvider) getWebhookURLForGroup(group string) string {
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				return override.WebhookURL
			}
		}
	}
	return provider.WebhookURL
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

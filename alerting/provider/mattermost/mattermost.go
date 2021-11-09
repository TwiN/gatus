package mattermost

import (
	"fmt"
	"net/http"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/alerting/provider/custom"
	"github.com/TwiN/gatus/v3/client"
	"github.com/TwiN/gatus/v3/core"
)

// AlertProvider is the configuration necessary for sending an alert using Mattermost
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"`

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if provider.ClientConfig == nil {
		provider.ClientConfig = client.GetDefaultConfig()
	}
	return len(provider.WebhookURL) > 0
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
func (provider *AlertProvider) ToCustomAlertProvider(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) *custom.AlertProvider {
	var message string
	var color string
	if resolved {
		message = fmt.Sprintf("An alert for *%s* has been resolved after passing successfully %d time(s) in a row", endpoint.Name, alert.SuccessThreshold)
		color = "#36A64F"
	} else {
		message = fmt.Sprintf("An alert for *%s* has been triggered due to having failed %d time(s) in a row", endpoint.Name, alert.FailureThreshold)
		color = "#DD0000"
	}
	var results string
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = ":white_check_mark:"
		} else {
			prefix = ":x:"
		}
		results += fmt.Sprintf("%s - `%s`\\n", prefix, conditionResult.Condition)
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = ":\\n> " + alertDescription
	}
	return &custom.AlertProvider{
		URL:          provider.WebhookURL,
		Method:       http.MethodPost,
		ClientConfig: provider.ClientConfig,
		Body: fmt.Sprintf(`{
  "text": "",
  "username": "gatus",
  "icon_url": "https://raw.githubusercontent.com/TwiN/gatus/master/.github/assets/logo.png",
  "attachments": [
    {
      "title": ":rescue_worker_helmet: Gatus",
      "fallback": "Gatus - %s",
      "text": "%s%s",
      "short": false,
      "color": "%s",
      "fields": [
        {
          "title": "URL",
          "value": "%s",
          "short": false
        },
        {
          "title": "Condition results",
          "value": "%s",
          "short": false
        }
      ]
    }
  ]
}`, message, message, description, color, endpoint.URL, results),
		Headers: map[string]string{"Content-Type": "application/json"},
	}
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

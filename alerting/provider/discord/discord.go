package discord

import (
	"fmt"
	"net/http"

	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/core"
)

// AlertProvider is the configuration necessary for sending an alert using Discord
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"`

	// DefaultAlert is the default alert configuration to use for services with an alert of the appropriate type
	DefaultAlert *core.Alert `yaml:"default-alert"`
}

//func (provider *AlertProvider) ParseWithDefaultAlert(alert *core.Alert) {
//	if provider.DefaultAlert == nil {
//		return
//	}
//	if alert.Enabled == nil {
//		alert.Enabled = provider.DefaultAlert.Enabled
//	}
//	if alert.SendOnResolved == nil {
//		alert.SendOnResolved = provider.DefaultAlert.SendOnResolved
//	}
//	if len(alert.Description) == 0 {
//		alert.Description = provider.DefaultAlert.Description
//	}
//	if alert.FailureThreshold == 0 {
//		alert.FailureThreshold = provider.DefaultAlert.FailureThreshold
//	}
//	if alert.SuccessThreshold == 0 {
//		alert.SuccessThreshold = provider.DefaultAlert.SuccessThreshold
//	}
//}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.WebhookURL) > 0
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
func (provider *AlertProvider) ToCustomAlertProvider(service *core.Service, alert *core.Alert, result *core.Result, resolved bool) *custom.AlertProvider {
	var message, results string
	var colorCode int
	if resolved {
		message = fmt.Sprintf("An alert for **%s** has been resolved after passing successfully %d time(s) in a row", service.Name, alert.SuccessThreshold)
		colorCode = 3066993
	} else {
		message = fmt.Sprintf("An alert for **%s** has been triggered due to having failed %d time(s) in a row", service.Name, alert.FailureThreshold)
		colorCode = 15158332
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = ":white_check_mark:"
		} else {
			prefix = ":x:"
		}
		results += fmt.Sprintf("%s - `%s`\\n", prefix, conditionResult.Condition)
	}
	return &custom.AlertProvider{
		URL:    provider.WebhookURL,
		Method: http.MethodPost,
		Body: fmt.Sprintf(`{
  "content": "",
  "embeds": [
    {
      "title": ":helmet_with_white_cross: Gatus",
      "description": "%s:\n> %s",
      "color": %d,
      "fields": [
        {
          "name": "Condition results",
          "value": "%s",
          "inline": false
        }
      ]
    }
  ]
}`, message, alert.GetDescription(), colorCode, results),
		Headers: map[string]string{"Content-Type": "application/json"},
	}
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *core.Alert {
	return provider.DefaultAlert
}

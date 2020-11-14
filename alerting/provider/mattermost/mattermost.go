package mattermost

import (
	"fmt"
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/core"
)

// AlertProvider is the configuration necessary for sending an alert using Mattermost
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"`
	Insecure   bool   `yaml:"insecure,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.WebhookURL) > 0
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
func (provider *AlertProvider) ToCustomAlertProvider(service *core.Service, alert *core.Alert, result *core.Result, resolved bool) *custom.AlertProvider {
	var message string
	var color string
	if resolved {
		message = fmt.Sprintf("An alert for *%s* has been resolved after passing successfully %d time(s) in a row", service.Name, alert.SuccessThreshold)
		color = "#36A64F"
	} else {
		message = fmt.Sprintf("An alert for *%s* has been triggered due to having failed %d time(s) in a row", service.Name, alert.FailureThreshold)
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
		results += fmt.Sprintf("%s - `%s`\n", prefix, conditionResult.Condition)
	}
	return &custom.AlertProvider{
		URL:    provider.WebhookURL,
		Method: "POST",
		Insecure: provider.Insecure,
		Body: fmt.Sprintf(`{
			"text": "",
			"username": "gatus",
			"icon_url": "https://raw.githubusercontent.com/TwinProduction/gatus/master/static/logo.png",
			"attachments": [
				{
					"title": ":rescue_worker_helmet: Gatus",
					"fallback": "Gatus - %s",
					"text": "%s:\n> %s",
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
		}`, message, message, alert.Description, color, service.URL, results),
		Headers: map[string]string{"Content-Type": "application/json"},
	}
}

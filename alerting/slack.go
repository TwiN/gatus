package alerting

import (
	"fmt"
	"github.com/TwinProduction/gatus/core"
)

type SlackAlertProvider struct {
	WebhookUrl string `yaml:"webhook-url"`
}

func (provider *SlackAlertProvider) IsValid() bool {
	return len(provider.WebhookUrl) > 0
}

func (provider *SlackAlertProvider) ToCustomAlertProvider(service *core.Service, alert *core.Alert, result *core.Result, resolved bool) *CustomAlertProvider {
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
			prefix = ":heavy_check_mark:"
		} else {
			prefix = ":x:"
		}
		results += fmt.Sprintf("%s - `%s`\n", prefix, conditionResult.Condition)
	}
	return &CustomAlertProvider{
		Url:    provider.WebhookUrl,
		Method: "POST",
		Body: fmt.Sprintf(`{
  "text": "",
  "attachments": [
    {
      "title": ":helmet_with_white_cross: Gatus",
      "text": "%s:\n> %s",
      "short": false,
      "color": "%s",
      "fields": [
        {
          "title": "Condition results",
          "value": "%s",
          "short": false
        }
      ]
    },
  ]
}`, message, alert.Description, color, results),
		Headers: map[string]string{"Content-Type": "application/json"},
	}
}

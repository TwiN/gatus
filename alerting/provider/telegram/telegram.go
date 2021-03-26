package telegram

import (
	"fmt"
	"net/http"

	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/core"
)

// AlertProvider is the configuration necessary for sending an alert using Telegram
type AlertProvider struct {
	Token string `yaml:"token"`
	ID    int32  `yaml:"id"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.Token) > 0 && provider.ID != 0
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
func (provider *AlertProvider) ToCustomAlertProvider(service *core.Service, alert *core.Alert, result *core.Result, resolved bool) *custom.AlertProvider {
	var message, results string
	if resolved {
		message = fmt.Sprintf("An alert for **%s** has been resolved after passing successfully %d time(s) in a row", service.Name, alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for **%s** has been triggered due to having failed %d time(s) in a row", service.Name, alert.FailureThreshold)
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
	var text string = fmt.Sprintf(`
	:helmet_with_white_cross: Gatus
	%s
	`, message)
	return &custom.AlertProvider{
		URL:    fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", provider.Token),
		Method: http.MethodPost,
		Body: fmt.Sprintf(`{
  "chat_id": "%d",
  "text": "%s"
  "embeds": [
    {
      "title": ":helmet_with_white_cross: Gatus",
      "description": "%s:\n> %s",
      "fields": [
        {
          "name": "Condition results",
          "value": "%s",
          "inline": false
        }
      ]
    }
  ]
}`, provider.ID, text, message, alert.Description, results), // need to change body
		Headers: map[string]string{"Content-Type": "application/json"},
	}
}

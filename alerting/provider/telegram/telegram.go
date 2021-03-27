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
	ID    string `yaml:"id"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.Token) > 0 && len(provider.ID) > 0
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
func (provider *AlertProvider) ToCustomAlertProvider(service *core.Service, alert *core.Alert, result *core.Result, resolved bool) *custom.AlertProvider {
	var message, results string
	if resolved {
		message = fmt.Sprintf("An alert for <b>%s</b> has been resolved:\\n—\\n<i>    healthcheck passing successfully %d time(s) in a row</i>\\n—  ", service.Name, alert.FailureThreshold)
	} else {
		message = fmt.Sprintf("An alert for <b>%s</b> has been triggered:\\n—\\n<i>    healthcheck failed %d time(s) in a row</i>\\n—  ", service.Name, alert.FailureThreshold)
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "✅"
		} else {
			prefix = "❌"
		}
		results += fmt.Sprintf("%s - <code>%s</code>\\n", prefix, conditionResult.Condition)
	}
	var text string = fmt.Sprintf("⛑ <b>Gatus</b> \\n%s \\n<b>Condition results</b>\\n%s", message, results)
	return &custom.AlertProvider{
		URL:     fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", provider.Token),
		Method:  http.MethodPost,
		Body:    fmt.Sprintf(`{"chat_id": "%s", "text": "%s", "parse_mode": "HTML" }`, provider.ID, text),
		Headers: map[string]string{"Content-Type": "application/json"},
	}
}

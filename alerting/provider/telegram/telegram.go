package telegram

import (
	"fmt"
	"net/http"

	"github.com/Meldiron/gatus/alerting/provider/custom"
	"github.com/Meldiron/gatus/core"
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
		message = fmt.Sprintf("An alert for *%s* has been resolved:\\n—\\n    _healthcheck passing successfully %d time(s) in a row_\\n—  ", service.Name, alert.FailureThreshold)
	} else {
		message = fmt.Sprintf("An alert for *%s* has been triggered:\\n—\\n    _healthcheck failed %d time(s) in a row_\\n—  ", service.Name, alert.FailureThreshold)
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "✅"
		} else {
			prefix = "❌"
		}
		results += fmt.Sprintf("%s - `%s`\\n", prefix, conditionResult.Condition)
	}
	var text string
	if len(alert.Description) > 0 {
		text = fmt.Sprintf("⛑ *Gatus* \\n%s \\n*Description* \\n_%s_  \\n\\n*Condition results*\\n%s", message, alert.Description, results)
	} else {
		text = fmt.Sprintf("⛑ *Gatus* \\n%s \\n*Condition results*\\n%s", message, results)
	}
	return &custom.AlertProvider{
		URL:     fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", provider.Token),
		Method:  http.MethodPost,
		Body:    fmt.Sprintf(`{"chat_id": "%s", "text": "%s", "parse_mode": "MARKDOWN"}`, provider.ID, text),
		Headers: map[string]string{"Content-Type": "application/json"},
	}
}

package messagebird

import (
	"fmt"
	"net/http"

	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/core"
)

const (
	restAPIURL = "https://rest.messagebird.com/messages"
)

// AlertProvider is the configuration necessary for sending an alert using Messagebird
type AlertProvider struct {
	AccessKey string `yaml:"access-key"`
	From      string `yaml:"from"`
	To        string `yaml:"to"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.AccessKey) > 0 && len(provider.From) > 0 && len(provider.To) > 0
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
// Reference doc for messagebird https://developers.messagebird.com/api/sms-messaging/#send-outbound-sms
func (provider *AlertProvider) ToCustomAlertProvider(service *core.Service, alert *core.Alert, _ *core.Result, resolved bool) *custom.AlertProvider {
	var message string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", service.Name, alert.Description)
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", service.Name, alert.Description)
	}

	return &custom.AlertProvider{
		URL:    restAPIURL,
		Method: http.MethodPost,
		Body: fmt.Sprintf(`{
  "originator": "%s",
  "recipients": "%s",
  "body": "%s"
}`, provider.From, provider.To, message),
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("AccessKey %s", provider.AccessKey),
		},
	}
}

package messagebird

import (
	"fmt"
	"net/http"

	"github.com/TwinProduction/gatus/alerting/alert"
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/core"
)

const (
	restAPIURL = "https://rest.messagebird.com/messages"
)

// AlertProvider is the configuration necessary for sending an alert using Messagebird
type AlertProvider struct {
	AccessKey  string `yaml:"access-key"`
	Originator string `yaml:"originator"`
	Recipients string `yaml:"recipients"`

	// DefaultAlert is the default alert configuration to use for services with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.AccessKey) > 0 && len(provider.Originator) > 0 && len(provider.Recipients) > 0
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
// Reference doc for messagebird https://developers.messagebird.com/api/sms-messaging/#send-outbound-sms
func (provider *AlertProvider) ToCustomAlertProvider(service *core.Service, alert *alert.Alert, _ *core.Result, resolved bool) *custom.AlertProvider {
	var message string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", service.Name, alert.GetDescription())
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", service.Name, alert.GetDescription())
	}
	return &custom.AlertProvider{
		URL:    restAPIURL,
		Method: http.MethodPost,
		Body: fmt.Sprintf(`{
  "originator": "%s",
  "recipients": "%s",
  "body": "%s"
}`, provider.Originator, provider.Recipients, message),
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": fmt.Sprintf("AccessKey %s", provider.AccessKey),
		},
	}
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

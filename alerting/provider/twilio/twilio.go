package twilio

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/alerting/provider/custom"
	"github.com/TwiN/gatus/v3/core"
)

// AlertProvider is the configuration necessary for sending an alert using Twilio
type AlertProvider struct {
	SID   string `yaml:"sid"`
	Token string `yaml:"token"`
	From  string `yaml:"from"`
	To    string `yaml:"to"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.Token) > 0 && len(provider.SID) > 0 && len(provider.From) > 0 && len(provider.To) > 0
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
func (provider *AlertProvider) ToCustomAlertProvider(endpoint *core.Endpoint, alert *alert.Alert, _ *core.Result, resolved bool) *custom.AlertProvider {
	var message string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", endpoint.Name, alert.GetDescription())
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", endpoint.Name, alert.GetDescription())
	}
	return &custom.AlertProvider{
		URL:    fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", provider.SID),
		Method: http.MethodPost,
		Body: url.Values{
			"To":   {provider.To},
			"From": {provider.From},
			"Body": {message},
		}.Encode(),
		Headers: map[string]string{
			"Content-Type":  "application/x-www-form-urlencoded",
			"Authorization": fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", provider.SID, provider.Token)))),
		},
	}
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

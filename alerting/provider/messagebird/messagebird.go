package messagebird

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/client"
	"github.com/TwiN/gatus/v3/core"
)

const (
	restAPIURL = "https://rest.messagebird.com/messages"
)

// AlertProvider is the configuration necessary for sending an alert using Messagebird
type AlertProvider struct {
	AccessKey  string `yaml:"access-key"`
	Originator string `yaml:"originator"`
	Recipients string `yaml:"recipients"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.AccessKey) > 0 && len(provider.Originator) > 0 && len(provider.Recipients) > 0
}

// Send an alert using the provider
// Reference doc for messagebird: https://developers.messagebird.com/api/sms-messaging/#send-outbound-sms
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer([]byte(provider.buildRequestBody(endpoint, alert, result, resolved)))
	request, err := http.NewRequest(http.MethodPost, restAPIURL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("AccessKey %s", provider.AccessKey))
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return err
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) string {
	var message string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", endpoint.DisplayName(), alert.GetDescription())
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", endpoint.DisplayName(), alert.GetDescription())
	}
	return fmt.Sprintf(`{
  "originator": "%s",
  "recipients": "%s",
  "body": "%s"
}`, provider.Originator, provider.Recipients, message)
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

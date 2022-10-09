package ntfy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v4/alerting/alert"
	"github.com/TwiN/gatus/v4/client"
	"github.com/TwiN/gatus/v4/core"
)

const (
	DefaultURL      = "https://ntfy.sh"
	DefaultPriority = 3
)

// AlertProvider is the configuration necessary for sending an alert using Slack
type AlertProvider struct {
	Topic    string `yaml:"topic"`
	URL      string `yaml:"url,omitempty"`      // Defaults to DefaultURL
	Priority int    `yaml:"priority,omitempty"` // Defaults to DefaultPriority

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if len(provider.URL) == 0 {
		provider.URL = DefaultURL
	}
	if provider.Priority == 0 {
		provider.Priority = DefaultPriority
	}
	return len(provider.URL) > 0 && len(provider.Topic) > 0 && provider.Priority > 0 && provider.Priority < 6
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer([]byte(provider.buildRequestBody(endpoint, alert, result, resolved)))
	request, err := http.NewRequest(http.MethodPost, provider.URL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
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
	var message, tag string
	if len(alert.GetDescription()) > 0 {
		message = endpoint.DisplayName() + " - " + alert.GetDescription()
	} else {
		message = endpoint.DisplayName()
	}
	if resolved {
		tag = "white_check_mark"
	} else {
		tag = "x"
	}
	return fmt.Sprintf(`{
  "topic": "%s",
  "title": "Gatus",
  "message": "%s",
  "tags": ["%s"],
  "priority": %d
}`, provider.Topic, message, tag, provider.Priority)
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

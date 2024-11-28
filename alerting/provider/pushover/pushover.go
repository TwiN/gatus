package pushover

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

const (
	restAPIURL      = "https://api.pushover.net/1/messages.json"
	defaultPriority = 0
)

// AlertProvider is the configuration necessary for sending an alert using Pushover
type AlertProvider struct {
	// Key used to authenticate the application sending
	// See "Your Applications" on the dashboard, or add a new one: https://pushover.net/apps/build
	ApplicationToken string `yaml:"application-token"`

	// Key of the user or group the messages should be sent to
	UserKey string `yaml:"user-key"`

	// The title of your message, likely the application name
	// default: the name of your application in Pushover
	Title string `yaml:"title,omitempty"`

	// Priority of all messages, ranging from -2 (very low) to 2 (Emergency)
	// default: 0
	Priority int `yaml:"priority,omitempty"`

	// Priority of resolved messages, ranging from -2 (very low) to 2 (Emergency)
	// default: 0
	ResolvedPriority int `yaml:"resolved-priority,omitempty"`

	// Sound of the messages (see: https://pushover.net/api#sounds)
	// default: "" (pushover)
	Sound string `yaml:"sound,omitempty"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if provider.Priority == 0 {
		provider.Priority = defaultPriority
	}
	if provider.ResolvedPriority == 0 {
		provider.ResolvedPriority = defaultPriority
	}
	return len(provider.ApplicationToken) == 30 && len(provider.UserKey) == 30 && provider.Priority >= -2 && provider.Priority <= 2 && provider.ResolvedPriority >= -2 && provider.ResolvedPriority <= 2
}

// Send an alert using the provider
// Reference doc for pushover: https://pushover.net/api
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildRequestBody(ep, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, restAPIURL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return err
}

type Body struct {
	Token    string `json:"token"`
	User     string `json:"user"`
	Title    string `json:"title,omitempty"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
	Sound    string `json:"sound,omitempty"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var message string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", ep.DisplayName(), alert.GetDescription())
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", ep.DisplayName(), alert.GetDescription())
	}
	body, _ := json.Marshal(Body{
		Token:    provider.ApplicationToken,
		User:     provider.UserKey,
		Title:    provider.Title,
		Message:  message,
		Priority: provider.priority(resolved),
		Sound:    provider.Sound,
	})
	return body
}

func (provider *AlertProvider) priority(resolved bool) int {
	if resolved && provider.ResolvedPriority == 0 {
		return defaultPriority
	}
	if !resolved && provider.Priority == 0 {
		return defaultPriority
	}
	if resolved {
		return provider.ResolvedPriority
	}
	return provider.Priority
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

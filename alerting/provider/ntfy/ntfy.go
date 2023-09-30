package ntfy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/core"
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
	Token    string `yaml:"token,omitempty"`    // Defaults to ""

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
	isTokenValid := true
	if len(provider.Token) > 0 {
		isTokenValid = strings.HasPrefix(provider.Token, "tk_")
	}
	return len(provider.URL) > 0 && len(provider.Topic) > 0 && provider.Priority > 0 && provider.Priority < 6 && isTokenValid
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildRequestBody(endpoint, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, provider.URL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if len(provider.Token) > 0 {
		request.Header.Set("Authorization", "Bearer "+provider.Token)
	}
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
	Topic    string   `json:"topic"`
	Title    string   `json:"title"`
	Message  string   `json:"message"`
	Tags     []string `json:"tags"`
	Priority int      `json:"priority"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) []byte {
	var message, results, tag string
	if resolved {
		tag = "white_check_mark"
		message = "An alert has been resolved after passing successfully " + strconv.Itoa(alert.SuccessThreshold) + " time(s) in a row"
	} else {
		tag = "rotating_light"
		message = "An alert has been triggered due to having failed " + strconv.Itoa(alert.FailureThreshold) + " time(s) in a row"
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "🟢"
		} else {
			prefix = "🔴"
		}
		results += fmt.Sprintf("\n%s %s", prefix, conditionResult.Condition)
	}
	if len(alert.GetDescription()) > 0 {
		message += " with the following description: " + alert.GetDescription()
	}
	message += results
	body, _ := json.Marshal(Body{
		Topic:    provider.Topic,
		Title:    "Gatus: " + endpoint.DisplayName(),
		Message:  message,
		Tags:     []string{tag},
		Priority: provider.Priority,
	})
	return body
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

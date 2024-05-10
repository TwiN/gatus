package gotify

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

const DefaultPriority = 5

// AlertProvider is the configuration necessary for sending an alert using Gotify
type AlertProvider struct {
	// ServerURL is the URL of the Gotify server
	ServerURL string `yaml:"server-url"`

	// Token is the token to use when sending a message to the Gotify server
	Token string `yaml:"token"`

	// Priority is the priority of the message
	Priority int `yaml:"priority,omitempty"` // Defaults to DefaultPriority

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Title is the title of the message that will be sent
	Title string `yaml:"title,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if provider.Priority == 0 {
		provider.Priority = DefaultPriority
	}
	return len(provider.ServerURL) > 0 && len(provider.Token) > 0
}

// Send an alert using the provider
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildRequestBody(ep, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, provider.ServerURL+"/message?token="+provider.Token, buffer)
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
		return fmt.Errorf("failed to send alert to Gotify: %s", string(body))
	}
	return nil
}

type Body struct {
	Message  string `json:"message"`
	Title    string `json:"title"`
	Priority int    `json:"priority"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var message string
	if resolved {
		message = fmt.Sprintf("An alert for `%s` has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for `%s` has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	var formattedConditionResults string
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "✓"
		} else {
			prefix = "✕"
		}
		formattedConditionResults += fmt.Sprintf("\n%s - %s", prefix, conditionResult.Condition)
	}
	if len(alert.GetDescription()) > 0 {
		message += " with the following description: " + alert.GetDescription()
	}
	message += formattedConditionResults
	title := "Gatus: " + ep.DisplayName()
	if provider.Title != "" {
		title = provider.Title
	}
	bodyAsJSON, _ := json.Marshal(Body{
		Message:  message,
		Title:    title,
		Priority: provider.Priority,
	})
	return bodyAsJSON
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

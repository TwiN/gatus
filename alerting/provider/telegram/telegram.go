package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/core"
)

const defaultAPIURL = "https://api.telegram.org"

// AlertProvider is the configuration necessary for sending an alert using Telegram
type AlertProvider struct {
	Token  string `yaml:"token"`
	ID     string `yaml:"id"`
	APIURL string `yaml:"api-url"`

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client,omitempty"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if provider.ClientConfig == nil {
		provider.ClientConfig = client.GetDefaultConfig()
	}
	return len(provider.Token) > 0 && len(provider.ID) > 0
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildRequestBody(endpoint, alert, result, resolved))
	apiURL := provider.APIURL
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/bot%s/sendMessage", apiURL, provider.Token), buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(provider.ClientConfig).Do(request)
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
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) []byte {
	var message, results string
	if resolved {
		message = fmt.Sprintf("An alert for *%s* has been resolved:\n—\n    _healthcheck passing successfully %d time(s) in a row_\n—  ", endpoint.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for *%s* has been triggered:\n—\n    _healthcheck failed %d time(s) in a row_\n—  ", endpoint.DisplayName(), alert.FailureThreshold)
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "✅"
		} else {
			prefix = "❌"
		}
		results += fmt.Sprintf("%s - `%s`\n", prefix, conditionResult.Condition)
	}
	var text string
	if len(alert.GetDescription()) > 0 {
		text = fmt.Sprintf("⛑ *Gatus* \n%s \n*Description* \n_%s_  \n\n*Condition results*\n%s", message, alert.GetDescription(), results)
	} else {
		text = fmt.Sprintf("⛑ *Gatus* \n%s \n*Condition results*\n%s", message, results)
	}
	body, _ := json.Marshal(Body{
		ChatID:    provider.ID,
		Text:      text,
		ParseMode: "MARKDOWN",
	})
	return body
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

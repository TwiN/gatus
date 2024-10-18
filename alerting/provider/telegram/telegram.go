package telegram

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

	// Overrides is a list of Overrid that may be prioritized over the default configuration
	Overrides []*Override `yaml:"overrides,omitempty"`
}

// Override is a configuration that may be prioritized over the default configuration
type Override struct {
	group string `yaml:"group"`
	token string `yaml:"token"`
	id    string `yaml:"id"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if provider.ClientConfig == nil {
		provider.ClientConfig = client.GetDefaultConfig()
	}

	registerGroups := make(map[string]bool)
	for _, override := range provider.Overrides {
		if len(override.group) == 0 {
			return false
		}
		if _, ok := registerGroups[override.group]; ok {
			return false
		}
		registerGroups[override.group] = true
	}

	return len(provider.Token) > 0 && len(provider.ID) > 0
}

// Send an alert using the provider
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildRequestBody(ep, alert, result, resolved))
	apiURL := provider.APIURL
	if apiURL == "" {
		apiURL = defaultAPIURL
	}
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/bot%s/sendMessage", apiURL, provider.getTokenForGroup(ep.Group)), buffer)
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

func (provider *AlertProvider) getTokenForGroup(group string) string {
	for _, override := range provider.Overrides {
		if override.group == group && len(override.token) > 0 {
			return override.token
		}
	}
	return provider.Token
}

type Body struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var message string
	if resolved {
		message = fmt.Sprintf("An alert for *%s* has been resolved:\n—\n    _healthcheck passing successfully %d time(s) in a row_\n—  ", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for *%s* has been triggered:\n—\n    _healthcheck failed %d time(s) in a row_\n—  ", ep.DisplayName(), alert.FailureThreshold)
	}
	var formattedConditionResults string
	if len(result.ConditionResults) > 0 {
		formattedConditionResults = "\n*Condition results*\n"
		for _, conditionResult := range result.ConditionResults {
			var prefix string
			if conditionResult.Success {
				prefix = "✅"
			} else {
				prefix = "❌"
			}
			formattedConditionResults += fmt.Sprintf("%s - `%s`\n", prefix, conditionResult.Condition)
		}
	}
	var text string
	if len(alert.GetDescription()) > 0 {
		text = fmt.Sprintf("⛑ *Gatus* \n%s \n*Description* \n_%s_  \n%s", message, alert.GetDescription(), formattedConditionResults)
	} else {
		text = fmt.Sprintf("⛑ *Gatus* \n%s%s", message, formattedConditionResults)
	}
	bodyAsJSON, _ := json.Marshal(Body{
		ChatID:    provider.getIDForGroup(ep.Group),
		Text:      text,
		ParseMode: "MARKDOWN",
	})
	return bodyAsJSON
}

func (provider *AlertProvider) getIDForGroup(group string) string {
	for _, override := range provider.Overrides {
		if override.group == group && len(override.id) > 0 {
			return override.id
		}
	}
	return provider.ID
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

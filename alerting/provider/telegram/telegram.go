package telegram

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

const ApiURL = "https://api.telegram.org"

var (
	ErrTokenNotSet            = errors.New("token not set")
	ErrIDNotSet               = errors.New("id not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	Token   string `yaml:"token"`
	ID      string `yaml:"id"`
	TopicID string `yaml:"topic-id,omitempty"`
	ApiUrl  string `yaml:"api-url"`

	ClientConfig *client.Config `yaml:"client,omitempty"`
}

func (cfg *Config) Validate() error {
	if len(cfg.ApiUrl) == 0 {
		cfg.ApiUrl = ApiURL
	}
	if len(cfg.Token) == 0 {
		return ErrTokenNotSet
	}
	if len(cfg.ID) == 0 {
		return ErrIDNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if override.ClientConfig != nil {
		cfg.ClientConfig = override.ClientConfig
	}
	if len(override.Token) > 0 {
		cfg.Token = override.Token
	}
	if len(override.ID) > 0 {
		cfg.ID = override.ID
	}
	if len(override.TopicID) > 0 {
		cfg.TopicID = override.TopicID
	}
	if len(override.ApiUrl) > 0 {
		cfg.ApiUrl = override.ApiUrl
	}
}

// AlertProvider is the configuration necessary for sending an alert using Telegram
type AlertProvider struct {
	DefaultConfig Config `yaml:",inline"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of overrides that may be prioritized over the default configuration
	Overrides []*Override `yaml:"overrides,omitempty"`
}

// Override is a configuration that may be prioritized over the default configuration
type Override struct {
	Group  string `yaml:"group"`
	Config `yaml:",inline"`
}

// Validate the provider's configuration
func (provider *AlertProvider) Validate() error {
	registeredGroups := make(map[string]bool)
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" {
				return ErrDuplicateGroupOverride
			}
			registeredGroups[override.Group] = true
		}
	}
	return provider.DefaultConfig.Validate()
}

// Send an alert using the provider
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(provider.buildRequestBody(cfg, ep, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/bot%s/sendMessage", cfg.ApiUrl, cfg.Token), buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(cfg.ClientConfig).Do(request)
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
	TopicID   string `json:"message_thread_id,omitempty"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
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
		text = fmt.Sprintf("⛑ *Gatus* \n%s \n*Description* \n%s  \n%s", message, alert.GetDescription(), formattedConditionResults)
	} else {
		text = fmt.Sprintf("⛑ *Gatus* \n%s%s", message, formattedConditionResults)
	}
	bodyAsJSON, _ := json.Marshal(Body{
		ChatID:    cfg.ID,
		Text:      text,
		ParseMode: "MARKDOWN",
		TopicID:   cfg.TopicID,
	})
	return bodyAsJSON
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// GetConfig returns the configuration for the provider with the overrides applied
func (provider *AlertProvider) GetConfig(group string, alert *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig
	// Handle group overrides
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				cfg.Merge(&override.Config)
				break
			}
		}
	}
	// Handle alert overrides
	if len(alert.ProviderOverride) != 0 {
		overrideConfig := Config{}
		if err := yaml.Unmarshal(alert.ProviderOverrideAsBytes(), &overrideConfig); err != nil {
			return nil, err
		}
		cfg.Merge(&overrideConfig)
	}
	// Validate the configuration
	err := cfg.Validate()
	return &cfg, err
}

// ValidateOverrides validates the alert's provider override and, if present, the group override
func (provider *AlertProvider) ValidateOverrides(group string, alert *alert.Alert) error {
	_, err := provider.GetConfig(group, alert)
	return err
}

package zulip

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrBotEmailNotSet         = errors.New("bot-email not set")
	ErrBotAPIKeyNotSet        = errors.New("bot-api-key not set")
	ErrDomainNotSet           = errors.New("domain not set")
	ErrChannelIDNotSet        = errors.New("channel-id not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	BotEmail  string `yaml:"bot-email"`   // Email of the bot user
	BotAPIKey string `yaml:"bot-api-key"` // API key of the bot user
	Domain    string `yaml:"domain"`      // Domain of the Zulip server
	ChannelID string `yaml:"channel-id"`  // ID of the channel to send the message to
}

func (cfg *Config) Validate() error {
	if len(cfg.BotEmail) == 0 {
		return ErrBotEmailNotSet
	}
	if len(cfg.BotAPIKey) == 0 {
		return ErrBotAPIKeyNotSet
	}
	if len(cfg.Domain) == 0 {
		return ErrDomainNotSet
	}
	if len(cfg.ChannelID) == 0 {
		return ErrChannelIDNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.BotEmail) > 0 {
		cfg.BotEmail = override.BotEmail
	}
	if len(override.BotAPIKey) > 0 {
		cfg.BotAPIKey = override.BotAPIKey
	}
	if len(override.Domain) > 0 {
		cfg.Domain = override.Domain
	}
	if len(override.ChannelID) > 0 {
		cfg.ChannelID = override.ChannelID
	}
}

// AlertProvider is the configuration necessary for sending an alert using Zulip
type AlertProvider struct {
	DefaultConfig Config `yaml:",inline"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

// Override is a case under which the default integration is overridden
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
	buffer := bytes.NewBufferString(provider.buildRequestBody(cfg, ep, alert, result, resolved))
	zulipEndpoint := fmt.Sprintf("https://%s/api/v1/messages", cfg.Domain)
	request, err := http.NewRequest(http.MethodPost, zulipEndpoint, buffer)
	if err != nil {
		return err
	}
	request.SetBasicAuth(cfg.BotEmail, cfg.BotAPIKey)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "Gatus")
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return nil
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) string {
	var message string
	if resolved {
		message = fmt.Sprintf("An alert for **%s** has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for **%s** has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		message += "\n> " + alertDescription + "\n"
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = ":check:"
		} else {
			prefix = ":cross_mark:"
		}
		message += fmt.Sprintf("\n%s - `%s`", prefix, conditionResult.Condition)
	}
	return url.Values{
		"type":    {"channel"},
		"to":      {cfg.ChannelID},
		"topic":   {"Gatus"},
		"content": {message},
	}.Encode()
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

package pushward

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

const (
	ApiURL = "https://api.pushward.app/notifications"

	defaultLevel         = "time-sensitive"
	defaultResolvedLevel = "active"

	maximumTitleLength = 256

	// maximumEncodedTextSize is the space for the JSON encoded title and body, since push services refuse a payload over 4096 bytes
	maximumEncodedTextSize = 3584
)

var (
	ErrIntegrationKeyNotSet   = errors.New("integration-key not set")
	ErrInvalidLevel           = errors.New("level and resolved-level must be one of passive, active, time-sensitive or critical")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	// Integration key used to authenticate with PushWard
	IntegrationKey string `yaml:"integration-key"`

	// The title of your message
	// default: "Gatus: <endpoint>"
	Title string `yaml:"title,omitempty"`

	// Level of triggered messages, one of passive, active, time-sensitive or critical
	// default: time-sensitive
	Level string `yaml:"level,omitempty"`

	// Level of resolved messages, one of passive, active, time-sensitive or critical
	// default: active
	ResolvedLevel string `yaml:"resolved-level,omitempty"`
}

func (cfg *Config) Validate() error {
	if len(cfg.Level) == 0 {
		cfg.Level = defaultLevel
	}
	if len(cfg.ResolvedLevel) == 0 {
		cfg.ResolvedLevel = defaultResolvedLevel
	}
	if len(cfg.IntegrationKey) == 0 {
		return ErrIntegrationKeyNotSet
	}
	for _, level := range []string{cfg.Level, cfg.ResolvedLevel} {
		if level != "passive" && level != "active" && level != "time-sensitive" && level != "critical" {
			return ErrInvalidLevel
		}
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.IntegrationKey) > 0 {
		cfg.IntegrationKey = override.IntegrationKey
	}
	if len(override.Title) > 0 {
		cfg.Title = override.Title
	}
	if len(override.Level) > 0 {
		cfg.Level = override.Level
	}
	if len(override.ResolvedLevel) > 0 {
		cfg.ResolvedLevel = override.ResolvedLevel
	}
}

// AlertProvider is the configuration necessary for sending an alert using PushWard
type AlertProvider struct {
	DefaultConfig Config `yaml:",inline"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

// Override is a case under which the default configuration is overridden
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
			if _, err := provider.GetConfig(override.Group, &alert.Alert{}); err != nil {
				return err
			}
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
	request, err := http.NewRequest(http.MethodPost, ApiURL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+cfg.IntegrationKey)
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return nil
}

type Body struct {
	Title             string `json:"title"`
	Body              string `json:"body"`
	Level             string `json:"level"`
	Source            string `json:"source"`
	SourceDisplayName string `json:"source_display_name"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var message, formattedConditionResults string
	level := cfg.Level
	if resolved {
		level = cfg.ResolvedLevel
		message = fmt.Sprintf("An alert for %s has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for %s has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "✅"
		} else {
			prefix = "❌"
		}
		formattedConditionResults += fmt.Sprintf("\n%s - %s", prefix, conditionResult.Condition)
	}
	if len(alert.GetDescription()) > 0 {
		message += " with the following description: " + alert.GetDescription()
	}
	message += formattedConditionResults
	title := "Gatus: " + ep.DisplayName()
	if cfg.Title != "" {
		title = cfg.Title
	}
	title = truncate(title, maximumTitleLength)
	body, _ := json.Marshal(Body{
		Title:             title,
		Body:              truncateEncoded(message, maximumEncodedTextSize-encodedSize(title)),
		Level:             level,
		Source:            "gatus",
		SourceDisplayName: "Gatus",
	})
	return body
}

// truncate caps the text at the given number of characters, since the API refuses longer values
func truncate(text string, maximum int) string {
	if runes := []rune(text); len(runes) > maximum {
		return string(runes[:maximum])
	}
	return text
}

// truncateEncoded cuts the text so that its JSON encoding takes at most the given number of bytes
func truncateEncoded(text string, maximum int) string {
	for index, character := range text {
		if maximum -= encodedSize(string(character)); maximum < 0 {
			return text[:index]
		}
	}
	return text
}

// encodedSize returns the number of bytes the text takes inside a JSON string
func encodedSize(text string) int {
	encoded, _ := json.Marshal(text)
	return len(encoded) - len(`""`)
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

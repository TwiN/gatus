package ifttt

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

var (
	ErrWebhookKeyNotSet       = errors.New("webhook-key not set")
	ErrEventNameNotSet        = errors.New("event-name not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	WebhookKey string `yaml:"webhook-key"` // IFTTT Webhook key
	EventName  string `yaml:"event-name"`  // IFTTT event name
}

func (cfg *Config) Validate() error {
	if len(cfg.WebhookKey) == 0 {
		return ErrWebhookKeyNotSet
	}
	if len(cfg.EventName) == 0 {
		return ErrEventNameNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.WebhookKey) > 0 {
		cfg.WebhookKey = override.WebhookKey
	}
	if len(override.EventName) > 0 {
		cfg.EventName = override.EventName
	}
}

// AlertProvider is the configuration necessary for sending an alert using IFTTT
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
	url := fmt.Sprintf("https://maker.ifttt.com/trigger/%s/with/key/%s", cfg.EventName, cfg.WebhookKey)
	body, err := provider.buildRequestBody(ep, alert, result, resolved)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(body)
	request, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to ifttt alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return err
}

type Body struct {
	Value1 string `json:"value1"` // Alert status/title
	Value2 string `json:"value2"` // Alert message
	Value3 string `json:"value3"` // Additional details
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) ([]byte, error) {
	var value1, value2, value3 string
	if resolved {
		value1 = fmt.Sprintf("✅ RESOLVED: %s", ep.DisplayName())
		value2 = fmt.Sprintf("Alert has been resolved after passing successfully %d time(s) in a row", alert.SuccessThreshold)
	} else {
		value1 = fmt.Sprintf("🚨 ALERT: %s", ep.DisplayName())
		value2 = fmt.Sprintf("Endpoint has failed %d time(s) in a row", alert.FailureThreshold)
	}
	// Build additional details
	value3 = fmt.Sprintf("Endpoint: %s", ep.DisplayName())
	if ep.Group != "" {
		value3 += fmt.Sprintf(" | Group: %s", ep.Group)
	}
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		value3 += fmt.Sprintf(" | Description: %s", alertDescription)
	}
	// Add condition results summary
	if len(result.ConditionResults) > 0 {
		successCount := 0
		for _, conditionResult := range result.ConditionResults {
			if conditionResult.Success {
				successCount++
			}
		}
		value3 += fmt.Sprintf(" | Conditions: %d/%d passed", successCount, len(result.ConditionResults))
	}
	body := Body{
		Value1: value1,
		Value2: value2,
		Value3: value3,
	}
	bodyAsJSON, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bodyAsJSON, nil
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

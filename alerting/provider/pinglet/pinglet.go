package pinglet

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

const (
	DefaultURL      = "https://pinglet.dev"
	DefaultPriority = "normal"
)

var (
	ErrAPIKeyNotSet           = errors.New("api-key not set")
	ErrNamespaceNotSet        = errors.New("namespace not set")
	ErrTopicNotSet            = errors.New("topic not set")
	ErrInvalidPriority        = errors.New("priority must be one of silent, normal or urgent")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")

	validPriorities = map[string]bool{"silent": true, "normal": true, "urgent": true}
)

type Config struct {
	APIKey    string `yaml:"api-key"`            // API key used to authenticate against Pinglet
	Namespace string `yaml:"namespace"`          // Namespace the topic resides in
	Topic     string `yaml:"topic"`              // Topic to publish the notification to
	URL       string `yaml:"url,omitempty"`      // Defaults to DefaultURL
	Priority  string `yaml:"priority,omitempty"` // Defaults to DefaultPriority
}

func (cfg *Config) Validate() error {
	if len(cfg.URL) == 0 {
		cfg.URL = DefaultURL
	}
	if len(cfg.Priority) == 0 {
		cfg.Priority = DefaultPriority
	}
	if !validPriorities[cfg.Priority] {
		return ErrInvalidPriority
	}
	if len(cfg.APIKey) == 0 {
		return ErrAPIKeyNotSet
	}
	if len(cfg.Namespace) == 0 {
		return ErrNamespaceNotSet
	}
	if len(cfg.Topic) == 0 {
		return ErrTopicNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.APIKey) > 0 {
		cfg.APIKey = override.APIKey
	}
	if len(override.Namespace) > 0 {
		cfg.Namespace = override.Namespace
	}
	if len(override.Topic) > 0 {
		cfg.Topic = override.Topic
	}
	if len(override.URL) > 0 {
		cfg.URL = override.URL
	}
	if len(override.Priority) > 0 {
		cfg.Priority = override.Priority
	}
}

// AlertProvider is the configuration necessary for sending an alert using Pinglet
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
			if len(override.Group) == 0 {
				return ErrDuplicateGroupOverride
			}
			if _, ok := registeredGroups[override.Group]; ok {
				return ErrDuplicateGroupOverride
			}
			if len(override.Priority) > 0 && !validPriorities[override.Priority] {
				return ErrInvalidPriority
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
	url := strings.TrimSuffix(cfg.URL, "/") + "/" + cfg.Namespace + "/" + cfg.Topic
	request, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+cfg.APIKey)
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
	Title    string `json:"title"`
	Message  string `json:"message"`
	Priority string `json:"priority"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var message string
	if resolved {
		message = "An alert has been resolved after passing successfully " + strconv.Itoa(alert.SuccessThreshold) + " time(s) in a row"
	} else {
		message = "An alert has been triggered due to having failed " + strconv.Itoa(alert.FailureThreshold) + " time(s) in a row"
	}
	if len(alert.GetDescription()) > 0 {
		message += " with the following description: " + alert.GetDescription()
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "🟢"
		} else {
			prefix = "🔴"
		}
		message += fmt.Sprintf("\n%s %s", prefix, conditionResult.Condition)
	}
	body, _ := json.Marshal(Body{
		Title:    "Gatus: " + ep.DisplayName(),
		Message:  message,
		Priority: cfg.Priority,
	})
	return body
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

package ntfy

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
	DefaultURL      = "https://ntfy.sh"
	DefaultPriority = 3
	TokenPrefix     = "tk_"
)

var (
	ErrInvalidToken           = errors.New("invalid token")
	ErrTopicNotSet            = errors.New("topic not set")
	ErrInvalidPriority        = errors.New("priority must between 1 and 5 inclusively")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	Topic           string `yaml:"topic"`
	URL             string `yaml:"url,omitempty"`              // Defaults to DefaultURL
	Priority        int    `yaml:"priority,omitempty"`         // Defaults to DefaultPriority
	Token           string `yaml:"token,omitempty"`            // Defaults to ""
	Email           string `yaml:"email,omitempty"`            // Defaults to ""
	Click           string `yaml:"click,omitempty"`            // Defaults to ""
	DisableFirebase bool   `yaml:"disable-firebase,omitempty"` // Defaults to false
	DisableCache    bool   `yaml:"disable-cache,omitempty"`    // Defaults to false
}

func (cfg *Config) Validate() error {
	if len(cfg.URL) == 0 {
		cfg.URL = DefaultURL
	}
	if cfg.Priority == 0 {
		cfg.Priority = DefaultPriority
	}
	if len(cfg.Token) > 0 && !strings.HasPrefix(cfg.Token, TokenPrefix) {
		return ErrInvalidToken
	}
	if len(cfg.Topic) == 0 {
		return ErrTopicNotSet
	}
	if cfg.Priority < 1 || cfg.Priority > 5 {
		return ErrInvalidPriority
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.Topic) > 0 {
		cfg.Topic = override.Topic
	}
	if len(override.URL) > 0 {
		cfg.URL = override.URL
	}
	if override.Priority > 0 {
		cfg.Priority = override.Priority
	}
	if len(override.Token) > 0 {
		cfg.Token = override.Token
	}
	if len(override.Email) > 0 {
		cfg.Email = override.Email
	}
	if len(override.Click) > 0 {
		cfg.Click = override.Click
	}
	if override.DisableFirebase {
		cfg.DisableFirebase = true
	}
	if override.DisableCache {
		cfg.DisableCache = true
	}
}

// AlertProvider is the configuration necessary for sending an alert using Slack
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
			if len(override.Token) > 0 && !strings.HasPrefix(override.Token, TokenPrefix) {
				return ErrDuplicateGroupOverride
			}
			if override.Priority < 0 || override.Priority >= 6 {
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
	url := cfg.URL
	request, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if token := cfg.Token; len(token) > 0 {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if cfg.DisableFirebase {
		request.Header.Set("Firebase", "no")
	}
	if cfg.DisableCache {
		request.Header.Set("Cache", "no")
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
	Email    string   `json:"email,omitempty"`
	Click    string   `json:"click,omitempty"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var message, formattedConditionResults, tag string
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
		formattedConditionResults += fmt.Sprintf("\n%s %s", prefix, conditionResult.Condition)
	}
	if len(alert.GetDescription()) > 0 {
		message += " with the following description: " + alert.GetDescription()
	}
	message += formattedConditionResults
	body, _ := json.Marshal(Body{
		Topic:    cfg.Topic,
		Title:    "Gatus: " + ep.DisplayName(),
		Message:  message,
		Tags:     []string{tag},
		Priority: cfg.Priority,
		Email:    cfg.Email,
		Click:    cfg.Click,
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

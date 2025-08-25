package datadog

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrAPIKeyNotSet           = errors.New("api-key not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	APIKey string   `yaml:"api-key"`        // Datadog API key
	Site   string   `yaml:"site,omitempty"` // Datadog site (e.g., datadoghq.com, datadoghq.eu)
	Tags   []string `yaml:"tags,omitempty"` // Additional tags to include
}

func (cfg *Config) Validate() error {
	if len(cfg.APIKey) == 0 {
		return ErrAPIKeyNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.APIKey) > 0 {
		cfg.APIKey = override.APIKey
	}
	if len(override.Site) > 0 {
		cfg.Site = override.Site
	}
	if len(override.Tags) > 0 {
		cfg.Tags = override.Tags
	}
}

// AlertProvider is the configuration necessary for sending an alert using Datadog
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
	site := cfg.Site
	if site == "" {
		site = "datadoghq.com"
	}
	body, err := provider.buildRequestBody(cfg, ep, alert, result, resolved)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(body)
	url := fmt.Sprintf("https://api.%s/api/v1/events", site)
	request, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("DD-API-KEY", cfg.APIKey)
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to datadog alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return nil
}

type Body struct {
	Title        string   `json:"title"`
	Text         string   `json:"text"`
	Priority     string   `json:"priority"`
	Tags         []string `json:"tags"`
	AlertType    string   `json:"alert_type"`
	SourceType   string   `json:"source_type_name"`
	DateHappened int64    `json:"date_happened,omitempty"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) ([]byte, error) {
	var title, text, priority, alertType string
	if resolved {
		title = fmt.Sprintf("Resolved: %s", ep.DisplayName())
		text = fmt.Sprintf("Alert for %s has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
		priority = "normal"
		alertType = "success"
	} else {
		title = fmt.Sprintf("Alert: %s", ep.DisplayName())
		text = fmt.Sprintf("Alert for %s has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
		priority = "normal"
		alertType = "error"
	}
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		text += fmt.Sprintf("\n\nDescription: %s", alertDescription)
	}
	if len(result.ConditionResults) > 0 {
		text += "\n\nCondition Results:"
		for _, conditionResult := range result.ConditionResults {
			var status string
			if conditionResult.Success {
				status = "✅"
			} else {
				status = "❌"
			}
			text += fmt.Sprintf("\n%s %s", status, conditionResult.Condition)
		}
	}
	tags := []string{
		"source:gatus",
		fmt.Sprintf("endpoint:%s", ep.Name),
		fmt.Sprintf("status:%s", alertType),
	}
	if ep.Group != "" {
		tags = append(tags, fmt.Sprintf("group:%s", ep.Group))
	}
	// Append custom tags
	if len(cfg.Tags) > 0 {
		tags = append(tags, cfg.Tags...)
	}
	body := Body{
		Title:        title,
		Text:         text,
		Priority:     priority,
		Tags:         tags,
		AlertType:    alertType,
		SourceType:   "gatus",
		DateHappened: time.Now().Unix(),
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

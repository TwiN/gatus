package newrelic

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
	ErrInsertKeyNotSet        = errors.New("insert-key not set")
	ErrAccountIDNotSet        = errors.New("account-id not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	InsertKey string `yaml:"insert-key"`       // New Relic Insert key
	AccountID string `yaml:"account-id"`       // New Relic account ID
	Region    string `yaml:"region,omitempty"` // Region (US or EU, defaults to US)
}

func (cfg *Config) Validate() error {
	if len(cfg.InsertKey) == 0 {
		return ErrInsertKeyNotSet
	}
	if len(cfg.AccountID) == 0 {
		return ErrAccountIDNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.InsertKey) > 0 {
		cfg.InsertKey = override.InsertKey
	}
	if len(override.AccountID) > 0 {
		cfg.AccountID = override.AccountID
	}
	if len(override.Region) > 0 {
		cfg.Region = override.Region
	}
}

// AlertProvider is the configuration necessary for sending an alert using New Relic
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
	// Determine the API endpoint based on region
	var apiHost string
	if cfg.Region == "EU" {
		apiHost = "insights-collector.eu01.nr-data.net"
	} else {
		apiHost = "insights-collector.newrelic.com"
	}
	body, err := provider.buildRequestBody(cfg, ep, alert, result, resolved)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(body)
	url := fmt.Sprintf("https://%s/v1/accounts/%s/events", apiHost, cfg.AccountID)
	request, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Insert-Key", cfg.InsertKey)
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to newrelic alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return nil
}

type Event struct {
	EventType   string  `json:"eventType"`
	Timestamp   int64   `json:"timestamp"`
	Service     string  `json:"service"`
	Endpoint    string  `json:"endpoint"`
	Group       string  `json:"group,omitempty"`
	AlertStatus string  `json:"alertStatus"`
	Message     string  `json:"message"`
	Description string  `json:"description,omitempty"`
	Severity    string  `json:"severity"`
	Source      string  `json:"source"`
	SuccessRate float64 `json:"successRate,omitempty"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) ([]byte, error) {
	var alertStatus, severity, message string
	var successRate float64
	if resolved {
		alertStatus = "resolved"
		severity = "INFO"
		message = fmt.Sprintf("Alert for %s has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
		successRate = 100
	} else {
		alertStatus = "triggered"
		severity = "CRITICAL"
		message = fmt.Sprintf("Alert for %s has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
		successRate = 0
	}
	// Calculate success rate from condition results
	if len(result.ConditionResults) > 0 {
		successCount := 0
		for _, conditionResult := range result.ConditionResults {
			if conditionResult.Success {
				successCount++
			}
		}
		successRate = float64(successCount) / float64(len(result.ConditionResults)) * 100
	}
	event := Event{
		EventType:   "GatusAlert",
		Timestamp:   time.Now().Unix() * 1000, // New Relic expects milliseconds
		Service:     "Gatus",
		Endpoint:    ep.DisplayName(),
		Group:       ep.Group,
		AlertStatus: alertStatus,
		Message:     message,
		Description: alert.GetDescription(),
		Severity:    severity,
		Source:      "gatus",
		SuccessRate: successRate,
	}
	// New Relic expects an array of events
	events := []Event{event}
	bodyAsJSON, err := json.Marshal(events)
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

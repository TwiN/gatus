package ilert

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
	restAPIUrl = "https://api.ilert.com/api/v1/events/gatus/"
)

var (
	ErrIntegrationKeyNotSet   = errors.New("integration key is not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	IntegrationKey string `yaml:"integration-key"`
}

func (cfg *Config) Validate() error {
	if len(cfg.IntegrationKey) == 0 {
		return ErrIntegrationKeyNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.IntegrationKey) > 0 {
		cfg.IntegrationKey = override.IntegrationKey
	}
}

// AlertProvider is the configuration necessary for sending an alert using ilert
type AlertProvider struct {
	DefaultConfig Config `yaml:",inline"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

type Override struct {
	Group  string `yaml:"group"`
	Config `yaml:",inline"`
}

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

func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(provider.buildRequestBody(cfg, ep, alert, result, resolved))

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s%s", restAPIUrl, cfg.IntegrationKey), buffer)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(nil).Do(req)
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
	Alert            alert.Alert                 `json:"alert"`
	Name             string                      `json:"name"`
	Group            string                      `json:"group"`
	Status           string                      `json:"status"`
	Title            string                      `json:"title"`
	Details          string                      `json:"details,omitempty"`
	ConditionResults []*endpoint.ConditionResult `json:"condition_results"`
	URL              string                      `json:"url"`
}

func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var details, status string
	if resolved {
		status = "resolved"
	} else {
		status = "firing"
	}

	if len(alert.GetDescription()) > 0 {
		details = alert.GetDescription()
	} else {
		details = "No description"
	}

	var body []byte
	body, _ = json.Marshal(Body{
		Alert:            *alert,
		Name:             ep.Name,
		Group:            ep.Group,
		Title:            ep.DisplayName(),
		Status:           status,
		Details:          details,
		ConditionResults: result.ConditionResults,
		URL:              ep.URL,
	})
	return body
}

func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

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

package zapier

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
	ErrWebhookURLNotSet       = errors.New("webhook-url not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	WebhookURL string `yaml:"webhook-url"` // Zapier webhook URL
}

func (cfg *Config) Validate() error {
	if len(cfg.WebhookURL) == 0 {
		return ErrWebhookURLNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.WebhookURL) > 0 {
		cfg.WebhookURL = override.WebhookURL
	}
}

// AlertProvider is the configuration necessary for sending an alert using Zapier
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
	body, err := provider.buildRequestBody(ep, alert, result, resolved)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(body)
	request, err := http.NewRequest(http.MethodPost, cfg.WebhookURL, buffer)
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
		return fmt.Errorf("call to zapier alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return nil
}

type Body struct {
	AlertType        string            `json:"alert_type"`
	Status           string            `json:"status"`
	Endpoint         string            `json:"endpoint"`
	Group            string            `json:"group,omitempty"`
	Message          string            `json:"message"`
	Description      string            `json:"description,omitempty"`
	Timestamp        string            `json:"timestamp"`
	SuccessThreshold int               `json:"success_threshold,omitempty"`
	FailureThreshold int               `json:"failure_threshold,omitempty"`
	ConditionResults []*endpoint.ConditionResult `json:"condition_results,omitempty"`
	TotalConditions  int               `json:"total_conditions"`
	PassedConditions int               `json:"passed_conditions"`
	FailedConditions int               `json:"failed_conditions"`
}


// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) ([]byte, error) {
	var alertType, status, message string
	var successThreshold, failureThreshold int
	if resolved {
		alertType = "resolved"
		status = "ok"
		message = fmt.Sprintf("Alert for %s has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
		successThreshold = alert.SuccessThreshold
	} else {
		alertType = "triggered"
		status = "critical"
		message = fmt.Sprintf("Alert for %s has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
		failureThreshold = alert.FailureThreshold
	}
	// Process condition results
	passedConditions := 0
	failedConditions := 0
	for _, cr := range result.ConditionResults {
		if cr.Success {
			passedConditions++
		} else {
			failedConditions++
		}
	}
	body := Body{
		AlertType:        alertType,
		Status:           status,
		Endpoint:         ep.DisplayName(),
		Group:            ep.Group,
		Message:          message,
		Description:      alert.GetDescription(),
		Timestamp:        time.Now().Format(time.RFC3339),
		SuccessThreshold: successThreshold,
		FailureThreshold: failureThreshold,
		ConditionResults: result.ConditionResults,
		TotalConditions:  len(result.ConditionResults),
		PassedConditions: passedConditions,
		FailedConditions: failedConditions,
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

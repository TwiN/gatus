package signal

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrApiURLNotSet           = errors.New("api-url not set")
	ErrNumberNotSet           = errors.New("number not set")
	ErrRecipientsNotSet       = errors.New("recipients not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	ApiURL     string   `yaml:"api-url"`    // Signal API URL (e.g., signal-cli-rest-api instance)
	Number     string   `yaml:"number"`     // Sender phone number
	Recipients []string `yaml:"recipients"` // List of recipient phone numbers
}

func (cfg *Config) Validate() error {
	if len(cfg.ApiURL) == 0 {
		return ErrApiURLNotSet
	}
	if !strings.HasSuffix(cfg.ApiURL, "/v2/send") {
		cfg.ApiURL = cfg.ApiURL + "/v2/send"
	}
	if len(cfg.Number) == 0 {
		return ErrNumberNotSet
	}
	if len(cfg.Recipients) == 0 {
		return ErrRecipientsNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.ApiURL) > 0 {
		cfg.ApiURL = override.ApiURL
	}
	if len(override.Number) > 0 {
		cfg.Number = override.Number
	}
	if len(override.Recipients) > 0 {
		cfg.Recipients = override.Recipients
	}
}

// AlertProvider is the configuration necessary for sending an alert using Signal
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
	for _, recipient := range cfg.Recipients {
		body, err := provider.buildRequestBody(cfg, ep, alert, result, resolved, recipient)
		if err != nil {
			return err
		}
		buffer := bytes.NewBuffer(body)
		request, err := http.NewRequest(http.MethodPost, cfg.ApiURL, buffer)
		if err != nil {
			return err
		}
		request.Header.Set("Content-Type", "application/json")
		response, err := client.GetHTTPClient(nil).Do(request)
		if err != nil {
			return err
		}
		if response.StatusCode >= 400 {
			body, _ := io.ReadAll(response.Body)
			response.Body.Close()
			return fmt.Errorf("call to signal alert returned status code %d: %s", response.StatusCode, string(body))
		}
		response.Body.Close()
	}
	return nil
}

type Body struct {
	Message    string   `json:"message"`
	Number     string   `json:"number"`
	Recipients []string `json:"recipients"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool, recipient string) ([]byte, error) {
	var message string
	if resolved {
		message = fmt.Sprintf("🟢 RESOLVED: %s\nAlert has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("🔴 ALERT: %s\nEndpoint has failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		message += fmt.Sprintf("\n\nDescription: %s", alertDescription)
	}
	if len(result.ConditionResults) > 0 {
		message += "\n\nCondition results:"
		for _, conditionResult := range result.ConditionResults {
			var status string
			if conditionResult.Success {
				status = "✅"
			} else {
				status = "❌"
			}
			message += fmt.Sprintf("\n%s %s", status, conditionResult.Condition)
		}
	}
	body := Body{
		Message:    message,
		Number:     cfg.Number,
		Recipients: []string{recipient},
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

package incidentio

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
	"github.com/TwiN/logr"
	"gopkg.in/yaml.v3"
)

const (
	restAPIUrl = "https://api.incident.io/v2/alert_events/http/"
)

var (
	ErrURLNotSet                    = errors.New("url not set")
	ErrURLNotPrefixedWithRestAPIURL = fmt.Errorf("url must be prefixed with %s", restAPIUrl)
	ErrDuplicateGroupOverride       = errors.New("duplicate group override")
	ErrAuthTokenNotSet              = errors.New("auth-token not set")
)

type Config struct {
	URL       string                 `yaml:"url,omitempty"`
	AuthToken string                 `yaml:"auth-token,omitempty"`
	SourceURL string                 `yaml:"source-url,omitempty"`
	Metadata  map[string]interface{} `yaml:"metadata,omitempty"`
}

func (cfg *Config) Validate() error {
	if len(cfg.URL) == 0 {
		return ErrURLNotSet
	}
	if !strings.HasPrefix(cfg.URL, restAPIUrl) {
		return ErrURLNotPrefixedWithRestAPIURL
	}
	if len(cfg.AuthToken) == 0 {
		return ErrAuthTokenNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.URL) > 0 {
		cfg.URL = override.URL
	}
	if len(override.AuthToken) > 0 {
		cfg.AuthToken = override.AuthToken
	}
	if len(override.SourceURL) > 0 {
		cfg.SourceURL = override.SourceURL
	}
	if len(override.Metadata) > 0 {
		cfg.Metadata = override.Metadata
	}
}

// AlertProvider is the configuration necessary for sending an alert using incident.io
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
	req, err := http.NewRequest(http.MethodPost, cfg.URL, buffer)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.AuthToken)
	response, err := client.GetHTTPClient(nil).Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	incidentioResponse := Response{}
	err = json.NewDecoder(response.Body).Decode(&incidentioResponse)
	if err != nil {
		// Silently fail. We don't want to create tons of alerts just because we failed to parse the body.
		logr.Errorf("[incidentio.Send] Ran into error decoding pagerduty response: %s", err.Error())
	}
	alert.ResolveKey = incidentioResponse.DeduplicationKey
	return err
}

type Body struct {
	AlertSourceConfigID string                 `json:"alert_source_config_id"`
	Status              string                 `json:"status"`
	Title               string                 `json:"title"`
	DeduplicationKey    string                 `json:"deduplication_key,omitempty"`
	Description         string                 `json:"description,omitempty"`
	SourceURL           string                 `json:"source_url,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
}

type Response struct {
	DeduplicationKey string `json:"deduplication_key"`
}

func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var message, formattedConditionResults, status string
	if resolved {
		message = "An alert has been resolved after passing successfully " + strconv.Itoa(alert.SuccessThreshold) + " time(s) in a row"
		status = "resolved"
	} else {
		message = "An alert has been triggered due to having failed " + strconv.Itoa(alert.FailureThreshold) + " time(s) in a row"
		status = "firing"
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "🟢"
		} else {
			prefix = "🔴"
		}
		formattedConditionResults += fmt.Sprintf(" %s %s ", prefix, conditionResult.Condition)
	}
	if len(alert.GetDescription()) > 0 {
		message += " with the following description: " + alert.GetDescription()
	}
	message += fmt.Sprintf(" and the following conditions: %s ", formattedConditionResults)

	// Generate deduplication key if empty (first firing)
	if alert.ResolveKey == "" {
		// Generate unique key (endpoint key, alert type, timestamp)
		alert.ResolveKey = generateDeduplicationKey(ep, alert)
	}
	// Extract alert_source_config_id from URL
	alertSourceID := strings.TrimPrefix(cfg.URL, restAPIUrl)
	// Merge metadata: cfg.Metadata + ep.ExtraLabels (if present)
	mergedMetadata := map[string]interface{}{}
	// Copy cfg.Metadata
	for k, v := range cfg.Metadata {
		mergedMetadata[k] = v
	}
	// Add extra labels from endpoint (if present)
	if ep.ExtraLabels != nil && len(ep.ExtraLabels) > 0 {
		for k, v := range ep.ExtraLabels {
			mergedMetadata[k] = v
		}
	}

	body, _ := json.Marshal(Body{
		AlertSourceConfigID: alertSourceID,
		Title:               "Gatus: " + ep.DisplayName(),
		Status:              status,
		DeduplicationKey:    alert.ResolveKey,
		Description:         message,
		SourceURL:           cfg.SourceURL,
		Metadata:            mergedMetadata,
	})
	fmt.Printf("%v", string(body))
	return body
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

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

func (provider *AlertProvider) ValidateOverrides(group string, alert *alert.Alert) error {
	_, err := provider.GetConfig(group, alert)
	return err
}

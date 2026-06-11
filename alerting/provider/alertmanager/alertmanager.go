package alertmanager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrURLsNotSet             = errors.New("urls not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

// Config is the configuration for the Alertmanager provider
type Config struct {
	// URLs is a list of Alertmanager API endpoint URLs to send alerts to
	URLs []string `yaml:"urls"`

	// DefaultSeverity is the default severity level for alerts
	DefaultSeverity string `yaml:"default-severity,omitempty"`

	// ExtraLabels are additional labels to add to all alerts
	ExtraLabels map[string]string `yaml:"extra-labels,omitempty"`

	// ExtraAnnotations are additional annotations to add to all alerts
	ExtraAnnotations map[string]string `yaml:"extra-annotations,omitempty"`

	// ClientConfig is the configuration of the client used to communicate with Alertmanager
	ClientConfig *client.Config `yaml:"client,omitempty"`
}

func (cfg *Config) Validate() error {
	if len(cfg.URLs) == 0 {
		return ErrURLsNotSet
	}
	if len(cfg.DefaultSeverity) == 0 {
		cfg.DefaultSeverity = "critical"
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if override.ClientConfig != nil {
		cfg.ClientConfig = override.ClientConfig
	}
	if len(override.URLs) > 0 {
		cfg.URLs = override.URLs
	}
	if len(override.DefaultSeverity) > 0 {
		cfg.DefaultSeverity = override.DefaultSeverity
	}
	if len(override.ExtraLabels) > 0 {
		if cfg.ExtraLabels == nil {
			cfg.ExtraLabels = make(map[string]string)
		}
		for k, v := range override.ExtraLabels {
			cfg.ExtraLabels[k] = v
		}
	}
	if len(override.ExtraAnnotations) > 0 {
		if cfg.ExtraAnnotations == nil {
			cfg.ExtraAnnotations = make(map[string]string)
		}
		for k, v := range override.ExtraAnnotations {
			cfg.ExtraAnnotations[k] = v
		}
	}
}

// AlertProvider is the configuration necessary for sending alerts to Alertmanager
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

// AlertmanagerAlert represents an alert in Alertmanager API v2 format
type AlertmanagerAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt,omitempty"`
	EndsAt      time.Time         `json:"endsAt,omitempty"`
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

// Send sends an alert to all configured Alertmanager instances
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	alertPayload := provider.buildAlert(cfg, ep, alert, result, resolved)
	return provider.sendToAlertmanagers(cfg, []AlertmanagerAlert{alertPayload})
}

// buildAlert constructs an Alertmanager alert payload
func (provider *AlertProvider) buildAlert(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) AlertmanagerAlert {
	now := time.Now()
	alertPayload := AlertmanagerAlert{
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		StartsAt:    now,
	}

	// Set core Prometheus labels following conventions
	alertPayload.Labels["alertname"] = "GatusEndpointDown"
	instance := ep.URL
	if instance == "" {
		instance = ep.DisplayName()
	}
	alertPayload.Labels["instance"] = instance
	alertPayload.Labels["job"] = "gatus"
	alertPayload.Labels["severity"] = cfg.DefaultSeverity

	// Add Gatus-specific labels
	alertPayload.Labels["endpoint"] = ep.Name
	if ep.Group != "" {
		alertPayload.Labels["group"] = ep.Group
	}

	// Add extra labels from config
	for k, v := range cfg.ExtraLabels {
		alertPayload.Labels[k] = v
	}

	// Set core annotations
	var message, formattedConditionResults string
	if resolved {
		alertPayload.Annotations["summary"] = fmt.Sprintf("Gatus: %s", ep.Name)
		message = "An alert has been resolved after passing successfully " + strconv.Itoa(alert.SuccessThreshold) + " time(s) in a row."
		alertPayload.EndsAt = now
	} else {
		alertPayload.Annotations["summary"] = fmt.Sprintf("Gatus: %s", ep.Name)
		message = "An alert has been triggered due to having failed " + strconv.Itoa(alert.FailureThreshold) + " time(s) in a row."
	}

	// Format condition results
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "🟢"
		} else {
			prefix = "🔴"
		}
		formattedConditionResults += fmt.Sprintf("\n%s %s", prefix, conditionResult.Condition)
	}

	// Add alert description if provided
	if len(alert.GetDescription()) > 0 {
		message += fmt.Sprintf(" %s.", alert.GetDescription())
	}

	// Append condition results
	message += formattedConditionResults

	// Set the final description
	alertPayload.Annotations["description"] = message

	// Add extra annotations from config
	for k, v := range cfg.ExtraAnnotations {
		alertPayload.Annotations[k] = v
	}

	return alertPayload
}

// sendToAlertmanagers sends alerts to all configured Alertmanager instances.
// It attempts to send to all instances and only returns an error if all fail.
func (provider *AlertProvider) sendToAlertmanagers(cfg *Config, alerts []AlertmanagerAlert) error {
	jsonPayload, err := json.Marshal(alerts)
	if err != nil {
		return fmt.Errorf("failed to marshal alerts: %w", err)
	}

	var errs []error
	for _, baseURL := range cfg.URLs {
		if err := provider.sendToURL(cfg, baseURL, jsonPayload); err != nil {
			errs = append(errs, err)
		}
	}

	// Only fail if all instances failed
	if len(errs) == len(cfg.URLs) {
		return fmt.Errorf("failed to send alert to all Alertmanager instances: %v", errs)
	}
	return nil
}

// sendToURL sends the alert payload to a single Alertmanager URL
func (provider *AlertProvider) sendToURL(cfg *Config, baseURL string, jsonPayload []byte) error {
	url := strings.TrimSuffix(baseURL, "/")
	url = strings.TrimSuffix(url, "/api/v2/alerts")
	url = strings.TrimSuffix(url, "/api/v2")
	url += "/api/v2/alerts"

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := client.GetHTTPClient(cfg.ClientConfig)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Alertmanager %s returned status %d: %s", url, resp.StatusCode, string(body))
	}
	return nil
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// GetConfig returns the configuration for the provider with the overrides applied
func (provider *AlertProvider) GetConfig(group string, alert *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig

	// Create deep copies of maps and slices to prevent shared state across alerts
	if cfg.URLs != nil {
		urls := make([]string, len(cfg.URLs))
		copy(urls, cfg.URLs)
		cfg.URLs = urls
	}
	if cfg.ExtraLabels != nil {
		extraLabels := make(map[string]string)
		for k, v := range cfg.ExtraLabels {
			extraLabels[k] = v
		}
		cfg.ExtraLabels = extraLabels
	}
	if cfg.ExtraAnnotations != nil {
		extraAnnotations := make(map[string]string)
		for k, v := range cfg.ExtraAnnotations {
			extraAnnotations[k] = v
		}
		cfg.ExtraAnnotations = extraAnnotations
	}

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

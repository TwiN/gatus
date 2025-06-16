package opsgenie

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
	restAPI = "https://api.opsgenie.com/v2/alerts"
)

var (
	ErrAPIKeyNotSet = errors.New("api-key not set")
)

type Config struct {
	// APIKey to use for
	APIKey string `yaml:"api-key"`

	// Priority to be used in Opsgenie alert payload
	//
	// default: P1
	Priority string `yaml:"priority"`

	// Source define source to be used in Opsgenie alert payload
	//
	// default: gatus
	Source string `yaml:"source"`

	// EntityPrefix is a prefix to be used in entity argument in Opsgenie alert payload
	//
	// default: gatus-
	EntityPrefix string `yaml:"entity-prefix"`

	//AliasPrefix is a prefix to be used in alias argument in Opsgenie alert payload
	//
	// default: gatus-healthcheck-
	AliasPrefix string `yaml:"alias-prefix"`

	// Tags to be used in Opsgenie alert payload
	//
	// default: []
	Tags []string `yaml:"tags"`
}

func (cfg *Config) Validate() error {
	if len(cfg.APIKey) == 0 {
		return ErrAPIKeyNotSet
	}
	if len(cfg.Source) == 0 {
		cfg.Source = "gatus"
	}
	if len(cfg.EntityPrefix) == 0 {
		cfg.EntityPrefix = "gatus-"
	}
	if len(cfg.AliasPrefix) == 0 {
		cfg.AliasPrefix = "gatus-healthcheck-"
	}
	if len(cfg.Priority) == 0 {
		cfg.Priority = "P1"
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.APIKey) > 0 {
		cfg.APIKey = override.APIKey
	}
	if len(override.Priority) > 0 {
		cfg.Priority = override.Priority
	}
	if len(override.Source) > 0 {
		cfg.Source = override.Source
	}
	if len(override.EntityPrefix) > 0 {
		cfg.EntityPrefix = override.EntityPrefix
	}
	if len(override.AliasPrefix) > 0 {
		cfg.AliasPrefix = override.AliasPrefix
	}
	if len(override.Tags) > 0 {
		cfg.Tags = override.Tags
	}
}

type AlertProvider struct {
	DefaultConfig Config `yaml:",inline"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// Validate the provider's configuration
func (provider *AlertProvider) Validate() error {
	return provider.DefaultConfig.Validate()
}

// Send an alert using the provider
//
// Relevant: https://docs.opsgenie.com/docs/alert-api
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	err = provider.sendAlertRequest(cfg, ep, alert, result, resolved)
	if err != nil {
		return err
	}
	if resolved {
		err = provider.closeAlert(cfg, ep, alert)
		if err != nil {
			return err
		}
	}
	if alert.IsSendingOnResolved() {
		if resolved {
			// The alert has been resolved and there's no error, so we can clear the alert's ResolveKey
			alert.ResolveKey = ""
		} else {
			alert.ResolveKey = cfg.AliasPrefix + buildKey(ep)
		}
	}
	return nil
}

func (provider *AlertProvider) sendAlertRequest(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	payload := provider.buildCreateRequestBody(cfg, ep, alert, result, resolved)
	return provider.sendRequest(cfg, restAPI, http.MethodPost, payload)
}

func (provider *AlertProvider) closeAlert(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert) error {
	payload := provider.buildCloseRequestBody(ep, alert)
	url := restAPI + "/" + cfg.AliasPrefix + buildKey(ep) + "/close?identifierType=alias"
	return provider.sendRequest(cfg, url, http.MethodPost, payload)
}

func (provider *AlertProvider) sendRequest(cfg *Config, url, method string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error build alert with payload %v: %w", payload, err)
	}
	request, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "GenieKey "+cfg.APIKey)
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 399 {
		rBody, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(rBody))
	}
	return nil
}

func (provider *AlertProvider) buildCreateRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) alertCreateRequest {
	var message, description string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", ep.Name, alert.GetDescription())
		description = fmt.Sprintf("An alert for *%s* has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("%s - %s", ep.Name, alert.GetDescription())
		description = fmt.Sprintf("An alert for *%s* has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	if ep.Group != "" {
		message = fmt.Sprintf("[%s] %s", ep.Group, message)
	}
	var formattedConditionResults string
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "▣"
		} else {
			prefix = "▢"
		}
		formattedConditionResults += fmt.Sprintf("%s - `%s`\n", prefix, conditionResult.Condition)
	}
	description = description + "\n" + formattedConditionResults
	key := buildKey(ep)
	details := map[string]string{
		"endpoint:url":    ep.URL,
		"endpoint:group":  ep.Group,
		"result:hostname": result.Hostname,
		"result:ip":       result.IP,
		"result:dns_code": result.DNSRCode,
		"result:errors":   strings.Join(result.Errors, ","),
	}
	for k, v := range details {
		if v == "" {
			delete(details, k)
		}
	}
	if result.HTTPStatus > 0 {
		details["result:http_status"] = strconv.Itoa(result.HTTPStatus)
	}
	return alertCreateRequest{
		Message:     message,
		Description: description,
		Source:      cfg.Source,
		Priority:    cfg.Priority,
		Alias:       cfg.AliasPrefix + key,
		Entity:      cfg.EntityPrefix + key,
		Tags:        cfg.Tags,
		Details:     details,
	}
}

func (provider *AlertProvider) buildCloseRequestBody(ep *endpoint.Endpoint, alert *alert.Alert) alertCloseRequest {
	return alertCloseRequest{
		Source: buildKey(ep),
		Note:   fmt.Sprintf("RESOLVED: %s - %s", ep.Name, alert.GetDescription()),
	}
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// GetConfig returns the configuration for the provider with the overrides applied
func (provider *AlertProvider) GetConfig(group string, alert *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig
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

func buildKey(ep *endpoint.Endpoint) string {
	name := toKebabCase(ep.Name)
	if ep.Group == "" {
		return name
	}
	return toKebabCase(ep.Group) + "-" + name
}

func toKebabCase(val string) string {
	return strings.ToLower(strings.ReplaceAll(val, " ", "-"))
}

type alertCreateRequest struct {
	Message     string            `json:"message"`
	Priority    string            `json:"priority"`
	Source      string            `json:"source"`
	Entity      string            `json:"entity"`
	Alias       string            `json:"alias"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags,omitempty"`
	Details     map[string]string `json:"details"`
}

type alertCloseRequest struct {
	Source string `json:"source"`
	Note   string `json:"note"`
}

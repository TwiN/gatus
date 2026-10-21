package signalgrid

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

const APIURL = "https://api.signalgrid.co/v1/push"

var (
	ErrClientKeyNotSet = errors.New("client key not set")
	ErrChannelNotSet   = errors.New("channel not set")
)

type Config struct {
	ClientKey string `yaml:"client-key"`
	Channel   string `yaml:"channel"`
	Critical  *bool  `yaml:"critical,omitempty"`
}

func (cfg *Config) Validate() error {
	if len(cfg.ClientKey) == 0 {
		return ErrClientKeyNotSet
	}
	if len(cfg.Channel) == 0 {
		return ErrChannelNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.ClientKey) > 0 {
		cfg.ClientKey = override.ClientKey
	}
	if len(override.Channel) > 0 {
		cfg.Channel = override.Channel
	}
	if override.Critical != nil {
		cfg.Critical = override.Critical
	}
}

func (cfg *Config) IsCritical() bool {
	return cfg.Critical != nil && *cfg.Critical
}

// AlertProvider is the configuration necessary for sending an alert using Signalgrid
type AlertProvider struct {
	DefaultConfig Config `yaml:",inline"`

	// DefaultAlert is the default alert configuration to use for endpoints
	// with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// Validate the provider's configuration
func (provider *AlertProvider) Validate() error {
	return provider.DefaultConfig.Validate()
}

// Send an alert using the provider
func (provider *AlertProvider) Send(
	ep *endpoint.Endpoint,
	alert *alert.Alert,
	result *endpoint.Result,
	resolved bool,
) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}

	payload := provider.buildRequestPayload(
		cfg,
		ep,
		alert,
		result,
		resolved,
	)

	request, err := http.NewRequest(
		http.MethodPost,
		APIURL,
		strings.NewReader(payload.Encode()),
	)
	if err != nil {
		return err
	}

	request.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)

		return fmt.Errorf(
			"failed to send alert to Signalgrid: status code %d: %s",
			response.StatusCode,
			string(body),
		)
	}

	return nil
}

// buildRequestPayload builds the Signalgrid form payload
func (provider *AlertProvider) buildRequestPayload(
	cfg *Config,
	ep *endpoint.Endpoint,
	alert *alert.Alert,
	result *endpoint.Result,
	resolved bool,
) url.Values {
	var message string

	if resolved {
		message = fmt.Sprintf(
			"An alert for `%s` has been resolved after passing successfully %d time(s) in a row",
			ep.DisplayName(),
			alert.SuccessThreshold,
		)
	} else {
		message = fmt.Sprintf(
			"An alert for `%s` has been triggered due to having failed %d time(s) in a row",
			ep.DisplayName(),
			alert.FailureThreshold,
		)
	}

	if len(alert.GetDescription()) > 0 {
		message += " with the following description: " +
			alert.GetDescription()
	}

	for _, conditionResult := range result.ConditionResults {
		prefix := "✕"
		if conditionResult.Success {
			prefix = "✓"
		}

		message += fmt.Sprintf(
			"\n%s - %s",
			prefix,
			conditionResult.Condition,
		)
	}

	notificationType := "CRIT"
	critical := cfg.IsCritical()

	if resolved {
		notificationType = "SUCCESS"
		critical = false
	}

	return url.Values{
		"client_key": {cfg.ClientKey},
		"channel":    {cfg.Channel},
		"title":      {"Gatus: " + ep.DisplayName()},
		"body":       {message},
		"type":       {notificationType},
		"critical":   {strconv.FormatBool(critical)},
	}
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// GetConfig returns the configuration for the provider with overrides applied
func (provider *AlertProvider) GetConfig(
	group string,
	alert *alert.Alert,
) (*Config, error) {
	cfg := provider.DefaultConfig

	if len(alert.ProviderOverride) != 0 {
		overrideConfig := Config{}

		if err := yaml.Unmarshal(
			alert.ProviderOverrideAsBytes(),
			&overrideConfig,
		); err != nil {
			return nil, err
		}

		cfg.Merge(&overrideConfig)
	}

	err := cfg.Validate()

	return &cfg, err
}

// ValidateOverrides validates the alert's provider override
func (provider *AlertProvider) ValidateOverrides(
	group string,
	alert *alert.Alert,
) error {
	_, err := provider.GetConfig(group, alert)

	return err
}

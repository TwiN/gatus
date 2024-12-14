package twilio

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrSIDNotSet   = errors.New("sid not set")
	ErrTokenNotSet = errors.New("token not set")
	ErrFromNotSet  = errors.New("from not set")
	ErrToNotSet    = errors.New("to not set")
)

type Config struct {
	SID   string `yaml:"sid"`
	Token string `yaml:"token"`
	From  string `yaml:"from"`
	To    string `yaml:"to"`
}

func (cfg *Config) Validate() error {
	if len(cfg.SID) == 0 {
		return ErrSIDNotSet
	}
	if len(cfg.Token) == 0 {
		return ErrTokenNotSet
	}
	if len(cfg.From) == 0 {
		return ErrFromNotSet
	}
	if len(cfg.To) == 0 {
		return ErrToNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.SID) > 0 {
		cfg.SID = override.SID
	}
	if len(override.Token) > 0 {
		cfg.Token = override.Token
	}
	if len(override.From) > 0 {
		cfg.From = override.From
	}
	if len(override.To) > 0 {
		cfg.To = override.To
	}
}

// AlertProvider is the configuration necessary for sending an alert using Twilio
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
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer([]byte(provider.buildRequestBody(cfg, ep, alert, result, resolved)))
	request, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", cfg.SID), buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(cfg.SID+":"+cfg.Token))))
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

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) string {
	var message string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", ep.DisplayName(), alert.GetDescription())
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", ep.DisplayName(), alert.GetDescription())
	}
	return url.Values{
		"To":   {cfg.To},
		"From": {cfg.From},
		"Body": {message},
	}.Encode()
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// GetConfig returns the configuration for the provider with the overrides applied
func (provider *AlertProvider) GetConfig(group string, alert *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig
	// Handle alert overrides
	if len(alert.Override) != 0 {
		overrideConfig := Config{}
		if err := yaml.Unmarshal(alert.Override, &overrideConfig); err != nil {
			return nil, err
		}
		cfg.Merge(&overrideConfig)
	}
	// Validate the configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

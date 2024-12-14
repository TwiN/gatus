package pushover

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
	restAPIURL      = "https://api.pushover.net/1/messages.json"
	defaultPriority = 0
)

var (
	ErrInvalidApplicationToken = errors.New("application-token must be 30 characters long")
	ErrInvalidUserKey          = errors.New("user-key must be 30 characters long")
	ErrInvalidPriority         = errors.New("priority and resolved-priority must be between -2 and 2")
)

type Config struct {
	// Key used to authenticate the application sending
	// See "Your Applications" on the dashboard, or add a new one: https://pushover.net/apps/build
	ApplicationToken string `yaml:"application-token"`

	// Key of the user or group the messages should be sent to
	UserKey string `yaml:"user-key"`

	// The title of your message, likely the application name
	// default: the name of your application in Pushover
	Title string `yaml:"title,omitempty"`

	// Priority of all messages, ranging from -2 (very low) to 2 (Emergency)
	// default: 0
	Priority int `yaml:"priority,omitempty"`

	// Priority of resolved messages, ranging from -2 (very low) to 2 (Emergency)
	// default: 0
	ResolvedPriority int `yaml:"resolved-priority,omitempty"`

	// Sound of the messages (see: https://pushover.net/api#sounds)
	// default: "" (pushover)
	Sound string `yaml:"sound,omitempty"`
}

func (cfg *Config) Validate() error {
	if cfg.Priority == 0 {
		cfg.Priority = defaultPriority
	}
	if cfg.ResolvedPriority == 0 {
		cfg.ResolvedPriority = defaultPriority
	}
	if len(cfg.ApplicationToken) != 30 {
		return ErrInvalidApplicationToken
	}
	if len(cfg.UserKey) != 30 {
		return ErrInvalidUserKey
	}
	if cfg.Priority < -2 || cfg.Priority > 2 || cfg.ResolvedPriority < -2 || cfg.ResolvedPriority > 2 {
		return ErrInvalidPriority
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.ApplicationToken) > 0 {
		cfg.ApplicationToken = override.ApplicationToken
	}
	if len(override.UserKey) > 0 {
		cfg.UserKey = override.UserKey
	}
	if len(override.Title) > 0 {
		cfg.Title = override.Title
	}
	if override.Priority != 0 {
		cfg.Priority = override.Priority
	}
	if override.ResolvedPriority != 0 {
		cfg.ResolvedPriority = override.ResolvedPriority
	}
	if len(override.Sound) > 0 {
		cfg.Sound = override.Sound
	}
}

// AlertProvider is the configuration necessary for sending an alert using Pushover
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
// Reference doc for pushover: https://pushover.net/api
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(provider.buildRequestBody(cfg, ep, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, restAPIURL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
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

type Body struct {
	Token    string `json:"token"`
	User     string `json:"user"`
	Title    string `json:"title,omitempty"`
	Message  string `json:"message"`
	Priority int    `json:"priority"`
	Sound    string `json:"sound,omitempty"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var message string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", ep.DisplayName(), alert.GetDescription())
	} else {
		message = fmt.Sprintf("TRIGGERED: %s - %s", ep.DisplayName(), alert.GetDescription())
	}
	priority := cfg.Priority
	if resolved {
		priority = cfg.ResolvedPriority
	}
	body, _ := json.Marshal(Body{
		Token:    cfg.ApplicationToken,
		User:     cfg.UserKey,
		Title:    cfg.Title,
		Message:  message,
		Priority: priority,
		Sound:    cfg.Sound,
	})
	return body
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

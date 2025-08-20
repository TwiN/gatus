package email

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	gomail "gopkg.in/mail.v2"
	"gopkg.in/yaml.v3"
)

var (
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
	ErrMissingFromOrToFields  = errors.New("from and to fields are required")
	ErrInvalidPort            = errors.New("port must be between 1 and 65535 inclusively")
	ErrMissingHost            = errors.New("host is required")
)

type Config struct {
	From     string `yaml:"from"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	To       string `yaml:"to"`

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client,omitempty"`
}

func (cfg *Config) Validate() error {
	if len(cfg.From) == 0 || len(cfg.To) == 0 {
		return ErrMissingFromOrToFields
	}
	if cfg.Port < 1 || cfg.Port > math.MaxUint16 {
		return ErrInvalidPort
	}
	if len(cfg.Host) == 0 {
		return ErrMissingHost
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if override.ClientConfig != nil {
		cfg.ClientConfig = override.ClientConfig
	}
	if len(override.From) > 0 {
		cfg.From = override.From
	}
	if len(override.Username) > 0 {
		cfg.Username = override.Username
	}
	if len(override.Password) > 0 {
		cfg.Password = override.Password
	}
	if len(override.Host) > 0 {
		cfg.Host = override.Host
	}
	if override.Port > 0 {
		cfg.Port = override.Port
	}
	if len(override.To) > 0 {
		cfg.To = override.To
	}
}

// AlertProvider is the configuration necessary for sending an alert using SMTP
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
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" || len(override.To) == 0 {
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
	var username string
	if len(cfg.Username) > 0 {
		username = cfg.Username
	} else {
		username = cfg.From
	}
	subject, body := provider.buildMessageSubjectAndBody(ep, alert, result, resolved)
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", strings.Split(cfg.To, ",")...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)
	var d *gomail.Dialer
	if len(cfg.Password) == 0 {
		// Get the domain in the From address
		localName := "localhost"
		fromParts := strings.Split(cfg.From, `@`)
		if len(fromParts) == 2 {
			localName = fromParts[1]
		}
		// Create a dialer with no authentication
		d = &gomail.Dialer{Host: cfg.Host, Port: cfg.Port, LocalName: localName}
	} else {
		// Create an authenticated dialer
		d = gomail.NewDialer(cfg.Host, cfg.Port, username, cfg.Password)
	}
	if cfg.ClientConfig != nil && cfg.ClientConfig.Insecure {
		d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return d.DialAndSend(m)
}

// buildMessageSubjectAndBody builds the message subject and body
func (provider *AlertProvider) buildMessageSubjectAndBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) (string, string) {
	var subject, message string
	if resolved {
		subject = fmt.Sprintf("[%s] Alert resolved", ep.DisplayName())
		message = fmt.Sprintf("An alert for %s has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		subject = fmt.Sprintf("[%s] Alert triggered", ep.DisplayName())
		message = fmt.Sprintf("An alert for %s has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	var formattedConditionResults string
	if len(result.ConditionResults) > 0 {
		formattedConditionResults = "\n\nCondition results:\n"
		for _, conditionResult := range result.ConditionResults {
			var prefix string
			if conditionResult.Success {
				prefix = "✅"
			} else {
				prefix = "❌"
			}
			formattedConditionResults += fmt.Sprintf("%s %s\n", prefix, conditionResult.Condition)
		}
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = "\n\nAlert description: " + alertDescription
	}
	var extraLabels string
	if len(ep.ExtraLabels) > 0 {
		extraLabels = "\n\nExtra labels:\n"
		for key, value := range ep.ExtraLabels {
			extraLabels += fmt.Sprintf("  %s: %s\n", key, value)
		}
	}
	return subject, message + description + extraLabels + formattedConditionResults
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

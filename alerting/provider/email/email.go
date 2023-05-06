package email

import (
	"fmt"
	"math"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/core"
	gomail "gopkg.in/mail.v2"
)

// AlertProvider is the configuration necessary for sending an alert using SMTP
type AlertProvider struct {
	From     string `yaml:"from"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	To       string `yaml:"to"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

// Override is a case under which the default integration is overridden
type Override struct {
	Group string `yaml:"group"`
	To    string `yaml:"to"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	registeredGroups := make(map[string]bool)
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" || len(override.To) == 0 {
				return false
			}
			registeredGroups[override.Group] = true
		}
	}

	return len(provider.From) > 0 && len(provider.Password) > 0 && len(provider.Host) > 0 && len(provider.To) > 0 && provider.Port > 0 && provider.Port < math.MaxUint16
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	var username string
	if len(provider.Username) > 0 {
		username = provider.Username
	} else {
		username = provider.From
	}
	subject, body := provider.buildMessageSubjectAndBody(endpoint, alert, result, resolved)
	m := gomail.NewMessage()
	m.SetHeader("From", provider.From)
	m.SetHeader("To", strings.Split(provider.getToForGroup(endpoint.Group), ",")...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)
	d := gomail.NewDialer(provider.Host, provider.Port, username, provider.Password)
	return d.DialAndSend(m)
}

// buildMessageSubjectAndBody builds the message subject and body
func (provider *AlertProvider) buildMessageSubjectAndBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) (string, string) {
	var message, results string
	subject := fmt.Sprintf("[%s] Alert triggered", endpoint.DisplayName())
	if resolved {
		message = fmt.Sprintf("An alert for %s has been resolved after passing successfully %d time(s) in a row", endpoint.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for %s has been triggered due to having failed %d time(s) in a row", endpoint.DisplayName(), alert.FailureThreshold)
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "✅"
		} else {
			prefix = "❌"
		}
		results += fmt.Sprintf("%s %s\n", prefix, conditionResult.Condition)
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = "\n\nAlert description: " + alertDescription
	}
	return subject, message + description + "\n\nCondition results:\n" + results
}

// getToForGroup returns the appropriate email integration to for a given group
func (provider *AlertProvider) getToForGroup(group string) string {
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				return override.To
			}
		}
	}
	return provider.To
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

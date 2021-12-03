package email

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/core"
	gomail "gopkg.in/mail.v2"
)

// AlertProvider is the configuration necessary for sending an alert using SMTP
type AlertProvider struct {
	From     string `yaml:"from"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	To       string `yaml:"to"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.From) > 0 && len(provider.Password) > 0 && len(provider.Host) > 0 && len(provider.To) > 0 && provider.Port > 0 && provider.Port < math.MaxUint16
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	if os.Getenv("MOCK_ALERT_PROVIDER") == "true" {
		if os.Getenv("MOCK_ALERT_PROVIDER_ERROR") == "true" {
			return errors.New("error")
		}
		return nil
	}
	subject, body := provider.buildMessageSubjectAndBody(endpoint, alert, result, resolved)
	m := gomail.NewMessage()
	m.SetHeader("From", provider.From)
	m.SetHeader("To", strings.Split(provider.To, ",")...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)
	d := gomail.NewDialer(provider.Host, provider.Port, provider.From, provider.Password)
	return d.DialAndSend(m)
}

// buildMessageSubjectAndBody builds the message subject and body
func (provider *AlertProvider) buildMessageSubjectAndBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) (string, string) {
	var subject, message, results string
	if resolved {
		subject = fmt.Sprintf("[%s] Alert resolved", endpoint.Name)
		message = fmt.Sprintf("An alert for %s has been resolved after passing successfully %d time(s) in a row", endpoint.Name, alert.SuccessThreshold)
	} else {
		subject = fmt.Sprintf("[%s] Alert triggered", endpoint.Name)
		message = fmt.Sprintf("An alert for %s has been triggered due to having failed %d time(s) in a row", endpoint.Name, alert.FailureThreshold)
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

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

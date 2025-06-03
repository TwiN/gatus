package email

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	strip "github.com/grokify/html-strip-tags-go"
	gomail "gopkg.in/mail.v2"
	"gopkg.in/yaml.v3"
)

var (
	ErrDuplicateGroupOverride  = errors.New("duplicate group override")
	ErrMissingFromOrToFields   = errors.New("from and to fields are required")
	ErrInvalidPort             = errors.New("port must be between 1 and 65535 inclusively")
	ErrMissingHost             = errors.New("host is required")
	ErrInvalidEmailTemplateDir = errors.New("invalid email template directory: it must be a valid directory path that exists and is accessible")
)

const (
	EmailTemplateDirEnvVar = "GATUS_EMAIL_TEMPLATE_DIR" // Environment variable to specify the directory to the email templates
)

type Config struct {
	From     string `yaml:"from"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	To       string `yaml:"to"`

	TextEmailSubjectTriggered string `yaml:"text-email-subject-triggered,omitempty"` // String used in the email subject (optional)
	TextEmailSubjectResolved  string `yaml:"text-email-subject-resolved,omitempty"`  // String used in the email subject (optional)
	TextEmailBodyTriggered    string `yaml:"text-email-body-triggered,omitempty"`    // String used in the email body (optional)
	TextEmailBodyResolved     string `yaml:"text-email-body-resolved,omitempty"`     // String used in the email body (optional)
	FileEmailBodyTriggered    string `yaml:"file-email-body-triggered,omitempty"`    // HTML file used as template in the email body (optional)
	FileEmailBodyResolved     string `yaml:"file-email-body-resolved,omitempty"`     // HTML file used as template in the email body (optional)

	// EmailFileTemplateTriggered is the name of the template used for triggered alerts
	EmailFileTemplateTriggered string `yaml:"-"` // Value is set from the loaded template file
	// EmailFileTemplateResolved is the name of the template used for resolved alerts
	EmailFileTemplateResolved string `yaml:"-"` // Value is set from the loaded template file

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
	// Validate template directory if specified
	if templateDirectory := os.Getenv(EmailTemplateDirEnvVar); len(templateDirectory) > 0 {
		info, err := os.Stat(templateDirectory)
		if err != nil || os.IsNotExist(err) || !info.IsDir() {
			return ErrInvalidEmailTemplateDir
		}
		// Load the email templates from the directory
		if len(cfg.FileEmailBodyTriggered) > 0 {
			fileContentTriggered, err := os.ReadFile(fmt.Sprintf("%s/%s", templateDirectory, cfg.FileEmailBodyTriggered))
			if err == nil && len(fileContentTriggered) > 0 {
				cfg.EmailFileTemplateTriggered = string(fileContentTriggered)
			}
		}
		if len(cfg.FileEmailBodyResolved) > 0 {
			fileContentResolved, err := os.ReadFile(fmt.Sprintf("%s/%s", templateDirectory, cfg.FileEmailBodyResolved))
			if err == nil && len(fileContentResolved) > 0 {
				cfg.EmailFileTemplateResolved = string(fileContentResolved)
			}
		}
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
	if len(override.TextEmailSubjectTriggered) > 0 {
		cfg.TextEmailSubjectTriggered = override.TextEmailSubjectTriggered
	}
	if len(override.TextEmailSubjectResolved) > 0 {
		cfg.TextEmailSubjectResolved = override.TextEmailSubjectResolved
	}
	if len(override.TextEmailBodyTriggered) > 0 {
		cfg.TextEmailBodyTriggered = override.TextEmailBodyTriggered
	}
	if len(override.TextEmailBodyResolved) > 0 {
		cfg.TextEmailBodyResolved = override.TextEmailBodyResolved
	}
	if len(override.FileEmailBodyTriggered) > 0 {
		cfg.FileEmailBodyTriggered = override.FileEmailBodyTriggered
	}
	if len(override.FileEmailBodyResolved) > 0 {
		cfg.FileEmailBodyResolved = override.FileEmailBodyResolved
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
	subject, body_text, body_html := provider.buildMessageSubjectAndBody(cfg, ep, alert, result, resolved)
	m := gomail.NewMessage()
	m.SetHeader("From", cfg.From)
	m.SetHeader("To", strings.Split(cfg.To, ",")...)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body_text)
	if len(body_html) > 0 {
		m.AddAlternative("text/html", body_html)
	}
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
func (provider *AlertProvider) buildMessageSubjectAndBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) (string, string, string) {
	var subject, body_text, body_html string
	if resolved {
		subject = fmt.Sprintf("[%s] Alert resolved", ep.DisplayName())
		body_text = fmt.Sprintf("An alert for %s has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		subject = fmt.Sprintf("[%s] Alert triggered", ep.DisplayName())
		body_text = fmt.Sprintf("An alert for %s has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
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
	// Override subject and body if specified in the configuration
	if len(cfg.TextEmailSubjectTriggered) > 0 && !resolved {
		subject = cfg.TextEmailSubjectTriggered
		subject = strings.ReplaceAll(subject, "[ENDPOINT_NAME]", ep.Name)
		subject = strings.ReplaceAll(subject, "[ENDPOINT_GROUP]", ep.Group)
	}
	if len(cfg.TextEmailSubjectResolved) > 0 && resolved {
		subject = cfg.TextEmailSubjectResolved
		subject = strings.ReplaceAll(subject, "[ENDPOINT_NAME]", ep.Name)
		subject = strings.ReplaceAll(subject, "[ENDPOINT_GROUP]", ep.Group)
	}
	// If HTML template is not empty, use it as a template for the message body
	if len(cfg.EmailFileTemplateTriggered) > 0 && !resolved {
		body_html = provider.replaceBodyPlaceholders(ep, alert, cfg.EmailFileTemplateTriggered, strings.ReplaceAll(formattedConditionResults, "\n", "<br>"))
		body_text = strip.StripTags(body_html)
		return subject, body_text, body_html
	}
	if len(cfg.EmailFileTemplateResolved) > 0 && resolved {
		body_html = provider.replaceBodyPlaceholders(ep, alert, cfg.EmailFileTemplateResolved, strings.ReplaceAll(formattedConditionResults, "\n", "<br>"))
		body_text = strip.StripTags(body_html)
		return subject, body_text, body_html
	}
	// If no HTML file is specified, use the text overrides from configuration
	if len(cfg.TextEmailBodyTriggered) > 0 && !resolved {
		body_text = provider.replaceBodyPlaceholders(ep, alert, cfg.TextEmailBodyTriggered, formattedConditionResults)
		return subject, body_text, ""
	}
	if len(cfg.TextEmailBodyResolved) > 0 && resolved {
		body_text = provider.replaceBodyPlaceholders(ep, alert, cfg.TextEmailBodyResolved, formattedConditionResults)
		return subject, body_text, ""
	}
	// Fallback to the default message body if no overrides are specified
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = "\n\nAlert description: " + alertDescription
	}
	return subject, body_text + description + formattedConditionResults, body_html
}

func (provider *AlertProvider) replaceBodyPlaceholders(ep *endpoint.Endpoint, alert *alert.Alert, str string, formattedConditionResults string) string {
	str = strings.ReplaceAll(str, "[ENDPOINT_NAME]", ep.Name)
	str = strings.ReplaceAll(str, "[ENDPOINT_GROUP]", ep.Group)
	str = strings.ReplaceAll(str, "[ENDPOINT_URL]", ep.URL)
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		str = strings.ReplaceAll(str, "[ALERT_DESCRIPTION]", alertDescription)
	}
	str = strings.ReplaceAll(str, "[SUCCESS_THRESHOLD]", string(rune(alert.SuccessThreshold)))
	str = strings.ReplaceAll(str, "[FAILURE_THRESHOLD]", string(rune(alert.FailureThreshold)))
	str = strings.ReplaceAll(str, "[CONDITION_RESULTS]", formattedConditionResults)
	return str
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

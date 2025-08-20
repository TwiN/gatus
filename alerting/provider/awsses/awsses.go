package awsses

import (
	"errors"
	"fmt"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/logr"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"gopkg.in/yaml.v3"
)

const (
	CharSet = "UTF-8"
)

var (
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
	ErrMissingFromOrToFields  = errors.New("from and to fields are required")
	ErrInvalidAWSAuthConfig   = errors.New("either both or neither of access-key-id and secret-access-key must be specified")
)

type Config struct {
	AccessKeyID     string `yaml:"access-key-id"`
	SecretAccessKey string `yaml:"secret-access-key"`
	Region          string `yaml:"region"`

	From string `yaml:"from"`
	To   string `yaml:"to"`
}

func (cfg *Config) Validate() error {
	if len(cfg.From) == 0 || len(cfg.To) == 0 {
		return ErrMissingFromOrToFields
	}
	if !((len(cfg.AccessKeyID) == 0 && len(cfg.SecretAccessKey) == 0) || (len(cfg.AccessKeyID) > 0 && len(cfg.SecretAccessKey) > 0)) {
		// if both AccessKeyID and SecretAccessKey are specified, we'll use these to authenticate,
		// otherwise if neither are specified, then we'll fall back on IAM authentication.
		return ErrInvalidAWSAuthConfig
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.AccessKeyID) > 0 {
		cfg.AccessKeyID = override.AccessKeyID
	}
	if len(override.SecretAccessKey) > 0 {
		cfg.SecretAccessKey = override.SecretAccessKey
	}
	if len(override.Region) > 0 {
		cfg.Region = override.Region
	}
	if len(override.From) > 0 {
		cfg.From = override.From
	}
	if len(override.To) > 0 {
		cfg.To = override.To
	}
}

// AlertProvider is the configuration necessary for sending an alert using AWS Simple Email Service
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
	awsSession, err := provider.createSession(cfg)
	if err != nil {
		return err
	}
	svc := ses.New(awsSession)
	subject, body := provider.buildMessageSubjectAndBody(ep, alert, result, resolved)
	emails := strings.Split(cfg.To, ",")

	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: aws.StringSlice(emails),
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(cfg.From),
	}
	if _, err = svc.SendEmail(input); err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				logr.Error(ses.ErrCodeMessageRejected + ": " + aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				logr.Error(ses.ErrCodeMailFromDomainNotVerifiedException + ": " + aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				logr.Error(ses.ErrCodeConfigurationSetDoesNotExistException + ": " + aerr.Error())
			default:
				logr.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			logr.Error(err.Error())
		}

		return err
	}
	return nil
}

func (provider *AlertProvider) createSession(cfg *Config) (*session.Session, error) {
	awsConfig := &aws.Config{
		Region: aws.String(cfg.Region),
	}
	if len(cfg.AccessKeyID) > 0 && len(cfg.SecretAccessKey) > 0 {
		awsConfig.Credentials = credentials.NewStaticCredentials(cfg.AccessKeyID, cfg.SecretAccessKey, "")
	}
	return session.NewSession(awsConfig)
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
	return subject, message + description + formattedConditionResults
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

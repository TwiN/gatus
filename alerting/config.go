package alerting

import (
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/alerting/provider/pagerduty"
	"github.com/TwinProduction/gatus/alerting/provider/slack"
	"github.com/TwinProduction/gatus/alerting/provider/twilio"
)

// Config is the configuration for alerting providers
type Config struct {
	// Slack is the configuration for the slack alerting provider
	Slack *slack.AlertProvider `yaml:"slack"`

	// Pagerduty is the configuration for the pagerduty alerting provider
	PagerDuty *pagerduty.AlertProvider `yaml:"pagerduty"`

	// Twilio is the configuration for the twilio alerting provider
	Twilio *twilio.AlertProvider `yaml:"twilio"`

	// Custom is the configuration for the custom alerting provider
	Custom *custom.AlertProvider `yaml:"custom"`
}

package alerting

import (
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/alerting/provider/pagerduty"
	"github.com/TwinProduction/gatus/alerting/provider/slack"
	"github.com/TwinProduction/gatus/alerting/provider/twilio"
)

type Config struct {
	Slack     *slack.AlertProvider     `yaml:"slack"`
	PagerDuty *pagerduty.AlertProvider `yaml:"pagerduty"`
	Twilio    *twilio.AlertProvider    `yaml:"twilio"`
	Custom    *custom.AlertProvider    `yaml:"custom"`
}

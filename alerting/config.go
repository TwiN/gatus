package alerting

import (
	"github.com/Meldiron/gatus/alerting/provider/custom"
	"github.com/Meldiron/gatus/alerting/provider/discord"
	"github.com/Meldiron/gatus/alerting/provider/mattermost"
	"github.com/Meldiron/gatus/alerting/provider/messagebird"
	"github.com/Meldiron/gatus/alerting/provider/pagerduty"
	"github.com/Meldiron/gatus/alerting/provider/slack"
	"github.com/Meldiron/gatus/alerting/provider/telegram"
	"github.com/Meldiron/gatus/alerting/provider/twilio"
)

// Config is the configuration for alerting providers
type Config struct {
	// Custom is the configuration for the custom alerting provider
	Custom *custom.AlertProvider `yaml:"custom"`

	// Discord is the configuration for the discord alerting provider
	Discord *discord.AlertProvider `yaml:"discord"`

	// Mattermost is the configuration for the mattermost alerting provider
	Mattermost *mattermost.AlertProvider `yaml:"mattermost"`

	// Messagebird is the configuration for the messagebird alerting provider
	Messagebird *messagebird.AlertProvider `yaml:"messagebird"`

	// PagerDuty is the configuration for the pagerduty alerting provider
	PagerDuty *pagerduty.AlertProvider `yaml:"pagerduty"`

	// Slack is the configuration for the slack alerting provider
	Slack *slack.AlertProvider `yaml:"slack"`

	// Telegram is the configuration for the telegram alerting provider
	Telegram *telegram.AlertProvider `yaml:"telegram"`

	// Twilio is the configuration for the twilio alerting provider
	Twilio *twilio.AlertProvider `yaml:"twilio"`
}

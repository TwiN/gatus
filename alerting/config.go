package alerting

import (
	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/alerting/provider"
	"github.com/TwiN/gatus/v3/alerting/provider/custom"
	"github.com/TwiN/gatus/v3/alerting/provider/discord"
	"github.com/TwiN/gatus/v3/alerting/provider/email"
	"github.com/TwiN/gatus/v3/alerting/provider/mattermost"
	"github.com/TwiN/gatus/v3/alerting/provider/messagebird"
	"github.com/TwiN/gatus/v3/alerting/provider/pagerduty"
	"github.com/TwiN/gatus/v3/alerting/provider/slack"
	"github.com/TwiN/gatus/v3/alerting/provider/teams"
	"github.com/TwiN/gatus/v3/alerting/provider/telegram"
	"github.com/TwiN/gatus/v3/alerting/provider/twilio"
)

// Config is the configuration for alerting providers
type Config struct {
	// Custom is the configuration for the custom alerting provider
	Custom *custom.AlertProvider `yaml:"custom,omitempty"`

	// Discord is the configuration for the discord alerting provider
	Discord *discord.AlertProvider `yaml:"discord,omitempty"`

	// Email is the configuration for the email alerting provider
	Email *email.AlertProvider `yaml:"email,omitempty"`

	// Mattermost is the configuration for the mattermost alerting provider
	Mattermost *mattermost.AlertProvider `yaml:"mattermost,omitempty"`

	// Messagebird is the configuration for the messagebird alerting provider
	Messagebird *messagebird.AlertProvider `yaml:"messagebird,omitempty"`

	// PagerDuty is the configuration for the pagerduty alerting provider
	PagerDuty *pagerduty.AlertProvider `yaml:"pagerduty,omitempty"`

	// Slack is the configuration for the slack alerting provider
	Slack *slack.AlertProvider `yaml:"slack,omitempty"`

	// Teams is the configuration for the teams alerting provider
	Teams *teams.AlertProvider `yaml:"teams,omitempty"`

	// Telegram is the configuration for the telegram alerting provider
	Telegram *telegram.AlertProvider `yaml:"telegram,omitempty"`

	// Twilio is the configuration for the twilio alerting provider
	Twilio *twilio.AlertProvider `yaml:"twilio,omitempty"`
}

// GetAlertingProviderByAlertType returns an provider.AlertProvider by its corresponding alert.Type
func (config Config) GetAlertingProviderByAlertType(alertType alert.Type) provider.AlertProvider {
	switch alertType {
	case alert.TypeCustom:
		if config.Custom == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Custom
	case alert.TypeDiscord:
		if config.Discord == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Discord
	case alert.TypeEmail:
		if config.Email == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Email
	case alert.TypeMattermost:
		if config.Mattermost == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Mattermost
	case alert.TypeMessagebird:
		if config.Messagebird == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Messagebird
	case alert.TypePagerDuty:
		if config.PagerDuty == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.PagerDuty
	case alert.TypeSlack:
		if config.Slack == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Slack
	case alert.TypeTeams:
		if config.Teams == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Teams
	case alert.TypeTelegram:
		if config.Telegram == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Telegram
	case alert.TypeTwilio:
		if config.Twilio == nil {
			// Since we're returning an interface, we need to explicitly return nil, even if the provider itself is nil
			return nil
		}
		return config.Twilio
	}
	return nil
}

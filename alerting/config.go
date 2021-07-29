package alerting

import (
	"github.com/TwinProduction/gatus/alerting/alert"
	"github.com/TwinProduction/gatus/alerting/provider"
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/alerting/provider/discord"
	"github.com/TwinProduction/gatus/alerting/provider/mattermost"
	"github.com/TwinProduction/gatus/alerting/provider/messagebird"
	"github.com/TwinProduction/gatus/alerting/provider/pagerduty"
	"github.com/TwinProduction/gatus/alerting/provider/slack"
	"github.com/TwinProduction/gatus/alerting/provider/teams"
	"github.com/TwinProduction/gatus/alerting/provider/telegram"
	"github.com/TwinProduction/gatus/alerting/provider/twilio"
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

	// Teams is the configuration for the teams alerting provider
	Teams *teams.AlertProvider `yaml:"teams"`

	// Telegram is the configuration for the telegram alerting provider
	Telegram *telegram.AlertProvider `yaml:"telegram"`

	// Twilio is the configuration for the twilio alerting provider
	Twilio *twilio.AlertProvider `yaml:"twilio"`
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

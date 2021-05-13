package provider

import (
	"github.com/Meldiron/gatus/alerting/provider/custom"
	"github.com/Meldiron/gatus/alerting/provider/discord"
	"github.com/Meldiron/gatus/alerting/provider/mattermost"
	"github.com/Meldiron/gatus/alerting/provider/messagebird"
	"github.com/Meldiron/gatus/alerting/provider/pagerduty"
	"github.com/Meldiron/gatus/alerting/provider/slack"
	"github.com/Meldiron/gatus/alerting/provider/telegram"
	"github.com/Meldiron/gatus/alerting/provider/twilio"
	"github.com/Meldiron/gatus/core"
)

// AlertProvider is the interface that each providers should implement
type AlertProvider interface {
	// IsValid returns whether the provider's configuration is valid
	IsValid() bool

	// ToCustomAlertProvider converts the provider into a custom.AlertProvider
	ToCustomAlertProvider(service *core.Service, alert *core.Alert, result *core.Result, resolved bool) *custom.AlertProvider
}

var (
	// Validate interface implementation on compile
	_ AlertProvider = (*custom.AlertProvider)(nil)
	_ AlertProvider = (*discord.AlertProvider)(nil)
	_ AlertProvider = (*mattermost.AlertProvider)(nil)
	_ AlertProvider = (*messagebird.AlertProvider)(nil)
	_ AlertProvider = (*pagerduty.AlertProvider)(nil)
	_ AlertProvider = (*slack.AlertProvider)(nil)
	_ AlertProvider = (*telegram.AlertProvider)(nil)
	_ AlertProvider = (*twilio.AlertProvider)(nil)
)

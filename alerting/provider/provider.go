package provider

import (
	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/alerting/provider/custom"
	"github.com/TwiN/gatus/v3/alerting/provider/discord"
	"github.com/TwiN/gatus/v3/alerting/provider/mattermost"
	"github.com/TwiN/gatus/v3/alerting/provider/messagebird"
	"github.com/TwiN/gatus/v3/alerting/provider/pagerduty"
	"github.com/TwiN/gatus/v3/alerting/provider/slack"
	"github.com/TwiN/gatus/v3/alerting/provider/teams"
	"github.com/TwiN/gatus/v3/alerting/provider/telegram"
	"github.com/TwiN/gatus/v3/alerting/provider/twilio"
	"github.com/TwiN/gatus/v3/core"
)

// AlertProvider is the interface that each providers should implement
type AlertProvider interface {
	// IsValid returns whether the provider's configuration is valid
	IsValid() bool

	// ToCustomAlertProvider converts the provider into a custom.AlertProvider
	ToCustomAlertProvider(service *core.Service, alert *alert.Alert, result *core.Result, resolved bool) *custom.AlertProvider

	// GetDefaultAlert returns the provider's default alert configuration
	GetDefaultAlert() *alert.Alert
}

// ParseWithDefaultAlert parses a service alert by using the provider's default alert as a baseline
func ParseWithDefaultAlert(providerDefaultAlert, serviceAlert *alert.Alert) {
	if providerDefaultAlert == nil || serviceAlert == nil {
		return
	}
	if serviceAlert.Enabled == nil {
		serviceAlert.Enabled = providerDefaultAlert.Enabled
	}
	if serviceAlert.SendOnResolved == nil {
		serviceAlert.SendOnResolved = providerDefaultAlert.SendOnResolved
	}
	if serviceAlert.Description == nil {
		serviceAlert.Description = providerDefaultAlert.Description
	}
	if serviceAlert.FailureThreshold == 0 {
		serviceAlert.FailureThreshold = providerDefaultAlert.FailureThreshold
	}
	if serviceAlert.SuccessThreshold == 0 {
		serviceAlert.SuccessThreshold = providerDefaultAlert.SuccessThreshold
	}
}

var (
	// Validate interface implementation on compile
	_ AlertProvider = (*custom.AlertProvider)(nil)
	_ AlertProvider = (*discord.AlertProvider)(nil)
	_ AlertProvider = (*mattermost.AlertProvider)(nil)
	_ AlertProvider = (*messagebird.AlertProvider)(nil)
	_ AlertProvider = (*pagerduty.AlertProvider)(nil)
	_ AlertProvider = (*slack.AlertProvider)(nil)
	_ AlertProvider = (*teams.AlertProvider)(nil)
	_ AlertProvider = (*telegram.AlertProvider)(nil)
	_ AlertProvider = (*twilio.AlertProvider)(nil)
)

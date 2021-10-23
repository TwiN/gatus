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
	ToCustomAlertProvider(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) *custom.AlertProvider

	// GetDefaultAlert returns the provider's default alert configuration
	GetDefaultAlert() *alert.Alert
}

// ParseWithDefaultAlert parses an Endpoint alert by using the provider's default alert as a baseline
func ParseWithDefaultAlert(providerDefaultAlert, endpointAlert *alert.Alert) {
	if providerDefaultAlert == nil || endpointAlert == nil {
		return
	}
	if endpointAlert.Enabled == nil {
		endpointAlert.Enabled = providerDefaultAlert.Enabled
	}
	if endpointAlert.SendOnResolved == nil {
		endpointAlert.SendOnResolved = providerDefaultAlert.SendOnResolved
	}
	if endpointAlert.Description == nil {
		endpointAlert.Description = providerDefaultAlert.Description
	}
	if endpointAlert.FailureThreshold == 0 {
		endpointAlert.FailureThreshold = providerDefaultAlert.FailureThreshold
	}
	if endpointAlert.SuccessThreshold == 0 {
		endpointAlert.SuccessThreshold = providerDefaultAlert.SuccessThreshold
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

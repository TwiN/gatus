package provider

import (
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/awsses"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/alerting/provider/discord"
	"github.com/TwiN/gatus/v5/alerting/provider/email"
	"github.com/TwiN/gatus/v5/alerting/provider/github"
	"github.com/TwiN/gatus/v5/alerting/provider/gitlab"
	"github.com/TwiN/gatus/v5/alerting/provider/googlechat"
	"github.com/TwiN/gatus/v5/alerting/provider/matrix"
	"github.com/TwiN/gatus/v5/alerting/provider/mattermost"
	"github.com/TwiN/gatus/v5/alerting/provider/messagebird"
	"github.com/TwiN/gatus/v5/alerting/provider/ntfy"
	"github.com/TwiN/gatus/v5/alerting/provider/opsgenie"
	"github.com/TwiN/gatus/v5/alerting/provider/pagerduty"
	"github.com/TwiN/gatus/v5/alerting/provider/pushover"
	"github.com/TwiN/gatus/v5/alerting/provider/slack"
	"github.com/TwiN/gatus/v5/alerting/provider/teams"
	"github.com/TwiN/gatus/v5/alerting/provider/telegram"
	"github.com/TwiN/gatus/v5/alerting/provider/twilio"
	"github.com/TwiN/gatus/v5/core"
)

// AlertProvider is the interface that each providers should implement
type AlertProvider interface {
	// IsValid returns whether the provider's configuration is valid
	IsValid() bool

	// GetDefaultAlert returns the provider's default alert configuration
	GetDefaultAlert() *alert.Alert

	// Send an alert using the provider
	Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error
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
	_ AlertProvider = (*awsses.AlertProvider)(nil)
	_ AlertProvider = (*custom.AlertProvider)(nil)
	_ AlertProvider = (*discord.AlertProvider)(nil)
	_ AlertProvider = (*email.AlertProvider)(nil)
	_ AlertProvider = (*github.AlertProvider)(nil)
	_ AlertProvider = (*gitlab.AlertProvider)(nil)
	_ AlertProvider = (*googlechat.AlertProvider)(nil)
	_ AlertProvider = (*matrix.AlertProvider)(nil)
	_ AlertProvider = (*mattermost.AlertProvider)(nil)
	_ AlertProvider = (*messagebird.AlertProvider)(nil)
	_ AlertProvider = (*ntfy.AlertProvider)(nil)
	_ AlertProvider = (*opsgenie.AlertProvider)(nil)
	_ AlertProvider = (*pagerduty.AlertProvider)(nil)
	_ AlertProvider = (*pushover.AlertProvider)(nil)
	_ AlertProvider = (*slack.AlertProvider)(nil)
	_ AlertProvider = (*teams.AlertProvider)(nil)
	_ AlertProvider = (*telegram.AlertProvider)(nil)
	_ AlertProvider = (*twilio.AlertProvider)(nil)
)

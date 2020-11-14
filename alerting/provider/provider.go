package provider

import (
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/alerting/provider/pagerduty"
	"github.com/TwinProduction/gatus/alerting/provider/slack"
	"github.com/TwinProduction/gatus/alerting/provider/mattermost"
	"github.com/TwinProduction/gatus/alerting/provider/twilio"
	"github.com/TwinProduction/gatus/core"
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
	_ AlertProvider = (*twilio.AlertProvider)(nil)
	_ AlertProvider = (*slack.AlertProvider)(nil)
	_ AlertProvider = (*mattermost.AlertProvider)(nil)
	_ AlertProvider = (*pagerduty.AlertProvider)(nil)
)

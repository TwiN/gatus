package core

// Alert is the service's alert configuration
type Alert struct {
	// Type of alert
	Type AlertType `yaml:"type"`

	// Enabled defines whether or not the alert is enabled
	Enabled bool `yaml:"enabled"`

	// Threshold is the number of failures in a row needed before triggering the alert
	Threshold int `yaml:"threshold"`

	// Description of the alert. Will be included in the alert sent.
	Description string `yaml:"description"`

	// SendOnResolved defines whether to send a second notification when the issue has been resolved
	SendOnResolved bool `yaml:"send-on-resolved"`

	// SuccessBeforeResolved defines whether to send a second notification when the issue has been resolved
	SuccessBeforeResolved int `yaml:"success-before-resolved"`

	// ResolveKey is an optional field that is used by some providers (i.e. PagerDuty's dedup_key) to resolve
	// ongoing/triggered incidents
	ResolveKey string

	// Triggered is used to determine whether an alert has been triggered. When an alert is resolved, this value
	// should be set back to false. It is used to prevent the same alert from going out twice.
	Triggered bool
}

type AlertType string

const (
	SlackAlert     AlertType = "slack"
	PagerDutyAlert AlertType = "pagerduty"
	TwilioAlert    AlertType = "twilio"
	CustomAlert    AlertType = "custom"
)

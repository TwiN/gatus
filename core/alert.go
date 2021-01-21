package core

// Alert is the service's alert configuration
type Alert struct {
	// Type of alert
	Type AlertType `yaml:"type"`

	// Enabled defines whether or not the alert is enabled
	Enabled bool `yaml:"enabled"`

	// FailureThreshold is the number of failures in a row needed before triggering the alert
	FailureThreshold int `yaml:"failure-threshold"`

	// Description of the alert. Will be included in the alert sent.
	Description string `yaml:"description"`

	// SendOnResolved defines whether to send a second notification when the issue has been resolved
	SendOnResolved bool `yaml:"send-on-resolved"`

	// SuccessThreshold defines how many successful executions must happen in a row before an ongoing incident is marked as resolved
	SuccessThreshold int `yaml:"success-threshold"`

	// ResolveKey is an optional field that is used by some providers (i.e. PagerDuty's dedup_key) to resolve
	// ongoing/triggered incidents
	ResolveKey string

	// Triggered is used to determine whether an alert has been triggered. When an alert is resolved, this value
	// should be set back to false. It is used to prevent the same alert from going out twice.
	//
	// This value should only be modified if the provider.AlertProvider's Send function does not return an error for an
	// alert that hasn't been triggered yet. This doubles as a lazy retry. The reason why this behavior isn't also
	// applied for alerts that are already triggered and has become "healthy" again is to prevent a case where, for
	// some reason, the alert provider always returns errors when trying to send the resolved notification
	// (SendOnResolved).
	Triggered bool
}

// AlertType is the type of the alert.
// The value will generally be the name of the alert provider
type AlertType string

const (
	// SlackAlert is the AlertType for the slack alerting provider
	SlackAlert AlertType = "slack"

	// MattermostAlert is the AlertType for the mattermost alerting provider
	MattermostAlert AlertType = "mattermost"

	// MessagebirdAlert is the AlertType for the messagebird alerting provider
	MessagebirdAlert AlertType = "messagebird"

	// PagerDutyAlert is the AlertType for the pagerduty alerting provider
	PagerDutyAlert AlertType = "pagerduty"

	// TwilioAlert is the AlertType for the twilio alerting provider
	TwilioAlert AlertType = "twilio"

	// CustomAlert is the AlertType for the custom alerting provider
	CustomAlert AlertType = "custom"
)

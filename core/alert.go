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
}

type AlertType string

const (
	SlackAlert  AlertType = "slack"
	CustomAlert AlertType = "custom"
)

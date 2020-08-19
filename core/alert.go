package core

type Alert struct {
	Type        AlertType `yaml:"type"`
	Enabled     bool      `yaml:"enabled"`
	Threshold   int       `yaml:"threshold"`
	Description string    `yaml:"description"`
}

type AlertType string

const (
	SlackAlert AlertType = "slack"
)

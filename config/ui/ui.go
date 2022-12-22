package ui

import (
	"bytes"
	"errors"
	"html/template"

	static "github.com/TwiN/gatus/v5/web"
)

const (
	defaultTitle       = "Health Dashboard | Gatus"
	defaultDescription = "Gatus is an advanced automated status page that lets you monitor your applications and configure alerts to notify you if there's an issue"
	defaultHeader      = "Health Status"
	defaultLogo        = ""
	defaultLink        = ""
)

var (
	ErrButtonValidationFailed = errors.New("invalid button configuration: missing required name or link")
)

// Config is the configuration for the UI of Gatus
type Config struct {
	Title       string   `yaml:"title,omitempty"`       // Title of the page
	Description string   `yaml:"description,omitempty"` // Meta description of the page
	Header      string   `yaml:"header,omitempty"`      // Header is the text at the top of the page
	Logo        string   `yaml:"logo,omitempty"`        // Logo to display on the page
	Link        string   `yaml:"link,omitempty"`        // Link to open when clicking on the logo
	Buttons     []Button `yaml:"buttons,omitempty"`     // Buttons to display below the header
}

// Button is the configuration for a button on the UI
type Button struct {
	Name string `yaml:"name,omitempty"` // Name is the text to display on the button
	Link string `yaml:"link,omitempty"` // Link to open when the button is clicked.
}

// Validate validates the button configuration
func (btn *Button) Validate() error {
	if len(btn.Name) == 0 || len(btn.Link) == 0 {
		return ErrButtonValidationFailed
	}
	return nil
}

// GetDefaultConfig returns a Config struct with the default values
func GetDefaultConfig() *Config {
	return &Config{
		Title:       defaultTitle,
		Description: defaultDescription,
		Header:      defaultHeader,
		Logo:        defaultLogo,
		Link:        defaultLink,
	}
}

// ValidateAndSetDefaults validates the UI configuration and sets the default values if necessary.
func (cfg *Config) ValidateAndSetDefaults() error {
	if len(cfg.Title) == 0 {
		cfg.Title = defaultTitle
	}
	if len(cfg.Description) == 0 {
		cfg.Description = defaultDescription
	}
	if len(cfg.Header) == 0 {
		cfg.Header = defaultHeader
	}
	if len(cfg.Header) == 0 {
		cfg.Header = defaultLink
	}
	for _, btn := range cfg.Buttons {
		if err := btn.Validate(); err != nil {
			return err
		}
	}
	// Validate that the template works
	t, err := template.ParseFS(static.FileSystem, static.IndexPath)
	if err != nil {
		return err
	}
	var buffer bytes.Buffer
	err = t.Execute(&buffer, cfg)
	if err != nil {
		return err
	}
	return nil
}

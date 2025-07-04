package ui

import (
	"bytes"
	"errors"
	"html/template"

	"github.com/TwiN/gatus/v5/storage"
	static "github.com/TwiN/gatus/v5/web"
)

const (
	defaultTitle       = "Health Dashboard | Gatus"
	defaultDescription = "Gatus is an advanced automated status page that lets you monitor your applications and configure alerts to notify you if there's an issue"
	defaultHeader      = "Health Status"
	defaultLogo        = ""
	defaultLink        = ""
	defaultCustomCSS   = ""
)

var (
	defaultDarkMode = true

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
	CustomCSS   string   `yaml:"custom-css,omitempty"`  // Custom CSS to include in the page
	DarkMode    *bool    `yaml:"dark-mode,omitempty"`   // DarkMode is a flag to enable dark mode by default

	//////////////////////////////////////////////
	// Non-configurable - used for UI rendering //
	//////////////////////////////////////////////
	MaximumNumberOfResults int `yaml:"-"` // MaximumNumberOfResults to display on the page, it's not configurable because we're passing it from the storage config
}

func (cfg *Config) IsDarkMode() bool {
	if cfg.DarkMode != nil {
		return *cfg.DarkMode
	}
	return defaultDarkMode
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
		Title:                  defaultTitle,
		Description:            defaultDescription,
		Header:                 defaultHeader,
		Logo:                   defaultLogo,
		Link:                   defaultLink,
		CustomCSS:              defaultCustomCSS,
		DarkMode:               &defaultDarkMode,
		MaximumNumberOfResults: storage.DefaultMaximumNumberOfResults,
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
	if len(cfg.Logo) == 0 {
		cfg.Logo = defaultLogo
	}
	if len(cfg.Link) == 0 {
		cfg.Link = defaultLink
	}
	if len(cfg.CustomCSS) == 0 {
		cfg.CustomCSS = defaultCustomCSS
	}
	if cfg.DarkMode == nil {
		cfg.DarkMode = &defaultDarkMode
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
	return t.Execute(&buffer, ViewData{UI: cfg, Theme: "dark"})
}

type ViewData struct {
	UI    *Config
	Theme string
}

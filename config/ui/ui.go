package ui

import (
	"bytes"
	"html/template"

	"github.com/TwiN/gatus/v3/web"
)

const (
	defaultTitle = "Health Dashboard | Gatus"
	defaultLogo  = ""
)

// Config is the configuration for the UI of Gatus
type Config struct {
	Title string `yaml:"title"` // Title of the page
	Logo  string `yaml:"logo"`  // Logo to display on the page
}

// GetDefaultConfig returns a Config struct with the default values
func GetDefaultConfig() *Config {
	return &Config{
		Title: defaultTitle,
		Logo:  defaultLogo,
	}
}

// ValidateAndSetDefaults validates the UI configuration and sets the default values if necessary.
func (cfg *Config) ValidateAndSetDefaults() error {
	if len(cfg.Title) == 0 {
		cfg.Title = defaultTitle
	}
	t, err := template.ParseFS(web.StaticFolder, "index.html")
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

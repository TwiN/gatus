package config

import (
	"bytes"
	"html/template"
)

const (
	defaultTitle = "Health Dashboard | Gatus"
	defaultLogo  = ""
)

// UIConfig is the configuration for the UI of Gatus
type UIConfig struct {
	Title string `yaml:"title"` // Title of the page
	Logo  string `yaml:"logo"`  // Logo to display on the page
}

// GetDefaultUIConfig returns a UIConfig struct with the default values
func GetDefaultUIConfig() *UIConfig {
	return &UIConfig{
		Title: defaultTitle,
		Logo:  defaultLogo,
	}
}

func (cfg *UIConfig) validateAndSetDefaults() error {
	if len(cfg.Title) == 0 {
		cfg.Title = defaultTitle
	}
	t, err := template.ParseFiles(StaticFolder + "/index.html")
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

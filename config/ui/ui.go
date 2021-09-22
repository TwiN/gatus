package ui

import (
	"bytes"
	"html/template"
)

const (
	defaultTitle = "Health Dashboard | Gatus"
	defaultLogo  = ""
)

var (
	// StaticFolder is the path to the location of the static folder from the root path of the project
	// The only reason this is exposed is to allow running tests from a different path than the root path of the project
	StaticFolder = "./web/static"
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

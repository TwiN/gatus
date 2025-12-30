package theme

import (
	"errors"

	"github.com/TwiN/gatus/v5/config/state"
)

var (
	ErrInvalidColorHexCode = errors.New("invalid color hex code: must be in the format #RRGGBB")
)

type Color string

func (color Color) Validate() error {
	if !IsValidColorHexCode(color) {
		return ErrInvalidColorHexCode
	}
	return nil
}

func IsValidColorHexCode(color Color) bool {
	if len(color) != 7 || color[0] != '#' {
		return false
	}
	for _, char := range color[1:] {
		if (char < '0' || char > '9') && (char < 'A' || char > 'F') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func GetDefaultColors() map[string]Color {
	return map[string]Color{
		state.DefaultHealthyStateName:     "#22C55E", // Green
		state.DefaultUnhealthyStateName:   "#E43B3C", // Red (Default for result bar before was "#EF4444 saw #AD0116 on GitHub (was too dark) so I used https://colordesigner.io/gradient-generator to use some color in between TODO#227 Change to darker red for better visibility good?)
		state.DefaultMaintenanceStateName: "#3B82F6", // Blue
	}
}

type Config struct {
	StateColors map[string]Color `yaml:"state-colors" json:"stateColors"` // StateColors is a map of state names to their corresponding colors
}

func GetDefaultConfig() *Config {
	return &Config{
		StateColors: GetDefaultColors(),
	}
}

func (cfg *Config) ValidateAndSetDefaults() error {
	if len(cfg.StateColors) == 0 {
		cfg.StateColors = GetDefaultColors()
	} else {
		// Validate provided colors
		for stateName, color := range cfg.StateColors {
			if err := color.Validate(); err != nil {
				return errors.New("invalid color for state '" + stateName + "': " + err.Error())
			}
		}
		// Set defaults for any missing state colors
		defaultColors := GetDefaultColors()
		for stateName, defaultColor := range defaultColors {
			if _, exists := cfg.StateColors[stateName]; !exists {
				cfg.StateColors[stateName] = defaultColor
			}
		}
	}
	return nil
}

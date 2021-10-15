package ui

import (
	"testing"
)

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	cfg := &Config{Title: ""}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Error("expected no error, got", err.Error())
	}
}

func TestGetDefaultConfig(t *testing.T) {
	defaultConfig := GetDefaultConfig()
	if defaultConfig.Title != defaultTitle {
		t.Error("expected GetDefaultConfig() to return defaultTitle, got", defaultConfig.Title)
	}
	if defaultConfig.Logo != defaultLogo {
		t.Error("expected GetDefaultConfig() to return defaultLogo, got", defaultConfig.Logo)
	}
}

package ui

import (
	"testing"
)

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	StaticFolder = "../../web/static"
	defer func() {
		StaticFolder = "./web/static"
	}()
	cfg := &Config{
		Title:  "",
		Header: "",
		Logo:   "",
		Link:   "",
	}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if cfg.Title != defaultTitle {
		t.Errorf("expected title to be %s, got %s", defaultTitle, cfg.Title)
	}
	if cfg.Header != defaultHeader {
		t.Errorf("expected header to be %s, got %s", defaultHeader, cfg.Header)
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

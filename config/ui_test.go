package config

import "testing"

func TestUIConfig_validateAndSetDefaults(t *testing.T) {
	StaticFolder = "../web/static"
	defer func() {
		StaticFolder = "./web/static"
	}()
	uiConfig := &UIConfig{Title: ""}
	if err := uiConfig.validateAndSetDefaults(); err != nil {
		t.Error("expected no error, got", err.Error())
	}
}

func TestGetDefaultUIConfig(t *testing.T) {
	defaultUIConfig := GetDefaultUIConfig()
	if defaultUIConfig.Title != defaultTitle {
		t.Error("expected GetDefaultUIConfig() to return defaultTitle, got", defaultUIConfig.Title)
	}
	if defaultUIConfig.Logo != defaultLogo {
		t.Error("expected GetDefaultUIConfig() to return defaultLogo, got", defaultUIConfig.Logo)
	}
}

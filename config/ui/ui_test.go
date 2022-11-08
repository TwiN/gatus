package ui

import (
	"strconv"
	"testing"
)

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	cfg := &Config{
		Title:       "",
		Description: "",
		Header:      "",
		Logo:        "",
		Link:        "",
	}
	if err := cfg.ValidateAndSetDefaults(); err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if cfg.Title != defaultTitle {
		t.Errorf("expected title to be %s, got %s", defaultTitle, cfg.Title)
	}
	if cfg.Description != defaultDescription {
		t.Errorf("expected description to be %s, got %s", defaultDescription, cfg.Description)
	}
	if cfg.Header != defaultHeader {
		t.Errorf("expected header to be %s, got %s", defaultHeader, cfg.Header)
	}
}

func TestButton_Validate(t *testing.T) {
	scenarios := []struct {
		Name, Link    string
		ExpectedError error
	}{
		{
			Name:          "",
			Link:          "",
			ExpectedError: ErrButtonValidationFailed,
		},
		{
			Name:          "",
			Link:          "link",
			ExpectedError: ErrButtonValidationFailed,
		},
		{
			Name:          "name",
			Link:          "",
			ExpectedError: ErrButtonValidationFailed,
		},
		{
			Name:          "name",
			Link:          "link",
			ExpectedError: nil,
		},
	}
	for i, scenario := range scenarios {
		t.Run(strconv.Itoa(i)+"_"+scenario.Name+"_"+scenario.Link, func(t *testing.T) {
			button := &Button{
				Name: scenario.Name,
				Link: scenario.Link,
			}
			if err := button.Validate(); err != scenario.ExpectedError {
				t.Errorf("expected error %v, got %v", scenario.ExpectedError, err)
			}
		})
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

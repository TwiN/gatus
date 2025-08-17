package ui

import (
	"errors"
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
	if cfg.DefaultSortBy != defaultSortBy {
		t.Errorf("expected defaultSortBy to be %s, got %s", defaultSortBy, cfg.DefaultSortBy)
	}
	if cfg.DefaultFilterBy != defaultFilterBy {
		t.Errorf("expected defaultFilterBy to be %s, got %s", defaultFilterBy, cfg.DefaultFilterBy)
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
	if defaultConfig.DefaultSortBy != defaultSortBy {
		t.Error("expected GetDefaultConfig() to return defaultSortBy, got", defaultConfig.DefaultSortBy)
	}
	if defaultConfig.DefaultFilterBy != defaultFilterBy {
		t.Error("expected GetDefaultConfig() to return defaultFilterBy, got", defaultConfig.DefaultFilterBy)
	}
}

func TestConfig_ValidateAndSetDefaults_DefaultSortBy(t *testing.T) {
	scenarios := []struct {
		Name          string
		DefaultSortBy string
		ExpectedError error
		ExpectedValue string
	}{
		{
			Name:          "EmptyDefaultSortBy",
			DefaultSortBy: "",
			ExpectedError: nil,
			ExpectedValue: defaultSortBy,
		},
		{
			Name:          "ValidDefaultSortBy_name",
			DefaultSortBy: "name",
			ExpectedError: nil,
			ExpectedValue: "name",
		},
		{
			Name:          "ValidDefaultSortBy_group",
			DefaultSortBy: "group",
			ExpectedError: nil,
			ExpectedValue: "group",
		},
		{
			Name:          "ValidDefaultSortBy_health",
			DefaultSortBy: "health",
			ExpectedError: nil,
			ExpectedValue: "health",
		},
		{
			Name:          "InvalidDefaultSortBy",
			DefaultSortBy: "invalid",
			ExpectedError: ErrInvalidDefaultSortBy,
			ExpectedValue: "invalid",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg := &Config{DefaultSortBy: scenario.DefaultSortBy}
			err := cfg.ValidateAndSetDefaults()
			if !errors.Is(err, scenario.ExpectedError) {
				t.Errorf("expected error %v, got %v", scenario.ExpectedError, err)
			}
			if cfg.DefaultSortBy != scenario.ExpectedValue {
				t.Errorf("expected DefaultSortBy to be %s, got %s", scenario.ExpectedValue, cfg.DefaultSortBy)
			}
		})
	}
}

func TestConfig_ValidateAndSetDefaults_DefaultFilterBy(t *testing.T) {
	scenarios := []struct {
		Name            string
		DefaultFilterBy string
		ExpectedError   error
		ExpectedValue   string
	}{
		{
			Name:            "EmptyDefaultFilterBy",
			DefaultFilterBy: "",
			ExpectedError:   nil,
			ExpectedValue:   defaultFilterBy,
		},
		{
			Name:            "ValidDefaultFilterBy_none",
			DefaultFilterBy: "none",
			ExpectedError:   nil,
			ExpectedValue:   "none",
		},
		{
			Name:            "ValidDefaultFilterBy_failing",
			DefaultFilterBy: "failing",
			ExpectedError:   nil,
			ExpectedValue:   "failing",
		},
		{
			Name:            "ValidDefaultFilterBy_unstable",
			DefaultFilterBy: "unstable",
			ExpectedError:   nil,
			ExpectedValue:   "unstable",
		},
		{
			Name:            "InvalidDefaultFilterBy",
			DefaultFilterBy: "invalid",
			ExpectedError:   ErrInvalidDefaultFilterBy,
			ExpectedValue:   "invalid",
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			cfg := &Config{DefaultFilterBy: scenario.DefaultFilterBy}
			err := cfg.ValidateAndSetDefaults()
			if !errors.Is(err, scenario.ExpectedError) {
				t.Errorf("expected error %v, got %v", scenario.ExpectedError, err)
			}
			if cfg.DefaultFilterBy != scenario.ExpectedValue {
				t.Errorf("expected DefaultFilterBy to be %s, got %s", scenario.ExpectedValue, cfg.DefaultFilterBy)
			}
		})
	}
}

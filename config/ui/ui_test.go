package ui

import (
	"errors"
	"strconv"
	"testing"
)

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	t.Run("empty-config", func(t *testing.T) {
		cfg := &Config{
			Title:               "",
			Description:         "",
			DashboardHeading:    "",
			DashboardSubheading: "",
			Header:              "",
			Logo:                "",
			Link:                "",
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
		if cfg.DashboardHeading != defaultDashboardHeading {
			t.Errorf("expected DashboardHeading to be %s, got %s", defaultDashboardHeading, cfg.DashboardHeading)
		}
		if cfg.DashboardSubheading != defaultDashboardSubheading {
			t.Errorf("expected DashboardSubheading to be %s, got %s", defaultDashboardSubheading, cfg.DashboardSubheading)
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
	})
	t.Run("custom-values", func(t *testing.T) {
		cfg := &Config{
			Title:               "Custom Title",
			Description:         "Custom Description",
			DashboardHeading:    "Production Status",
			DashboardSubheading: "Monitor all production endpoints",
			Header:              "My Company",
			Logo:                "https://example.com/logo.png",
			Link:                "https://example.com",
			DefaultSortBy:       "health",
			DefaultFilterBy:     "failing",
		}
		if err := cfg.ValidateAndSetDefaults(); err != nil {
			t.Error("expected no error, got", err.Error())
		}
		if cfg.Title != "Custom Title" {
			t.Errorf("expected title to be preserved, got %s", cfg.Title)
		}
		if cfg.Description != "Custom Description" {
			t.Errorf("expected description to be preserved, got %s", cfg.Description)
		}
		if cfg.DashboardHeading != "Production Status" {
			t.Errorf("expected DashboardHeading to be preserved, got %s", cfg.DashboardHeading)
		}
		if cfg.DashboardSubheading != "Monitor all production endpoints" {
			t.Errorf("expected DashboardSubheading to be preserved, got %s", cfg.DashboardSubheading)
		}
		if cfg.Header != "My Company" {
			t.Errorf("expected header to be preserved, got %s", cfg.Header)
		}
		if cfg.Logo != "https://example.com/logo.png" {
			t.Errorf("expected logo to be preserved, got %s", cfg.Logo)
		}
		if cfg.Link != "https://example.com" {
			t.Errorf("expected link to be preserved, got %s", cfg.Link)
		}
		if cfg.DefaultSortBy != "health" {
			t.Errorf("expected defaultSortBy to be preserved, got %s", cfg.DefaultSortBy)
		}
		if cfg.DefaultFilterBy != "failing" {
			t.Errorf("expected defaultFilterBy to be preserved, got %s", cfg.DefaultFilterBy)
		}
	})
	t.Run("partial-custom-values", func(t *testing.T) {
		cfg := &Config{
			Title:               "Custom Title",
			DashboardHeading:    "My Dashboard",
			Header:              "",
			DashboardSubheading: "",
		}
		if err := cfg.ValidateAndSetDefaults(); err != nil {
			t.Error("expected no error, got", err.Error())
		}
		if cfg.Title != "Custom Title" {
			t.Errorf("expected custom title to be preserved, got %s", cfg.Title)
		}
		if cfg.DashboardHeading != "My Dashboard" {
			t.Errorf("expected custom DashboardHeading to be preserved, got %s", cfg.DashboardHeading)
		}
		if cfg.DashboardSubheading != defaultDashboardSubheading {
			t.Errorf("expected DashboardSubheading to use default, got %s", cfg.DashboardSubheading)
		}
		if cfg.Header != defaultHeader {
			t.Errorf("expected header to use default, got %s", cfg.Header)
		}
		if cfg.Description != defaultDescription {
			t.Errorf("expected description to use default, got %s", cfg.Description)
		}
	})
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
	if defaultConfig.DashboardHeading != defaultDashboardHeading {
		t.Error("expected GetDefaultConfig() to return defaultDashboardHeading, got", defaultConfig.DashboardHeading)
	}
	if defaultConfig.DashboardSubheading != defaultDashboardSubheading {
		t.Error("expected GetDefaultConfig() to return defaultDashboardSubheading, got", defaultConfig.DashboardSubheading)
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

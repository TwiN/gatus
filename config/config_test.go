package config

import (
	"fmt"
	"github.com/TwinProduction/gatus/core"
	"testing"
	"time"
)

func TestParseAndValidateConfigBytes(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    interval: 15s
    conditions:
      - "[STATUS] == 200"
  - name: github
    url: https://api.github.com/healthz
    conditions:
      - "[STATUS] != 400"
      - "[STATUS] != 500"
`))
	if err != nil {
		t.Error("No error should've been returned")
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if len(config.Services) != 2 {
		t.Error("Should have returned two services")
	}
	if config.Services[0].Url != "https://twinnation.org/actuator/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/actuator/health")
	}
	if config.Services[1].Url != "https://api.github.com/healthz" {
		t.Errorf("URL should have been %s", "https://api.github.com/healthz")
	}
	fmt.Println(config.Services[0].Interval)
	if config.Services[0].Interval != 15*time.Second {
		t.Errorf("Interval should have been %s", 15*time.Second)
	}
	if config.Services[1].Interval != 10*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 10*time.Second)
	}
	if len(config.Services[0].Conditions) != 1 {
		t.Errorf("There should have been %d conditions", 1)
	}
	if len(config.Services[1].Conditions) != 2 {
		t.Errorf("There should have been %d conditions", 2)
	}
}

func TestParseAndValidateConfigBytesDefault(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("No error should've been returned")
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Metrics {
		t.Error("Metrics should've been false by default")
	}
	if config.Services[0].Url != "https://twinnation.org/actuator/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/actuator/health")
	}
	if config.Services[0].Interval != 10*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 10*time.Second)
	}
}

func TestParseAndValidateConfigBytesWithMetrics(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
metrics: true
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("No error should've been returned")
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if !config.Metrics {
		t.Error("Metrics should have been true")
	}
	if config.Services[0].Url != "https://twinnation.org/actuator/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/actuator/health")
	}
	if config.Services[0].Interval != 10*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 10*time.Second)
	}
}

func TestParseAndValidateBadConfigBytes(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
badconfig:
  - asdsa: w0w
    usadasdrl: asdxzczxc	
    asdas:
      - soup
`))
	if err == nil {
		t.Error("An error should've been returned")
	}
	if err != ErrNoServiceInConfig {
		t.Error("The error returned should have been of type ErrNoServiceInConfig")
	}
}

func TestParseAndValidateConfigBytesWithAlerting(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  slack: "http://example.com"
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    alerts:
      - type: slack
        enabled: true
        threshold: 7
        description: "Healthcheck failed 7 times in a row"
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("No error should've been returned")
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Metrics {
		t.Error("Metrics should've been false by default")
	}
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.Slack != "http://example.com" {
		t.Errorf("Slack webhook should've been %s, but was %s", "http://example.com", config.Alerting.Slack)
	}
	if len(config.Services) != 1 {
		t.Error("There should've been 1 service")
	}
	if config.Services[0].Url != "https://twinnation.org/actuator/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/actuator/health")
	}
	if config.Services[0].Interval != 10*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 10*time.Second)
	}
	if config.Services[0].Alerts == nil {
		t.Fatal("The service alerts shouldn't have been nil")
	}
	if len(config.Services[0].Alerts) != 1 {
		t.Fatal("There should've been 1 alert configured")
	}
	if !config.Services[0].Alerts[0].Enabled {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[0].Threshold != 7 {
		t.Errorf("The threshold of the alert should've been %d, but it was %d", 7, config.Services[0].Alerts[0].Threshold)
	}
	if config.Services[0].Alerts[0].Type != core.SlackAlert {
		t.Errorf("The type of the alert should've been %s, but it was %s", core.SlackAlert, config.Services[0].Alerts[0].Type)
	}
	if config.Services[0].Alerts[0].Description != "Healthcheck failed 7 times in a row" {
		t.Errorf("The type of the alert should've been %s, but it was %s", "Healthcheck failed 7 times in a row", config.Services[0].Alerts[0].Description)
	}
}

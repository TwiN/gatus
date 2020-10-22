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
	if config.Services[0].Method != "GET" {
		t.Errorf("Method should have been %s (default)", "GET")
	}
	if config.Services[1].Method != "GET" {
		t.Errorf("Method should have been %s (default)", "GET")
	}
	if config.Services[0].Interval != 15*time.Second {
		t.Errorf("Interval should have been %s", 15*time.Second)
	}
	if config.Services[1].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
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
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
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
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
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
  slack:
    webhook-url: "http://example.com"
  pagerduty:
    integration-key: "00000000000000000000000000000000"
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    alerts:
      - type: slack
        enabled: true
      - type: pagerduty
        enabled: true
        failure-threshold: 7
        success-threshold: 5
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
	if config.Alerting.Slack == nil || !config.Alerting.Slack.IsValid() {
		t.Fatal("Slack alerting config should've been valid")
	}
	if config.Alerting.Slack.WebhookUrl != "http://example.com" {
		t.Errorf("Slack webhook should've been %s, but was %s", "http://example.com", config.Alerting.Slack.WebhookUrl)
	}
	if config.Alerting.PagerDuty == nil || !config.Alerting.PagerDuty.IsValid() {
		t.Fatal("PagerDuty alerting config should've been valid")
	}
	if config.Alerting.PagerDuty.IntegrationKey != "00000000000000000000000000000000" {
		t.Errorf("PagerDuty integration key should've been %s, but was %s", "00000000000000000000000000000000", config.Alerting.PagerDuty.IntegrationKey)
	}
	if len(config.Services) != 1 {
		t.Error("There should've been 1 service")
	}
	if config.Services[0].Url != "https://twinnation.org/actuator/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/actuator/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Services[0].Alerts == nil {
		t.Fatal("The service alerts shouldn't have been nil")
	}
	if len(config.Services[0].Alerts) != 2 {
		t.Fatal("There should've been 2 alert configured")
	}
	if !config.Services[0].Alerts[0].Enabled {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[0].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Services[0].Alerts[0].FailureThreshold)
	}
	if config.Services[0].Alerts[0].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[0].SuccessThreshold)
	}
	if config.Services[0].Alerts[1].FailureThreshold != 7 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 7, config.Services[0].Alerts[1].FailureThreshold)
	}
	if config.Services[0].Alerts[1].SuccessThreshold != 5 {
		t.Errorf("The success threshold of the alert should've been %d, but it was %d", 5, config.Services[0].Alerts[1].SuccessThreshold)
	}
	if config.Services[0].Alerts[0].Type != core.SlackAlert {
		t.Errorf("The type of the alert should've been %s, but it was %s", core.SlackAlert, config.Services[0].Alerts[0].Type)
	}
	if config.Services[0].Alerts[1].Type != core.PagerDutyAlert {
		t.Errorf("The type of the alert should've been %s, but it was %s", core.PagerDutyAlert, config.Services[0].Alerts[1].Type)
	}
	if config.Services[0].Alerts[1].Description != "Healthcheck failed 7 times in a row" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "Healthcheck failed 7 times in a row", config.Services[0].Alerts[0].Description)
	}
}

func TestParseAndValidateConfigBytesWithInvalidPagerDutyAlertingConfig(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  pagerduty:
    integration-key: "INVALID_KEY"
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    alerts:
      - type: pagerduty
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("No error should've been returned")
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.PagerDuty == nil {
		t.Fatal("PagerDuty alerting config shouldn't have been nil")
	}
	if config.Alerting.PagerDuty.IsValid() {
		t.Fatal("PagerDuty alerting config should've been invalid")
	}
}

func TestParseAndValidateConfigBytesWithInvalidSecurityConfig(t *testing.T) {
	defer func() { recover() }()
	_, _ = parseAndValidateConfigBytes([]byte(`
security:
  basic:
    username: "admin"
    password-sha512: "invalid-sha512-hash"
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    conditions:
      - "[STATUS] == 200"
`))
	t.Error("Function should've panicked")
}

func TestParseAndValidateConfigBytesWithValidSecurityConfig(t *testing.T) {
	const expectedUsername = "admin"
	const expectedPasswordHash = "6b97ed68d14eb3f1aa959ce5d49c7dc612e1eb1dafd73b1e705847483fd6a6c809f2ceb4e8df6ff9984c6298ff0285cace6614bf8daa9f0070101b6c89899e22"
	config, err := parseAndValidateConfigBytes([]byte(fmt.Sprintf(`
security:
  basic:
    username: "%s"
    password-sha512: "%s"
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    conditions:
      - "[STATUS] == 200"
`, expectedUsername, expectedPasswordHash)))
	if err != nil {
		t.Error("No error should've been returned")
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Security == nil {
		t.Fatal("config.Security shouldn't have been nil")
	}
	if !config.Security.IsValid() {
		t.Error("Security config should've been valid")
	}
	if config.Security.Basic == nil {
		t.Fatal("config.Security.Basic shouldn't have been nil")
	}
	if config.Security.Basic.Username != expectedUsername {
		t.Errorf("config.Security.Basic.Username should've been %s, but was %s", expectedUsername, config.Security.Basic.Username)
	}
	if config.Security.Basic.PasswordSha512Hash != expectedPasswordHash {
		t.Errorf("config.Security.Basic.PasswordSha512Hash should've been %s, but was %s", expectedPasswordHash, config.Security.Basic.PasswordSha512Hash)
	}
}

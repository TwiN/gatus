package config

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TwinProduction/gatus/alerting"
	"github.com/TwinProduction/gatus/alerting/alert"
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/alerting/provider/discord"
	"github.com/TwinProduction/gatus/alerting/provider/mattermost"
	"github.com/TwinProduction/gatus/alerting/provider/messagebird"
	"github.com/TwinProduction/gatus/alerting/provider/pagerduty"
	"github.com/TwinProduction/gatus/alerting/provider/slack"
	"github.com/TwinProduction/gatus/alerting/provider/telegram"
	"github.com/TwinProduction/gatus/alerting/provider/twilio"
	"github.com/TwinProduction/gatus/alerting/provider/teams"
	"github.com/TwinProduction/gatus/client"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/k8stest"
	v1 "k8s.io/api/core/v1"
)

func TestLoadFileThatDoesNotExist(t *testing.T) {
	_, err := Load("file-that-does-not-exist.yaml")
	if err == nil {
		t.Error("Should've returned an error, because the file specified doesn't exist")
	}
}

func TestLoadDefaultConfigurationFile(t *testing.T) {
	_, err := LoadDefaultConfiguration()
	if err == nil {
		t.Error("Should've returned an error, because there's no configuration files at the default path nor the default fallback path")
	}
}

func TestParseAndValidateConfigBytes(t *testing.T) {
	file := t.TempDir() + "/test.db"
	config, err := parseAndValidateConfigBytes([]byte(fmt.Sprintf(`
storage:
  file: %s

services:
  - name: twinnation
    url: https://twinnation.org/health
    interval: 15s
    conditions:
      - "[STATUS] == 200"

  - name: github
    url: https://api.github.com/healthz
    client:
      insecure: true
      ignore-redirect: true
      timeout: 5s
    conditions:
      - "[STATUS] != 400"
      - "[STATUS] != 500"

  - name: example
    url: https://example.com/
    interval: 30m
    client:
      insecure: true
    conditions:
      - "[STATUS] == 200"
`, file)))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if len(config.Services) != 3 {
		t.Error("Should have returned two services")
	}

	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[0].Method != "GET" {
		t.Errorf("Method should have been %s (default)", "GET")
	}
	if config.Services[0].Interval != 15*time.Second {
		t.Errorf("Interval should have been %s", 15*time.Second)
	}
	if config.Services[0].ClientConfig.Insecure != client.GetDefaultConfig().Insecure {
		t.Errorf("ClientConfig.Insecure should have been %v, got %v", true, config.Services[0].ClientConfig.Insecure)
	}
	if config.Services[0].ClientConfig.IgnoreRedirect != client.GetDefaultConfig().IgnoreRedirect {
		t.Errorf("ClientConfig.IgnoreRedirect should have been %v, got %v", true, config.Services[0].ClientConfig.IgnoreRedirect)
	}
	if config.Services[0].ClientConfig.Timeout != client.GetDefaultConfig().Timeout {
		t.Errorf("ClientConfig.Timeout should have been %v, got %v", client.GetDefaultConfig().Timeout, config.Services[0].ClientConfig.Timeout)
	}
	if len(config.Services[0].Conditions) != 1 {
		t.Errorf("There should have been %d conditions", 1)
	}

	if config.Services[1].URL != "https://api.github.com/healthz" {
		t.Errorf("URL should have been %s", "https://api.github.com/healthz")
	}
	if config.Services[1].Method != "GET" {
		t.Errorf("Method should have been %s (default)", "GET")
	}
	if config.Services[1].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if !config.Services[1].ClientConfig.Insecure {
		t.Errorf("ClientConfig.Insecure should have been %v, got %v", true, config.Services[1].ClientConfig.Insecure)
	}
	if !config.Services[1].ClientConfig.IgnoreRedirect {
		t.Errorf("ClientConfig.IgnoreRedirect should have been %v, got %v", true, config.Services[1].ClientConfig.IgnoreRedirect)
	}
	if config.Services[1].ClientConfig.Timeout != 5*time.Second {
		t.Errorf("ClientConfig.Timeout should have been %v, got %v", 5*time.Second, config.Services[1].ClientConfig.Timeout)
	}
	if len(config.Services[1].Conditions) != 2 {
		t.Errorf("There should have been %d conditions", 2)
	}

	if config.Services[2].URL != "https://example.com/" {
		t.Errorf("URL should have been %s", "https://example.com/")
	}
	if config.Services[2].Method != "GET" {
		t.Errorf("Method should have been %s (default)", "GET")
	}
	if config.Services[2].Interval != 30*time.Minute {
		t.Errorf("Interval should have been %s, because it is the default value", 30*time.Minute)
	}
	if !config.Services[2].ClientConfig.Insecure {
		t.Errorf("ClientConfig.Insecure should have been %v, got %v", true, config.Services[2].ClientConfig.Insecure)
	}
	if config.Services[2].ClientConfig.IgnoreRedirect {
		t.Errorf("ClientConfig.IgnoreRedirect should have been %v by default, got %v", false, config.Services[2].ClientConfig.IgnoreRedirect)
	}
	if config.Services[2].ClientConfig.Timeout != 10*time.Second {
		t.Errorf("ClientConfig.Timeout should have been %v by default, got %v", 10*time.Second, config.Services[2].ClientConfig.Timeout)
	}
	if len(config.Services[2].Conditions) != 1 {
		t.Errorf("There should have been %d conditions", 1)
	}
}

func TestParseAndValidateConfigBytesDefault(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
services:
  - name: twinnation
    url: https://twinnation.org/health
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Metrics {
		t.Error("Metrics should've been false by default")
	}
	if config.Web.Address != DefaultAddress {
		t.Errorf("Bind address should have been %s, because it is the default value", DefaultAddress)
	}
	if config.Web.Port != DefaultPort {
		t.Errorf("Port should have been %d, because it is the default value", DefaultPort)
	}
	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Services[0].ClientConfig.Insecure != client.GetDefaultConfig().Insecure {
		t.Errorf("ClientConfig.Insecure should have been %v by default, got %v", true, config.Services[0].ClientConfig.Insecure)
	}
	if config.Services[0].ClientConfig.IgnoreRedirect != client.GetDefaultConfig().IgnoreRedirect {
		t.Errorf("ClientConfig.IgnoreRedirect should have been %v by default, got %v", true, config.Services[0].ClientConfig.IgnoreRedirect)
	}
	if config.Services[0].ClientConfig.Timeout != client.GetDefaultConfig().Timeout {
		t.Errorf("ClientConfig.Timeout should have been %v by default, got %v", client.GetDefaultConfig().Timeout, config.Services[0].ClientConfig.Timeout)
	}
}

func TestParseAndValidateConfigBytesWithAddress(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
web:
  address: 127.0.0.1
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Metrics {
		t.Error("Metrics should've been false by default")
	}
	if config.Services[0].URL != "https://twinnation.org/actuator/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/actuator/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Web.Address != "127.0.0.1" {
		t.Errorf("Bind address should have been %s, because it is specified in config", "127.0.0.1")
	}
	if config.Web.Port != DefaultPort {
		t.Errorf("Port should have been %d, because it is the default value", DefaultPort)
	}
}

func TestParseAndValidateConfigBytesWithPort(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
web:
  port: 12345
services:
  - name: twinnation
    url: https://twinnation.org/health
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Metrics {
		t.Error("Metrics should've been false by default")
	}
	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Web.Address != DefaultAddress {
		t.Errorf("Bind address should have been %s, because it is the default value", DefaultAddress)
	}
	if config.Web.Port != 12345 {
		t.Errorf("Port should have been %d, because it is specified in config", 12345)
	}
}

func TestParseAndValidateConfigBytesWithPortAndHost(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
web:
  port: 12345
  address: 127.0.0.1
services:
  - name: twinnation
    url: https://twinnation.org/health
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Metrics {
		t.Error("Metrics should've been false by default")
	}
	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Web.Address != "127.0.0.1" {
		t.Errorf("Bind address should have been %s, because it is specified in config", "127.0.0.1")
	}
	if config.Web.Port != 12345 {
		t.Errorf("Port should have been %d, because it is specified in config", 12345)
	}
}

func TestParseAndValidateConfigBytesWithInvalidPort(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
web:
  port: 65536
  address: 127.0.0.1
services:
  - name: twinnation
    url: https://twinnation.org/health
    conditions:
      - "[STATUS] == 200"
`))
	if err == nil {
		t.Fatal("Should've returned an error because the configuration specifies an invalid port value")
	}
}

func TestParseAndValidateConfigBytesWithMetricsAndCustomUserAgentHeader(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
metrics: true
services:
  - name: twinnation
    url: https://twinnation.org/health
    headers:
      User-Agent: Test/2.0
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if !config.Metrics {
		t.Error("Metrics should have been true")
	}
	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Web.Address != DefaultAddress {
		t.Errorf("Bind address should have been %s, because it is the default value", DefaultAddress)
	}
	if config.Web.Port != DefaultPort {
		t.Errorf("Port should have been %d, because it is the default value", DefaultPort)
	}
	if userAgent := config.Services[0].Headers["User-Agent"]; userAgent != "Test/2.0" {
		t.Errorf("User-Agent should've been %s, got %s", "Test/2.0", userAgent)
	}
}

func TestParseAndValidateConfigBytesWithMetricsAndHostAndPort(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
metrics: true
web:
  address: 192.168.0.1
  port: 9090
services:
  - name: twinnation
    url: https://twinnation.org/health
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if !config.Metrics {
		t.Error("Metrics should have been true")
	}
	if config.Web.Address != "192.168.0.1" {
		t.Errorf("Bind address should have been %s, because it is the default value", "192.168.0.1")
	}
	if config.Web.Port != 9090 {
		t.Errorf("Port should have been %d, because it is specified in config", 9090)
	}
	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if userAgent := config.Services[0].Headers["User-Agent"]; userAgent != core.GatusUserAgent {
		t.Errorf("User-Agent should've been %s because it's the default value, got %s", core.GatusUserAgent, userAgent)
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
debug: true
alerting:
  slack:
    webhook-url: "http://example.com"
  discord:
    webhook-url: "http://example.org"
  pagerduty:
    integration-key: "00000000000000000000000000000000"
  mattermost:
    webhook-url: "http://example.com"
    client:
      insecure: true
  messagebird:
    access-key: "1"
    originator: "31619191918"
    recipients: "31619191919"
  telegram:
    token: 123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
    id: 0123456789
  twilio:
    sid: "1234"
    token: "5678"
    from: "+1-234-567-8901"
    to: "+1-234-567-8901"
  teams:
    webhook-url: "http://example.com"

services:
  - name: twinnation
    url: https://twinnation.org/health
    alerts:
      - type: slack
        enabled: true
      - type: pagerduty
        enabled: true
        failure-threshold: 7
        success-threshold: 5
        description: "Healthcheck failed 7 times in a row"
      - type: mattermost
        enabled: true
      - type: messagebird
      - type: discord
        enabled: true
        failure-threshold: 10
      - type: telegram
        enabled: true
      - type: twilio
        enabled: true
        failure-threshold: 12
        success-threshold: 15
      - type: teams
        enabled: true
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	// Alerting providers
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.Slack == nil || !config.Alerting.Slack.IsValid() {
		t.Fatal("Slack alerting config should've been valid")
	}
	// Services
	if len(config.Services) != 1 {
		t.Error("There should've been 1 service")
	}
	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if len(config.Services[0].Alerts) != 8 {
		t.Fatal("There should've been 8 alerts configured")
	}

	if config.Services[0].Alerts[0].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Services[0].Alerts[0].Type)
	}
	if !config.Services[0].Alerts[0].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[0].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Services[0].Alerts[0].FailureThreshold)
	}
	if config.Services[0].Alerts[0].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[0].SuccessThreshold)
	}

	if config.Services[0].Alerts[1].Type != alert.TypePagerDuty {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypePagerDuty, config.Services[0].Alerts[1].Type)
	}
	if config.Services[0].Alerts[1].GetDescription() != "Healthcheck failed 7 times in a row" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "Healthcheck failed 7 times in a row", config.Services[0].Alerts[1].GetDescription())
	}
	if config.Services[0].Alerts[1].FailureThreshold != 7 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 7, config.Services[0].Alerts[1].FailureThreshold)
	}
	if config.Services[0].Alerts[1].SuccessThreshold != 5 {
		t.Errorf("The success threshold of the alert should've been %d, but it was %d", 5, config.Services[0].Alerts[1].SuccessThreshold)
	}

	if config.Services[0].Alerts[2].Type != alert.TypeMattermost {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeMattermost, config.Services[0].Alerts[2].Type)
	}
	if !config.Services[0].Alerts[2].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[2].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Services[0].Alerts[2].FailureThreshold)
	}
	if config.Services[0].Alerts[2].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[2].SuccessThreshold)
	}

	if config.Services[0].Alerts[3].Type != alert.TypeMessagebird {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeMessagebird, config.Services[0].Alerts[3].Type)
	}
	if config.Services[0].Alerts[3].IsEnabled() {
		t.Error("The alert should've been disabled")
	}

	if config.Services[0].Alerts[4].Type != alert.TypeDiscord {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeDiscord, config.Services[0].Alerts[4].Type)
	}
	if !config.Services[0].Alerts[4].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[4].FailureThreshold != 10 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 10, config.Services[0].Alerts[4].FailureThreshold)
	}
	if config.Services[0].Alerts[4].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[4].SuccessThreshold)
	}

	if config.Services[0].Alerts[5].Type != alert.TypeTelegram {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTelegram, config.Services[0].Alerts[5].Type)
	}
	if !config.Services[0].Alerts[5].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[5].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Services[0].Alerts[5].FailureThreshold)
	}
	if config.Services[0].Alerts[5].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[5].SuccessThreshold)
	}

	if config.Services[0].Alerts[6].Type != alert.TypeTwilio {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTwilio, config.Services[0].Alerts[6].Type)
	}
	if !config.Services[0].Alerts[6].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[6].FailureThreshold != 12 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 12, config.Services[0].Alerts[6].FailureThreshold)
	}
	if config.Services[0].Alerts[6].SuccessThreshold != 15 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 15, config.Services[0].Alerts[6].SuccessThreshold)
	}

	if config.Services[0].Alerts[7].Type != alert.TypeTeams {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTeams, config.Services[0].Alerts[7].Type)
	}
	if !config.Services[0].Alerts[7].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[7].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Services[0].Alerts[7].FailureThreshold)
	}
	if config.Services[0].Alerts[7].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[7].SuccessThreshold)
	}
}

func TestParseAndValidateConfigBytesWithAlertingAndDefaultAlert(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
debug: true

alerting:
  slack:
    webhook-url: "http://example.com"
    default-alert:
      enabled: true
  discord:
    webhook-url: "http://example.org"
    default-alert:
      enabled: true
      failure-threshold: 10
      success-threshold: 1
  pagerduty:
    integration-key: "00000000000000000000000000000000"
    default-alert:
      enabled: true
      description: default description
      failure-threshold: 7
      success-threshold: 5
  mattermost:
    webhook-url: "http://example.com"
    default-alert:
      enabled: true
  messagebird:
    access-key: "1"
    originator: "31619191918"
    recipients: "31619191919"
    default-alert:
      enabled: false
      send-on-resolved: true
  telegram:
    token: 123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11
    id: 0123456789
    default-alert:
      enabled: true
  twilio:
    sid: "1234"
    token: "5678"
    from: "+1-234-567-8901"
    to: "+1-234-567-8901"
    default-alert:
      enabled: true
      failure-threshold: 12
      success-threshold: 15
  teams:
    webhook-url: "http://example.com"
    default-alert:
      enabled: true

services:
 - name: twinnation
   url: https://twinnation.org/health
   alerts:
     - type: slack
     - type: pagerduty
     - type: mattermost
     - type: messagebird
     - type: discord
       success-threshold: 2 # test service alert override
     - type: telegram
     - type: twilio
     - type: teams
   conditions:
     - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Metrics {
		t.Error("Metrics should've been false by default")
	}
	// Alerting providers
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.Slack == nil || !config.Alerting.Slack.IsValid() {
		t.Fatal("Slack alerting config should've been valid")
	}
	if config.Alerting.Slack.GetDefaultAlert() == nil {
		t.Fatal("Slack.GetDefaultAlert() shouldn't have returned nil")
	}
	if config.Alerting.Slack.WebhookURL != "http://example.com" {
		t.Errorf("Slack webhook should've been %s, but was %s", "http://example.com", config.Alerting.Slack.WebhookURL)
	}

	if config.Alerting.PagerDuty == nil || !config.Alerting.PagerDuty.IsValid() {
		t.Fatal("PagerDuty alerting config should've been valid")
	}
	if config.Alerting.PagerDuty.GetDefaultAlert() == nil {
		t.Fatal("PagerDuty.GetDefaultAlert() shouldn't have returned nil")
	}
	if config.Alerting.PagerDuty.IntegrationKey != "00000000000000000000000000000000" {
		t.Errorf("PagerDuty integration key should've been %s, but was %s", "00000000000000000000000000000000", config.Alerting.PagerDuty.IntegrationKey)
	}

	if config.Alerting.Mattermost == nil || !config.Alerting.Mattermost.IsValid() {
		t.Fatal("Mattermost alerting config should've been valid")
	}
	if config.Alerting.Mattermost.GetDefaultAlert() == nil {
		t.Fatal("Mattermost.GetDefaultAlert() shouldn't have returned nil")
	}

	if config.Alerting.Messagebird == nil || !config.Alerting.Messagebird.IsValid() {
		t.Fatal("Messagebird alerting config should've been valid")
	}
	if config.Alerting.Messagebird.GetDefaultAlert() == nil {
		t.Fatal("Messagebird.GetDefaultAlert() shouldn't have returned nil")
	}
	if config.Alerting.Messagebird.AccessKey != "1" {
		t.Errorf("Messagebird access key should've been %s, but was %s", "1", config.Alerting.Messagebird.AccessKey)
	}
	if config.Alerting.Messagebird.Originator != "31619191918" {
		t.Errorf("Messagebird originator field should've been %s, but was %s", "31619191918", config.Alerting.Messagebird.Originator)
	}
	if config.Alerting.Messagebird.Recipients != "31619191919" {
		t.Errorf("Messagebird to recipients should've been %s, but was %s", "31619191919", config.Alerting.Messagebird.Recipients)
	}

	if config.Alerting.Discord == nil || !config.Alerting.Discord.IsValid() {
		t.Fatal("Discord alerting config should've been valid")
	}
	if config.Alerting.Discord.GetDefaultAlert() == nil {
		t.Fatal("Discord.GetDefaultAlert() shouldn't have returned nil")
	}
	if config.Alerting.Discord.WebhookURL != "http://example.org" {
		t.Errorf("Discord webhook should've been %s, but was %s", "http://example.org", config.Alerting.Discord.WebhookURL)
	}
	if config.Alerting.GetAlertingProviderByAlertType(alert.TypeDiscord) != config.Alerting.Discord {
		t.Error("expected discord configuration")
	}

	if config.Alerting.Telegram == nil || !config.Alerting.Telegram.IsValid() {
		t.Fatal("Telegram alerting config should've been valid")
	}
	if config.Alerting.Telegram.GetDefaultAlert() == nil {
		t.Fatal("Telegram.GetDefaultAlert() shouldn't have returned nil")
	}
	if config.Alerting.Telegram.Token != "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11" {
		t.Errorf("Telegram token should've been %s, but was %s", "123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11", config.Alerting.Telegram.Token)
	}
	if config.Alerting.Telegram.ID != "0123456789" {
		t.Errorf("Telegram ID should've been %s, but was %s", "012345689", config.Alerting.Telegram.ID)
	}

	if config.Alerting.Twilio == nil || !config.Alerting.Twilio.IsValid() {
		t.Fatal("Twilio alerting config should've been valid")
	}
	if config.Alerting.Twilio.GetDefaultAlert() == nil {
		t.Fatal("Twilio.GetDefaultAlert() shouldn't have returned nil")
	}

	if config.Alerting.Teams == nil || !config.Alerting.Teams.IsValid() {
		t.Fatal("Teams alerting config should've been valid")
	}
	if config.Alerting.Teams.GetDefaultAlert() == nil {
		t.Fatal("Teams.GetDefaultAlert() shouldn't have returned nil")
	}

	// Services
	if len(config.Services) != 1 {
		t.Error("There should've been 1 service")
	}
	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if len(config.Services[0].Alerts) != 8 {
		t.Fatal("There should've been 8 alerts configured")
	}

	if config.Services[0].Alerts[0].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Services[0].Alerts[0].Type)
	}
	if !config.Services[0].Alerts[0].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[0].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Services[0].Alerts[0].FailureThreshold)
	}
	if config.Services[0].Alerts[0].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[0].SuccessThreshold)
	}

	if config.Services[0].Alerts[1].Type != alert.TypePagerDuty {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypePagerDuty, config.Services[0].Alerts[1].Type)
	}
	if config.Services[0].Alerts[1].GetDescription() != "default description" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "default description", config.Services[0].Alerts[1].GetDescription())
	}
	if config.Services[0].Alerts[1].FailureThreshold != 7 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 7, config.Services[0].Alerts[1].FailureThreshold)
	}
	if config.Services[0].Alerts[1].SuccessThreshold != 5 {
		t.Errorf("The success threshold of the alert should've been %d, but it was %d", 5, config.Services[0].Alerts[1].SuccessThreshold)
	}

	if config.Services[0].Alerts[2].Type != alert.TypeMattermost {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeMattermost, config.Services[0].Alerts[2].Type)
	}
	if !config.Services[0].Alerts[2].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[2].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Services[0].Alerts[2].FailureThreshold)
	}
	if config.Services[0].Alerts[2].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[2].SuccessThreshold)
	}

	if config.Services[0].Alerts[3].Type != alert.TypeMessagebird {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeMessagebird, config.Services[0].Alerts[3].Type)
	}
	if config.Services[0].Alerts[3].IsEnabled() {
		t.Error("The alert should've been disabled")
	}
	if !config.Services[0].Alerts[3].IsSendingOnResolved() {
		t.Error("The alert should be sending on resolve")
	}

	if config.Services[0].Alerts[4].Type != alert.TypeDiscord {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeDiscord, config.Services[0].Alerts[4].Type)
	}
	if !config.Services[0].Alerts[4].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[4].FailureThreshold != 10 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 10, config.Services[0].Alerts[4].FailureThreshold)
	}
	if config.Services[0].Alerts[4].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[4].SuccessThreshold)
	}

	if config.Services[0].Alerts[5].Type != alert.TypeTelegram {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTelegram, config.Services[0].Alerts[5].Type)
	}
	if !config.Services[0].Alerts[5].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[5].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Services[0].Alerts[5].FailureThreshold)
	}
	if config.Services[0].Alerts[5].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[5].SuccessThreshold)
	}

	if config.Services[0].Alerts[6].Type != alert.TypeTwilio {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTwilio, config.Services[0].Alerts[6].Type)
	}
	if !config.Services[0].Alerts[6].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[6].FailureThreshold != 12 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 12, config.Services[0].Alerts[6].FailureThreshold)
	}
	if config.Services[0].Alerts[6].SuccessThreshold != 15 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 15, config.Services[0].Alerts[6].SuccessThreshold)
	}

	if config.Services[0].Alerts[7].Type != alert.TypeTeams {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTeams, config.Services[0].Alerts[7].Type)
	}
	if !config.Services[0].Alerts[7].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[7].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Services[0].Alerts[7].FailureThreshold)
	}
	if config.Services[0].Alerts[7].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Services[0].Alerts[7].SuccessThreshold)
	}

}

func TestParseAndValidateConfigBytesWithAlertingAndDefaultAlertAndMultipleAlertsOfSameTypeWithOverriddenParameters(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  slack:
    webhook-url: "http://example.com"
    default-alert:
      enabled: true
      description: "description"

services:
 - name: twinnation
   url: https://twinnation.org/health
   alerts:
     - type: slack
       failure-threshold: 10
     - type: slack
       failure-threshold: 20
       description: "wow"
     - type: slack
       enabled: false
       failure-threshold: 30
   conditions:
     - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	// Alerting providers
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.Slack == nil || !config.Alerting.Slack.IsValid() {
		t.Fatal("Slack alerting config should've been valid")
	}
	// Services
	if len(config.Services) != 1 {
		t.Error("There should've been 2 services")
	}
	if config.Services[0].Alerts[0].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Services[0].Alerts[0].Type)
	}
	if config.Services[0].Alerts[1].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Services[0].Alerts[1].Type)
	}
	if config.Services[0].Alerts[2].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Services[0].Alerts[2].Type)
	}
	if !config.Services[0].Alerts[0].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if !config.Services[0].Alerts[1].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Services[0].Alerts[2].IsEnabled() {
		t.Error("The alert should've been disabled")
	}
	if config.Services[0].Alerts[0].GetDescription() != "description" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "description", config.Services[0].Alerts[0].GetDescription())
	}
	if config.Services[0].Alerts[1].GetDescription() != "wow" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "description", config.Services[0].Alerts[1].GetDescription())
	}
	if config.Services[0].Alerts[2].GetDescription() != "description" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "description", config.Services[0].Alerts[2].GetDescription())
	}
	if config.Services[0].Alerts[0].FailureThreshold != 10 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 10, config.Services[0].Alerts[0].FailureThreshold)
	}
	if config.Services[0].Alerts[1].FailureThreshold != 20 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 20, config.Services[0].Alerts[1].FailureThreshold)
	}
	if config.Services[0].Alerts[2].FailureThreshold != 30 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 30, config.Services[0].Alerts[2].FailureThreshold)
	}
}

func TestParseAndValidateConfigBytesWithInvalidPagerDutyAlertingConfig(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  pagerduty:
    integration-key: "INVALID_KEY"
services:
  - name: twinnation
    url: https://twinnation.org/health
    alerts:
      - type: pagerduty
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
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

func TestParseAndValidateConfigBytesWithCustomAlertingConfig(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  custom:
    url: "https://example.com"
    body: |
      {
        "text": "[ALERT_TRIGGERED_OR_RESOLVED]: [SERVICE_NAME] - [ALERT_DESCRIPTION]"
      }
services:
  - name: twinnation
    url: https://twinnation.org/health
    alerts:
      - type: custom
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.Custom == nil {
		t.Fatal("PagerDuty alerting config shouldn't have been nil")
	}
	if !config.Alerting.Custom.IsValid() {
		t.Fatal("Custom alerting config should've been valid")
	}
	if config.Alerting.Custom.GetAlertStatePlaceholderValue(true) != "RESOLVED" {
		t.Fatal("ALERT_TRIGGERED_OR_RESOLVED placeholder value for RESOLVED should've been 'RESOLVED', got", config.Alerting.Custom.GetAlertStatePlaceholderValue(true))
	}
	if config.Alerting.Custom.GetAlertStatePlaceholderValue(false) != "TRIGGERED" {
		t.Fatal("ALERT_TRIGGERED_OR_RESOLVED placeholder value for TRIGGERED should've been 'TRIGGERED', got", config.Alerting.Custom.GetAlertStatePlaceholderValue(false))
	}
	if config.Alerting.Custom.Insecure {
		t.Fatal("config.Alerting.Custom.Insecure shouldn't have been true")
	}
	if config.Alerting.Custom.ClientConfig.Insecure {
		t.Errorf("ClientConfig.Insecure should have been %v, got %v", false, config.Alerting.Custom.ClientConfig.Insecure)
	}
}

func TestParseAndValidateConfigBytesWithCustomAlertingConfigAndCustomPlaceholderValues(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  custom:
    placeholders:
      ALERT_TRIGGERED_OR_RESOLVED:
        TRIGGERED: "partial_outage"
        RESOLVED: "operational"
    url: "https://example.com"
    insecure: true
    body: "[ALERT_TRIGGERED_OR_RESOLVED]: [SERVICE_NAME] - [ALERT_DESCRIPTION]"
services:
  - name: twinnation
    url: https://twinnation.org/health
    alerts:
      - type: custom
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.Custom == nil {
		t.Fatal("PagerDuty alerting config shouldn't have been nil")
	}
	if !config.Alerting.Custom.IsValid() {
		t.Fatal("Custom alerting config should've been valid")
	}
	if config.Alerting.Custom.GetAlertStatePlaceholderValue(true) != "operational" {
		t.Fatal("ALERT_TRIGGERED_OR_RESOLVED placeholder value for RESOLVED should've been 'operational'")
	}
	if config.Alerting.Custom.GetAlertStatePlaceholderValue(false) != "partial_outage" {
		t.Fatal("ALERT_TRIGGERED_OR_RESOLVED placeholder value for TRIGGERED should've been 'partial_outage'")
	}
	if !config.Alerting.Custom.Insecure {
		t.Fatal("config.Alerting.Custom.Insecure shouldn't have been true")
	}
}

func TestParseAndValidateConfigBytesWithCustomAlertingConfigAndOneCustomPlaceholderValue(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  custom:
    placeholders:
      ALERT_TRIGGERED_OR_RESOLVED:
        TRIGGERED: "partial_outage"
    url: "https://example.com"
    insecure: true
    body: "[ALERT_TRIGGERED_OR_RESOLVED]: [SERVICE_NAME] - [ALERT_DESCRIPTION]"
services:
  - name: twinnation
    url: https://twinnation.org/health
    alerts:
      - type: custom
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.Custom == nil {
		t.Fatal("PagerDuty alerting config shouldn't have been nil")
	}
	if !config.Alerting.Custom.IsValid() {
		t.Fatal("Custom alerting config should've been valid")
	}
	if config.Alerting.Custom.GetAlertStatePlaceholderValue(true) != "RESOLVED" {
		t.Fatal("ALERT_TRIGGERED_OR_RESOLVED placeholder value for RESOLVED should've been 'RESOLVED'")
	}
	if config.Alerting.Custom.GetAlertStatePlaceholderValue(false) != "partial_outage" {
		t.Fatal("ALERT_TRIGGERED_OR_RESOLVED placeholder value for TRIGGERED should've been 'partial_outage'")
	}
	if !config.Alerting.Custom.Insecure {
		t.Fatal("config.Alerting.Custom.Insecure shouldn't have been true")
	}
}

func TestParseAndValidateConfigBytesWithCustomAlertingConfigThatHasInsecureSetToTrue(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  custom:
    url: "https://example.com"
    method: "POST"
    insecure: true
    body: |
      {
        "text": "[ALERT_TRIGGERED_OR_RESOLVED]: [SERVICE_NAME] - [ALERT_DESCRIPTION]"
      }
services:
  - name: twinnation
    url: https://twinnation.org/health
    alerts:
      - type: custom
    conditions:
      - "[STATUS] == 200"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.Custom == nil {
		t.Fatal("PagerDuty alerting config shouldn't have been nil")
	}
	if !config.Alerting.Custom.IsValid() {
		t.Error("Custom alerting config should've been valid")
	}
	if config.Alerting.Custom.Method != "POST" {
		t.Error("config.Alerting.Custom.Method should've been POST")
	}
	if !config.Alerting.Custom.Insecure {
		t.Error("config.Alerting.Custom.Insecure shouldn't have been true")
	}
}

func TestParseAndValidateConfigBytesWithInvalidServiceName(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
services:
  - name: ""
    url: https://twinnation.org/health
    conditions:
      - "[STATUS] == 200"
`))
	if err != core.ErrServiceWithNoName {
		t.Error("should've returned an error")
	}
}

func TestParseAndValidateConfigBytesWithInvalidStorageConfig(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
storage:
  type: sqlite
services:
  - name: example
    url: https://example.org
    conditions:
      - "[STATUS] == 200"
`))
	if err == nil {
		t.Error("should've returned an error, because a file must be specified for a storage of type sqlite")
	}
}

func TestParseAndValidateConfigBytesWithInvalidYAML(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
storage:
  invalid yaml
services:
  - name: example
    url: https://example.org
    conditions:
      - "[STATUS] == 200"
`))
	if err == nil {
		t.Error("should've returned an error")
	}
}

func TestParseAndValidateConfigBytesWithInvalidSecurityConfig(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
security:
  basic:
    username: "admin"
    password-sha512: "invalid-sha512-hash"
services:
  - name: twinnation
    url: https://twinnation.org/health
    conditions:
      - "[STATUS] == 200"
`))
	if err == nil {
		t.Error("should've returned an error")
	}
}

func TestParseAndValidateConfigBytesWithValidSecurityConfig(t *testing.T) {
	const expectedUsername = "admin"
	const expectedPasswordHash = "6b97ed68d14eb3f1aa959ce5d49c7dc612e1eb1dafd73b1e705847483fd6a6c809f2ceb4e8df6ff9984c6298ff0285cace6614bf8daa9f0070101b6c89899e22"
	config, err := parseAndValidateConfigBytes([]byte(fmt.Sprintf(`debug: true
security:
  basic:
    username: "%s"
    password-sha512: "%s"
services:
  - name: twinnation
    url: https://twinnation.org/health
    conditions:
      - "[STATUS] == 200"
`, expectedUsername, expectedPasswordHash)))
	if err != nil {
		t.Error("expected no error, got", err.Error())
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

func TestParseAndValidateConfigBytesWithNoServicesOrAutoDiscovery(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(``))
	if err != ErrNoServiceInConfig {
		t.Error("The error returned should have been of type ErrNoServiceInConfig")
	}
}

func TestParseAndValidateConfigBytesWithKubernetesAutoDiscovery(t *testing.T) {
	var kubernetesServices []v1.Service
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-1", "default", 8080))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-2", "default", 8080))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-2-canary", "default", 8080))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-3", "kube-system", 8080))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-4", "tools", 8080))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-5", "tools", 8080))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-6", "tools", 8080))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-7", "metrics", 8080))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-7-canary", "metrics", 8080))
	k8stest.InitializeMockedKubernetesClient(kubernetesServices)
	config, err := parseAndValidateConfigBytes([]byte(`
debug: true

kubernetes:
  cluster-mode: "mock"
  auto-discover: true
  excluded-service-suffixes:
    - canary
  service-template:
    interval: 29s
    conditions:
      - "[STATUS] == 200"
  namespaces:
    - name: default
      hostname-suffix: ".default.svc.cluster.local"
      target-path: "/health"
    - name: tools
      hostname-suffix: ".tools.svc.cluster.local"
      target-path: "/health"
      excluded-services:
        - service-6
    - name: metrics
      hostname-suffix: ".metrics.svc.cluster.local"
      target-path: "/health"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Kubernetes == nil {
		t.Fatal("Kuberbetes config shouldn't have been nil")
	}
	if len(config.Services) != 5 {
		t.Error("Expected 5 services to have been added through k8s auto discovery, got", len(config.Services))
	}
	for _, service := range config.Services {
		if service.Name == "service-2-canary" || service.Name == "service-7-canary" {
			t.Errorf("service '%s' should've been excluded because excluded-service-suffixes has 'canary'", service.Name)
		} else if service.Name == "service-6" {
			t.Errorf("service '%s' should've been excluded because excluded-services has 'service-6'", service.Name)
		} else if service.Name == "service-3" {
			t.Errorf("service '%s' should've been excluded because the namespace 'kube-system' is not configured for auto discovery", service.Name)
		} else {
			if service.Interval != 29*time.Second {
				t.Errorf("service '%s' should've had an interval of 29s, because the template is configured for it", service.Name)
			}
			if len(service.Conditions) != 1 {
				t.Errorf("service '%s' should've had 1 condition", service.Name)
			}
			if len(service.Conditions) == 1 && *service.Conditions[0] != "[STATUS] == 200" {
				t.Errorf("service '%s' should've had the condition '[STATUS] == 200', because the template is configured for it", service.Name)
			}
			if !strings.HasSuffix(service.URL, ".svc.cluster.local:8080/health") {
				t.Errorf("service '%s' should've had an URL with the suffix '.svc.cluster.local:8080/health'", service.Name)
			}
		}
	}
}

func TestParseAndValidateConfigBytesWithKubernetesAutoDiscoveryButNoServiceTemplate(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
kubernetes:
  cluster-mode: "mock"
  auto-discover: true
  namespaces:
    - name: default
      hostname-suffix: ".default.svc.cluster.local"
      target-path: "/health"
`))
	if err == nil {
		t.Error("should've returned an error because providing a service-template is mandatory")
	}
}

func TestParseAndValidateConfigBytesWithKubernetesAutoDiscoveryUsingClusterModeIn(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
kubernetes:
  cluster-mode: "in"
  auto-discover: true
  service-template:
    interval: 30s
    conditions:
      - "[STATUS] == 200"
  namespaces:
    - name: default
      hostname-suffix: ".default.svc.cluster.local"
      target-path: "/health"
`))
	if err == nil {
		t.Error("should've returned an error because testing with ClusterModeIn isn't supported")
	}
}

func TestGetAlertingProviderByAlertType(t *testing.T) {
	alertingConfig := &alerting.Config{
		Custom:      &custom.AlertProvider{},
		Discord:     &discord.AlertProvider{},
		Mattermost:  &mattermost.AlertProvider{},
		Messagebird: &messagebird.AlertProvider{},
		PagerDuty:   &pagerduty.AlertProvider{},
		Slack:       &slack.AlertProvider{},
		Telegram:    &telegram.AlertProvider{},
		Twilio:      &twilio.AlertProvider{},
		Teams:      &teams.AlertProvider{},
	}
	if alertingConfig.GetAlertingProviderByAlertType(alert.TypeCustom) != alertingConfig.Custom {
		t.Error("expected Custom configuration")
	}
	if alertingConfig.GetAlertingProviderByAlertType(alert.TypeDiscord) != alertingConfig.Discord {
		t.Error("expected Discord configuration")
	}
	if alertingConfig.GetAlertingProviderByAlertType(alert.TypeMattermost) != alertingConfig.Mattermost {
		t.Error("expected Mattermost configuration")
	}
	if alertingConfig.GetAlertingProviderByAlertType(alert.TypeMessagebird) != alertingConfig.Messagebird {
		t.Error("expected Messagebird configuration")
	}
	if alertingConfig.GetAlertingProviderByAlertType(alert.TypePagerDuty) != alertingConfig.PagerDuty {
		t.Error("expected PagerDuty configuration")
	}
	if alertingConfig.GetAlertingProviderByAlertType(alert.TypeSlack) != alertingConfig.Slack {
		t.Error("expected Slack configuration")
	}
	if alertingConfig.GetAlertingProviderByAlertType(alert.TypeTelegram) != alertingConfig.Telegram {
		t.Error("expected Telegram configuration")
	}
	if alertingConfig.GetAlertingProviderByAlertType(alert.TypeTwilio) != alertingConfig.Twilio {
		t.Error("expected Twilio configuration")
	}
	if alertingConfig.GetAlertingProviderByAlertType(alert.TypeTeams) != alertingConfig.Teams {
		t.Error("expected Teams configuration")
	}
}

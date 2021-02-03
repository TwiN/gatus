package config

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/k8stest"
	v1 "k8s.io/api/core/v1"
)

func TestGetBeforeConfigIsLoaded(t *testing.T) {
	defer func() { recover() }()
	Get()
	t.Fatal("Should've panicked because the configuration hasn't been loaded yet")
}

func TestSet(t *testing.T) {
	if config != nil {
		t.Fatal("config should've been nil")
	}
	Set(&Config{})
	if config == nil {
		t.Fatal("config shouldn't have been nil")
	}
}

func TestLoadFileThatDoesNotExist(t *testing.T) {
	err := Load("file-that-does-not-exist.yaml")
	if err == nil {
		t.Error("Should've returned an error, because the file specified doesn't exist")
	}
}

func TestLoadDefaultConfigurationFile(t *testing.T) {
	err := LoadDefaultConfiguration()
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
    conditions:
      - "[STATUS] != 400"
      - "[STATUS] != 500"
`, file)))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if len(config.Services) != 2 {
		t.Error("Should have returned two services")
	}
	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[1].URL != "https://api.github.com/healthz" {
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
	if config.Web.Port != DefaultPort {
		t.Errorf("Port should have been %d, because it is the default value", DefaultPort)
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
	defer func() { recover() }()
	_, _ = parseAndValidateConfigBytes([]byte(`
web:
  port: 65536
  address: 127.0.0.1
services:
  - name: twinnation
    url: https://twinnation.org/health
    conditions:
      - "[STATUS] == 200"
`))
	t.Fatal("Should've panicked because the configuration specifies an invalid port value")
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
  pagerduty:
    integration-key: "00000000000000000000000000000000"
  messagebird:
    access-key: "1"
    originator: "31619191918"
    recipients: "31619191919"
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
      - type: messagebird
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
	if config.Metrics {
		t.Error("Metrics should've been false by default")
	}
	if config.Alerting == nil {
		t.Fatal("config.Alerting shouldn't have been nil")
	}
	if config.Alerting.Slack == nil || !config.Alerting.Slack.IsValid() {
		t.Fatal("Slack alerting config should've been valid")
	}
	if config.Alerting.Slack.WebhookURL != "http://example.com" {
		t.Errorf("Slack webhook should've been %s, but was %s", "http://example.com", config.Alerting.Slack.WebhookURL)
	}
	if config.Alerting.PagerDuty == nil || !config.Alerting.PagerDuty.IsValid() {
		t.Fatal("PagerDuty alerting config should've been valid")
	}
	if config.Alerting.Messagebird == nil || !config.Alerting.Messagebird.IsValid() {
		t.Fatal("Messagebird alerting config should've been valid")
	}
	if config.Alerting.PagerDuty.IntegrationKey != "00000000000000000000000000000000" {
		t.Errorf("PagerDuty integration key should've been %s, but was %s", "00000000000000000000000000000000", config.Alerting.PagerDuty.IntegrationKey)
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
	if len(config.Services) != 1 {
		t.Error("There should've been 1 service")
	}
	if config.Services[0].URL != "https://twinnation.org/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/health")
	}
	if config.Services[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Services[0].Alerts == nil {
		t.Fatal("The service alerts shouldn't have been nil")
	}
	if len(config.Services[0].Alerts) != 3 {
		t.Fatal("There should've been 3 alert configured")
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
	if config.Services[0].Alerts[2].Type != core.MessagebirdAlert {
		t.Errorf("The type of the alert should've been %s, but it was %s", core.MessagebirdAlert, config.Services[0].Alerts[1].Type)
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
	if config.Alerting.Custom.Insecure {
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

func TestParseAndValidateConfigBytesWithInvalidSecurityConfig(t *testing.T) {
	defer func() { recover() }()
	_, _ = parseAndValidateConfigBytes([]byte(`
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
	defer func() { recover() }()
	_, _ = parseAndValidateConfigBytes([]byte(`
kubernetes:
  cluster-mode: "mock"
  auto-discover: true
  namespaces:
    - name: default
      hostname-suffix: ".default.svc.cluster.local"
      target-path: "/health"
`))
	t.Error("Function should've panicked because providing a service-template is mandatory")
}

func TestParseAndValidateConfigBytesWithKubernetesAutoDiscoveryUsingClusterModeIn(t *testing.T) {
	defer func() { recover() }()
	_, _ = parseAndValidateConfigBytes([]byte(`
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
	// TODO: find a way to test this?
	t.Error("Function should've panicked because testing with ClusterModeIn isn't supported")
}

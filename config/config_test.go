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
	if config.Services[0].URL != "https://twinnation.org/actuator/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/actuator/health")
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
	if config.Services[0].URL != "https://twinnation.org/actuator/health" {
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
	if config.Services[0].URL != "https://twinnation.org/actuator/health" {
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
	if config.Alerting.Slack.WebhookURL != "http://example.com" {
		t.Errorf("Slack webhook should've been %s, but was %s", "http://example.com", config.Alerting.Slack.WebhookURL)
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
	if config.Services[0].URL != "https://twinnation.org/actuator/health" {
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

func TestParseAndValidateConfigBytesWithNoServicesOrAutoDiscovery(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(``))
	if err != ErrNoServiceInConfig {
		t.Error("The error returned should have been of type ErrNoServiceInConfig")
	}
}

func TestParseAndValidateConfigBytesWithKubernetesAutoDiscovery(t *testing.T) {
	var kubernetesServices []v1.Service
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-1", "default"))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-2", "default"))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-2-canary", "default"))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-3", "kube-system"))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-4", "tools"))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-5", "tools"))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-6", "tools"))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-7", "metrics"))
	kubernetesServices = append(kubernetesServices, k8stest.CreateTestServices("service-7-canary", "metrics"))
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
		t.Error("No error should've been returned")
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
			if !strings.HasSuffix(service.URL, ".svc.cluster.local/health") {
				t.Errorf("service '%s' should've had an URL with the suffix '.svc.cluster.local/health'", service.Name)
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

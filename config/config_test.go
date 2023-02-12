package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/alerting/provider/discord"
	"github.com/TwiN/gatus/v5/alerting/provider/email"
	"github.com/TwiN/gatus/v5/alerting/provider/github"
	"github.com/TwiN/gatus/v5/alerting/provider/googlechat"
	"github.com/TwiN/gatus/v5/alerting/provider/matrix"
	"github.com/TwiN/gatus/v5/alerting/provider/mattermost"
	"github.com/TwiN/gatus/v5/alerting/provider/messagebird"
	"github.com/TwiN/gatus/v5/alerting/provider/ntfy"
	"github.com/TwiN/gatus/v5/alerting/provider/opsgenie"
	"github.com/TwiN/gatus/v5/alerting/provider/pagerduty"
	"github.com/TwiN/gatus/v5/alerting/provider/pushover"
	"github.com/TwiN/gatus/v5/alerting/provider/slack"
	"github.com/TwiN/gatus/v5/alerting/provider/teams"
	"github.com/TwiN/gatus/v5/alerting/provider/telegram"
	"github.com/TwiN/gatus/v5/alerting/provider/twilio"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/web"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage"
	"gopkg.in/yaml.v3"
)

func TestLoadConfiguration(t *testing.T) {
	dir := t.TempDir()
	scenarios := []struct {
		name           string
		configPath     string            // value to pass as the configPath parameter in LoadConfiguration
		pathAndFiles   map[string]string // files to create in dir
		expectedConfig *Config
		expectedError  error
	}{
		{
			name:       "empty-config-file",
			configPath: filepath.Join(dir, "config.yaml"),
			pathAndFiles: map[string]string{
				"config.yaml": "",
			},
			expectedError: ErrConfigFileNotFound,
		},
		{
			name:          "config-file-that-does-not-exist",
			configPath:    filepath.Join(dir, "config.yaml"),
			expectedError: ErrConfigFileNotFound,
		},
		{
			name:       "config-file-with-endpoint-that-has-no-url",
			configPath: filepath.Join(dir, "config.yaml"),
			pathAndFiles: map[string]string{
				"config.yaml": `
endpoints:
  - name: website`,
			},
			expectedError: core.ErrEndpointWithNoURL,
		},
		{
			name:       "config-file-with-endpoint-that-has-no-conditions",
			configPath: filepath.Join(dir, "config.yaml"),
			pathAndFiles: map[string]string{
				"config.yaml": `
endpoints:
  - name: website
    url: https://twin.sh/health`,
			},
			expectedError: core.ErrEndpointWithNoCondition,
		},
		{
			name:       "config-file",
			configPath: filepath.Join(dir, "config.yaml"),
			pathAndFiles: map[string]string{
				"config.yaml": `
endpoints:
  - name: website
    url: https://twin.sh/health
    conditions:
      - "[STATUS] == 200"`,
			},
			expectedConfig: &Config{
				Endpoints: []*core.Endpoint{
					{
						Name:       "website",
						URL:        "https://twin.sh/health",
						Conditions: []core.Condition{"[STATUS] == 200"},
					},
				},
			},
		},
		{
			name:          "empty-dir",
			configPath:    dir,
			pathAndFiles:  map[string]string{},
			expectedError: ErrConfigFileNotFound,
		},
		{
			name:       "dir-with-empty-config-file",
			configPath: dir,
			pathAndFiles: map[string]string{
				"config.yaml": "",
			},
			expectedError: ErrNoEndpointInConfig,
		},
		{
			name:       "dir-with-two-config-files",
			configPath: dir,
			pathAndFiles: map[string]string{
				"config.yaml": `endpoints: 
  - name: one
    url: https://example.com
    conditions:
      - "[CONNECTED] == true"
      - "[STATUS] == 200"

  - name: two
    url: https://example.org
    conditions:
      - "len([BODY]) > 0"`,
				"config.yml": `endpoints: 
  - name: three
    url: https://twin.sh/health
    conditions:
      - "[STATUS] == 200"
      - "[BODY].status == UP"`,
			},
			expectedConfig: &Config{
				Endpoints: []*core.Endpoint{
					{
						Name:       "one",
						URL:        "https://example.com",
						Conditions: []core.Condition{"[CONNECTED] == true", "[STATUS] == 200"},
					},
					{
						Name:       "two",
						URL:        "https://example.org",
						Conditions: []core.Condition{"len([BODY]) > 0"},
					},
					{
						Name:       "three",
						URL:        "https://twin.sh/health",
						Conditions: []core.Condition{"[STATUS] == 200", "[BODY].status == UP"},
					},
				},
			},
		},
		{
			name:       "dir-with-2-config-files-deep-merge-with-map-slice-and-primitive",
			configPath: dir,
			pathAndFiles: map[string]string{
				"a.yaml": `
metrics: true

alerting:
  slack:
    webhook-url: https://hooks.slack.com/services/xxx/yyy/zzz

endpoints:
  - name: example
    url: https://example.org
    interval: 5s
    conditions:
      - "[STATUS] == 200"`,
				"b.yaml": `
debug: true

alerting:
  discord:
    webhook-url: https://discord.com/api/webhooks/xxx/yyy

endpoints:
  - name: frontend
    url: https://example.com
    conditions:
      - "[STATUS] == 200"`,
			},
			expectedConfig: &Config{
				Debug:   true,
				Metrics: true,
				Alerting: &alerting.Config{
					Discord: &discord.AlertProvider{WebhookURL: "https://discord.com/api/webhooks/xxx/yyy"},
					Slack:   &slack.AlertProvider{WebhookURL: "https://hooks.slack.com/services/xxx/yyy/zzz"},
				},
				Endpoints: []*core.Endpoint{
					{
						Name:       "example",
						URL:        "https://example.org",
						Interval:   5 * time.Second,
						Conditions: []core.Condition{"[STATUS] == 200"},
					},
					{
						Name:       "frontend",
						URL:        "https://example.com",
						Conditions: []core.Condition{"[STATUS] == 200"},
					},
				},
			},
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for path, content := range scenario.pathAndFiles {
				if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0644); err != nil {
					t.Fatalf("[%s] failed to write file: %v", scenario.name, err)
				}
			}
			defer func(pathAndFiles map[string]string) {
				for path := range pathAndFiles {
					_ = os.Remove(filepath.Join(dir, path))
				}
			}(scenario.pathAndFiles)
			config, err := LoadConfiguration(scenario.configPath)
			if !errors.Is(err, scenario.expectedError) {
				t.Errorf("[%s] expected error %v, got %v", scenario.name, scenario.expectedError, err)
				return
			} else if err != nil && errors.Is(err, scenario.expectedError) {
				return
			}
			// parse the expected output so that expectations are closer to reality (under the right circumstances, even I can be poetic)
			expectedConfigAsYAML, _ := yaml.Marshal(scenario.expectedConfig)
			expectedConfigAfterBeingParsedAndValidated, err := parseAndValidateConfigBytes(expectedConfigAsYAML)
			if err != nil {
				t.Fatalf("[%s] failed to parse expected config: %v", scenario.name, err)
			}
			// Marshal em' before comparing em' so that we don't have to deal with formatting and ordering
			actualConfigAsYAML, err := yaml.Marshal(config)
			if err != nil {
				t.Fatalf("[%s] failed to marshal actual config: %v", scenario.name, err)
			}
			expectedConfigAfterBeingParsedAndValidatedAsYAML, _ := yaml.Marshal(expectedConfigAfterBeingParsedAndValidated)
			if string(actualConfigAsYAML) != string(expectedConfigAfterBeingParsedAndValidatedAsYAML) {
				t.Errorf("[%s] expected config %s, got %s", scenario.name, string(expectedConfigAfterBeingParsedAndValidatedAsYAML), string(actualConfigAsYAML))
			}
		})
	}
}

func TestConfig_HasLoadedConfigurationBeenModified(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	configFilePath := filepath.Join(dir, "config.yaml")
	_ = os.WriteFile(configFilePath, []byte(`endpoints:
  - name: website
    url: https://twin.sh/health
    conditions:
      - "[STATUS] == 200"
`), 0644)

	t.Run("config-file-as-config-path", func(t *testing.T) {
		config, err := LoadConfiguration(configFilePath)
		if err != nil {
			t.Fatalf("failed to load configuration: %v", err)
		}
		if config.HasLoadedConfigurationBeenModified() {
			t.Errorf("expected config.HasLoadedConfigurationBeenModified() to return false because nothing has happened since it was created")
		}
		time.Sleep(time.Second) // Because the file mod time only has second precision, we have to wait for a second
		// Update the config file
		if err = os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`endpoints:
  - name: website
    url: https://twin.sh/health
    conditions:
      - "[STATUS] == 200"`), 0644); err != nil {
			t.Fatalf("failed to overwrite config file: %v", err)
		}
		if !config.HasLoadedConfigurationBeenModified() {
			t.Errorf("expected config.HasLoadedConfigurationBeenModified() to return true because a new file has been added in the directory")
		}
	})
	t.Run("config-directory-as-config-path", func(t *testing.T) {
		config, err := LoadConfiguration(dir)
		if err != nil {
			t.Fatalf("failed to load configuration: %v", err)
		}
		if config.HasLoadedConfigurationBeenModified() {
			t.Errorf("expected config.HasLoadedConfigurationBeenModified() to return false because nothing has happened since it was created")
		}
		time.Sleep(time.Second) // Because the file mod time only has second precision, we have to wait for a second
		// Update the config file
		if err = os.WriteFile(filepath.Join(dir, "metrics.yaml"), []byte(`metrics: true`), 0644); err != nil {
			t.Fatalf("failed to overwrite config file: %v", err)
		}
		if !config.HasLoadedConfigurationBeenModified() {
			t.Errorf("expected config.HasLoadedConfigurationBeenModified() to return true because a new file has been added in the directory")
		}
	})
}

func TestParseAndValidateConfigBytes(t *testing.T) {
	file := t.TempDir() + "/test.db"
	config, err := parseAndValidateConfigBytes([]byte(fmt.Sprintf(`
storage:
  type: sqlite
  path: %s
maintenance:
  enabled: true
  start: 00:00
  duration: 4h
  every: [Monday, Thursday]
ui:
  title: T
  header: H
  link: https://example.org
  buttons:
    - name: "Home"
      link: "https://example.org"
    - name: "Status page"
      link: "https://status.example.org"
endpoints:
  - name: website
    url: https://twin.sh/health
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
	if config.Storage == nil || config.Storage.Path != file || config.Storage.Type != storage.TypeSQLite {
		t.Error("expected storage to be set to sqlite, got", config.Storage)
	}
	if config.UI == nil || config.UI.Title != "T" || config.UI.Header != "H" || config.UI.Link != "https://example.org" || len(config.UI.Buttons) != 2 || config.UI.Buttons[0].Name != "Home" || config.UI.Buttons[0].Link != "https://example.org" || config.UI.Buttons[1].Name != "Status page" || config.UI.Buttons[1].Link != "https://status.example.org" {
		t.Error("expected ui to be set to T, H, https://example.org, 2 buttons, Home and Status page, got", config.UI)
	}
	if mc := config.Maintenance; mc == nil || mc.Start != "00:00" || !mc.IsEnabled() || mc.Duration != 4*time.Hour || len(mc.Every) != 2 {
		t.Error("Expected Config.Maintenance to be configured properly")
	}
	if len(config.Endpoints) != 3 {
		t.Error("Should have returned two endpoints")
	}

	if config.Endpoints[0].URL != "https://twin.sh/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/health")
	}
	if config.Endpoints[0].Method != "GET" {
		t.Errorf("Method should have been %s (default)", "GET")
	}
	if config.Endpoints[0].Interval != 15*time.Second {
		t.Errorf("Interval should have been %s", 15*time.Second)
	}
	if config.Endpoints[0].ClientConfig.Insecure != client.GetDefaultConfig().Insecure {
		t.Errorf("ClientConfig.Insecure should have been %v, got %v", true, config.Endpoints[0].ClientConfig.Insecure)
	}
	if config.Endpoints[0].ClientConfig.IgnoreRedirect != client.GetDefaultConfig().IgnoreRedirect {
		t.Errorf("ClientConfig.IgnoreRedirect should have been %v, got %v", true, config.Endpoints[0].ClientConfig.IgnoreRedirect)
	}
	if config.Endpoints[0].ClientConfig.Timeout != client.GetDefaultConfig().Timeout {
		t.Errorf("ClientConfig.Timeout should have been %v, got %v", client.GetDefaultConfig().Timeout, config.Endpoints[0].ClientConfig.Timeout)
	}
	if len(config.Endpoints[0].Conditions) != 1 {
		t.Errorf("There should have been %d conditions", 1)
	}

	if config.Endpoints[1].URL != "https://api.github.com/healthz" {
		t.Errorf("URL should have been %s", "https://api.github.com/healthz")
	}
	if config.Endpoints[1].Method != "GET" {
		t.Errorf("Method should have been %s (default)", "GET")
	}
	if config.Endpoints[1].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if !config.Endpoints[1].ClientConfig.Insecure {
		t.Errorf("ClientConfig.Insecure should have been %v, got %v", true, config.Endpoints[1].ClientConfig.Insecure)
	}
	if !config.Endpoints[1].ClientConfig.IgnoreRedirect {
		t.Errorf("ClientConfig.IgnoreRedirect should have been %v, got %v", true, config.Endpoints[1].ClientConfig.IgnoreRedirect)
	}
	if config.Endpoints[1].ClientConfig.Timeout != 5*time.Second {
		t.Errorf("ClientConfig.Timeout should have been %v, got %v", 5*time.Second, config.Endpoints[1].ClientConfig.Timeout)
	}
	if len(config.Endpoints[1].Conditions) != 2 {
		t.Errorf("There should have been %d conditions", 2)
	}

	if config.Endpoints[2].URL != "https://example.com/" {
		t.Errorf("URL should have been %s", "https://example.com/")
	}
	if config.Endpoints[2].Method != "GET" {
		t.Errorf("Method should have been %s (default)", "GET")
	}
	if config.Endpoints[2].Interval != 30*time.Minute {
		t.Errorf("Interval should have been %s, because it is the default value", 30*time.Minute)
	}
	if !config.Endpoints[2].ClientConfig.Insecure {
		t.Errorf("ClientConfig.Insecure should have been %v, got %v", true, config.Endpoints[2].ClientConfig.Insecure)
	}
	if config.Endpoints[2].ClientConfig.IgnoreRedirect {
		t.Errorf("ClientConfig.IgnoreRedirect should have been %v by default, got %v", false, config.Endpoints[2].ClientConfig.IgnoreRedirect)
	}
	if config.Endpoints[2].ClientConfig.Timeout != 10*time.Second {
		t.Errorf("ClientConfig.Timeout should have been %v by default, got %v", 10*time.Second, config.Endpoints[2].ClientConfig.Timeout)
	}
	if len(config.Endpoints[2].Conditions) != 1 {
		t.Errorf("There should have been %d conditions", 1)
	}
}

func TestParseAndValidateConfigBytesDefault(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
endpoints:
  - name: website
    url: https://twin.sh/health
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
	if config.Web.Address != web.DefaultAddress {
		t.Errorf("Bind address should have been %s, because it is the default value", web.DefaultAddress)
	}
	if config.Web.Port != web.DefaultPort {
		t.Errorf("Port should have been %d, because it is the default value", web.DefaultPort)
	}
	if config.Endpoints[0].URL != "https://twin.sh/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/health")
	}
	if config.Endpoints[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Endpoints[0].ClientConfig.Insecure != client.GetDefaultConfig().Insecure {
		t.Errorf("ClientConfig.Insecure should have been %v by default, got %v", true, config.Endpoints[0].ClientConfig.Insecure)
	}
	if config.Endpoints[0].ClientConfig.IgnoreRedirect != client.GetDefaultConfig().IgnoreRedirect {
		t.Errorf("ClientConfig.IgnoreRedirect should have been %v by default, got %v", true, config.Endpoints[0].ClientConfig.IgnoreRedirect)
	}
	if config.Endpoints[0].ClientConfig.Timeout != client.GetDefaultConfig().Timeout {
		t.Errorf("ClientConfig.Timeout should have been %v by default, got %v", client.GetDefaultConfig().Timeout, config.Endpoints[0].ClientConfig.Timeout)
	}
}

func TestParseAndValidateConfigBytesWithAddress(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
web:
  address: 127.0.0.1
endpoints:
  - name: website
    url: https://twin.sh/actuator/health
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
	if config.Endpoints[0].URL != "https://twin.sh/actuator/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/actuator/health")
	}
	if config.Endpoints[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Web.Address != "127.0.0.1" {
		t.Errorf("Bind address should have been %s, because it is specified in config", "127.0.0.1")
	}
	if config.Web.Port != web.DefaultPort {
		t.Errorf("Port should have been %d, because it is the default value", web.DefaultPort)
	}
}

func TestParseAndValidateConfigBytesWithPort(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
web:
  port: 12345
endpoints:
  - name: website
    url: https://twin.sh/health
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
	if config.Endpoints[0].URL != "https://twin.sh/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/health")
	}
	if config.Endpoints[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Web.Address != web.DefaultAddress {
		t.Errorf("Bind address should have been %s, because it is the default value", web.DefaultAddress)
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
endpoints:
  - name: website
    url: https://twin.sh/health
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
	if config.Endpoints[0].URL != "https://twin.sh/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/health")
	}
	if config.Endpoints[0].Interval != 60*time.Second {
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
endpoints:
  - name: website
    url: https://twin.sh/health
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
endpoints:
  - name: website
    url: https://twin.sh/health
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
	if config.Endpoints[0].URL != "https://twin.sh/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/health")
	}
	if config.Endpoints[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if config.Web.Address != web.DefaultAddress {
		t.Errorf("Bind address should have been %s, because it is the default value", web.DefaultAddress)
	}
	if config.Web.Port != web.DefaultPort {
		t.Errorf("Port should have been %d, because it is the default value", web.DefaultPort)
	}
	if userAgent := config.Endpoints[0].Headers["User-Agent"]; userAgent != "Test/2.0" {
		t.Errorf("User-Agent should've been %s, got %s", "Test/2.0", userAgent)
	}
}

func TestParseAndValidateConfigBytesWithMetricsAndHostAndPort(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
metrics: true
web:
  address: 192.168.0.1
  port: 9090
endpoints:
  - name: website
    url: https://twin.sh/health
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
	if config.Endpoints[0].URL != "https://twin.sh/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/health")
	}
	if config.Endpoints[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if userAgent := config.Endpoints[0].Headers["User-Agent"]; userAgent != core.GatusUserAgent {
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
	if err != ErrNoEndpointInConfig {
		t.Error("The error returned should have been of type ErrNoEndpointInConfig")
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
  pushover:
    application-token: "000000000000000000000000000000"
    user-key: "000000000000000000000000000000"
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

endpoints:
  - name: website
    url: https://twin.sh/health
    alerts:
      - type: slack
      - type: pagerduty
        failure-threshold: 7
        success-threshold: 5
        description: "Healthcheck failed 7 times in a row"
      - type: mattermost
      - type: messagebird
        enabled: false
      - type: discord
        failure-threshold: 10
      - type: telegram
        enabled: true
      - type: twilio
        failure-threshold: 12
        success-threshold: 15
      - type: teams
      - type: pushover
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
	// Endpoints
	if len(config.Endpoints) != 1 {
		t.Error("There should've been 1 endpoint")
	}
	if config.Endpoints[0].URL != "https://twin.sh/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/health")
	}
	if config.Endpoints[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if len(config.Endpoints[0].Alerts) != 9 {
		t.Fatal("There should've been 9 alerts configured")
	}

	if config.Endpoints[0].Alerts[0].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Endpoints[0].Alerts[0].Type)
	}
	if !config.Endpoints[0].Alerts[0].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[0].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Endpoints[0].Alerts[0].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[0].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[0].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[1].Type != alert.TypePagerDuty {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypePagerDuty, config.Endpoints[0].Alerts[1].Type)
	}
	if config.Endpoints[0].Alerts[1].GetDescription() != "Healthcheck failed 7 times in a row" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "Healthcheck failed 7 times in a row", config.Endpoints[0].Alerts[1].GetDescription())
	}
	if config.Endpoints[0].Alerts[1].FailureThreshold != 7 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 7, config.Endpoints[0].Alerts[1].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[1].SuccessThreshold != 5 {
		t.Errorf("The success threshold of the alert should've been %d, but it was %d", 5, config.Endpoints[0].Alerts[1].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[2].Type != alert.TypeMattermost {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeMattermost, config.Endpoints[0].Alerts[2].Type)
	}
	if !config.Endpoints[0].Alerts[2].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[2].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Endpoints[0].Alerts[2].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[2].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[2].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[3].Type != alert.TypeMessagebird {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeMessagebird, config.Endpoints[0].Alerts[3].Type)
	}
	if config.Endpoints[0].Alerts[3].IsEnabled() {
		t.Error("The alert should've been disabled")
	}

	if config.Endpoints[0].Alerts[4].Type != alert.TypeDiscord {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeDiscord, config.Endpoints[0].Alerts[4].Type)
	}
	if !config.Endpoints[0].Alerts[4].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[4].FailureThreshold != 10 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 10, config.Endpoints[0].Alerts[4].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[4].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[4].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[5].Type != alert.TypeTelegram {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTelegram, config.Endpoints[0].Alerts[5].Type)
	}
	if !config.Endpoints[0].Alerts[5].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[5].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Endpoints[0].Alerts[5].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[5].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[5].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[6].Type != alert.TypeTwilio {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTwilio, config.Endpoints[0].Alerts[6].Type)
	}
	if !config.Endpoints[0].Alerts[6].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[6].FailureThreshold != 12 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 12, config.Endpoints[0].Alerts[6].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[6].SuccessThreshold != 15 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 15, config.Endpoints[0].Alerts[6].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[7].Type != alert.TypeTeams {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTeams, config.Endpoints[0].Alerts[7].Type)
	}
	if !config.Endpoints[0].Alerts[7].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[7].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Endpoints[0].Alerts[7].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[7].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[7].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[8].Type != alert.TypePushover {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypePushover, config.Endpoints[0].Alerts[8].Type)
	}
	if !config.Endpoints[0].Alerts[8].IsEnabled() {
		t.Error("The alert should've been enabled")
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
  pushover:
    application-token: "000000000000000000000000000000"
    user-key: "000000000000000000000000000000"
    default-alert:
      enabled: true
      description: default description
      failure-threshold: 5
      success-threshold: 3
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

endpoints:
 - name: website
   url: https://twin.sh/health
   alerts:
     - type: slack
     - type: pagerduty
     - type: mattermost
     - type: messagebird
     - type: discord
       success-threshold: 2 # test endpoint alert override
     - type: telegram
     - type: twilio
     - type: teams
     - type: pushover
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

	if config.Alerting.Pushover == nil || !config.Alerting.Pushover.IsValid() {
		t.Fatal("Pushover alerting config should've been valid")
	}
	if config.Alerting.Pushover.GetDefaultAlert() == nil {
		t.Fatal("Pushover.GetDefaultAlert() shouldn't have returned nil")
	}
	if config.Alerting.Pushover.ApplicationToken != "000000000000000000000000000000" {
		t.Errorf("Pushover application token should've been %s, but was %s", "000000000000000000000000000000", config.Alerting.Pushover.ApplicationToken)
	}
	if config.Alerting.Pushover.UserKey != "000000000000000000000000000000" {
		t.Errorf("Pushover user key should've been %s, but was %s", "000000000000000000000000000000", config.Alerting.Pushover.UserKey)
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

	// Endpoints
	if len(config.Endpoints) != 1 {
		t.Error("There should've been 1 endpoint")
	}
	if config.Endpoints[0].URL != "https://twin.sh/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/health")
	}
	if config.Endpoints[0].Interval != 60*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 60*time.Second)
	}
	if len(config.Endpoints[0].Alerts) != 9 {
		t.Fatal("There should've been 9 alerts configured")
	}

	if config.Endpoints[0].Alerts[0].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Endpoints[0].Alerts[0].Type)
	}
	if !config.Endpoints[0].Alerts[0].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[0].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Endpoints[0].Alerts[0].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[0].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[0].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[1].Type != alert.TypePagerDuty {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypePagerDuty, config.Endpoints[0].Alerts[1].Type)
	}
	if config.Endpoints[0].Alerts[1].GetDescription() != "default description" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "default description", config.Endpoints[0].Alerts[1].GetDescription())
	}
	if config.Endpoints[0].Alerts[1].FailureThreshold != 7 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 7, config.Endpoints[0].Alerts[1].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[1].SuccessThreshold != 5 {
		t.Errorf("The success threshold of the alert should've been %d, but it was %d", 5, config.Endpoints[0].Alerts[1].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[2].Type != alert.TypeMattermost {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeMattermost, config.Endpoints[0].Alerts[2].Type)
	}
	if !config.Endpoints[0].Alerts[2].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[2].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Endpoints[0].Alerts[2].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[2].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[2].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[3].Type != alert.TypeMessagebird {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeMessagebird, config.Endpoints[0].Alerts[3].Type)
	}
	if config.Endpoints[0].Alerts[3].IsEnabled() {
		t.Error("The alert should've been disabled")
	}
	if !config.Endpoints[0].Alerts[3].IsSendingOnResolved() {
		t.Error("The alert should be sending on resolve")
	}

	if config.Endpoints[0].Alerts[4].Type != alert.TypeDiscord {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeDiscord, config.Endpoints[0].Alerts[4].Type)
	}
	if !config.Endpoints[0].Alerts[4].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[4].FailureThreshold != 10 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 10, config.Endpoints[0].Alerts[4].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[4].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[4].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[5].Type != alert.TypeTelegram {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTelegram, config.Endpoints[0].Alerts[5].Type)
	}
	if !config.Endpoints[0].Alerts[5].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[5].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Endpoints[0].Alerts[5].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[5].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[5].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[6].Type != alert.TypeTwilio {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTwilio, config.Endpoints[0].Alerts[6].Type)
	}
	if !config.Endpoints[0].Alerts[6].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[6].FailureThreshold != 12 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 12, config.Endpoints[0].Alerts[6].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[6].SuccessThreshold != 15 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 15, config.Endpoints[0].Alerts[6].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[7].Type != alert.TypeTeams {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeTeams, config.Endpoints[0].Alerts[7].Type)
	}
	if !config.Endpoints[0].Alerts[7].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[7].FailureThreshold != 3 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Endpoints[0].Alerts[7].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[7].SuccessThreshold != 2 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[7].SuccessThreshold)
	}

	if config.Endpoints[0].Alerts[8].Type != alert.TypePushover {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypePushover, config.Endpoints[0].Alerts[8].Type)
	}
	if !config.Endpoints[0].Alerts[8].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[8].FailureThreshold != 5 {
		t.Errorf("The default failure threshold of the alert should've been %d, but it was %d", 3, config.Endpoints[0].Alerts[8].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[8].SuccessThreshold != 3 {
		t.Errorf("The default success threshold of the alert should've been %d, but it was %d", 2, config.Endpoints[0].Alerts[8].SuccessThreshold)
	}

}

func TestParseAndValidateConfigBytesWithAlertingAndDefaultAlertAndMultipleAlertsOfSameTypeWithOverriddenParameters(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  slack:
    webhook-url: "https://example.com"
    default-alert:
      enabled: true
      description: "description"

endpoints:
 - name: website
   url: https://twin.sh/health
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
	// Endpoints
	if len(config.Endpoints) != 1 {
		t.Error("There should've been 2 endpoints")
	}
	if config.Endpoints[0].Alerts[0].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Endpoints[0].Alerts[0].Type)
	}
	if config.Endpoints[0].Alerts[1].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Endpoints[0].Alerts[1].Type)
	}
	if config.Endpoints[0].Alerts[2].Type != alert.TypeSlack {
		t.Errorf("The type of the alert should've been %s, but it was %s", alert.TypeSlack, config.Endpoints[0].Alerts[2].Type)
	}
	if !config.Endpoints[0].Alerts[0].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if !config.Endpoints[0].Alerts[1].IsEnabled() {
		t.Error("The alert should've been enabled")
	}
	if config.Endpoints[0].Alerts[2].IsEnabled() {
		t.Error("The alert should've been disabled")
	}
	if config.Endpoints[0].Alerts[0].GetDescription() != "description" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "description", config.Endpoints[0].Alerts[0].GetDescription())
	}
	if config.Endpoints[0].Alerts[1].GetDescription() != "wow" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "description", config.Endpoints[0].Alerts[1].GetDescription())
	}
	if config.Endpoints[0].Alerts[2].GetDescription() != "description" {
		t.Errorf("The description of the alert should've been %s, but it was %s", "description", config.Endpoints[0].Alerts[2].GetDescription())
	}
	if config.Endpoints[0].Alerts[0].FailureThreshold != 10 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 10, config.Endpoints[0].Alerts[0].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[1].FailureThreshold != 20 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 20, config.Endpoints[0].Alerts[1].FailureThreshold)
	}
	if config.Endpoints[0].Alerts[2].FailureThreshold != 30 {
		t.Errorf("The failure threshold of the alert should've been %d, but it was %d", 30, config.Endpoints[0].Alerts[2].FailureThreshold)
	}
}

func TestParseAndValidateConfigBytesWithInvalidPagerDutyAlertingConfig(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  pagerduty:
    integration-key: "INVALID_KEY"
endpoints:
  - name: website
    url: https://twin.sh/health
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
	if config.Alerting.PagerDuty != nil {
		t.Fatal("PagerDuty alerting config should've been set to nil, because its IsValid() method returned false and therefore alerting.Config.SetAlertingProviderToNil() should've been called")
	}
}
func TestParseAndValidateConfigBytesWithInvalidPushoverAlertingConfig(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  pushover:
    application-token: "INVALID_TOKEN"
endpoints:
  - name: website
    url: https://twin.sh/health
    alerts:
      - type: pushover
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
	if config.Alerting.Pushover != nil {
		t.Fatal("Pushover alerting config should've been set to nil, because its IsValid() method returned false and therefore alerting.Config.SetAlertingProviderToNil() should've been called")
	}
}

func TestParseAndValidateConfigBytesWithCustomAlertingConfig(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  custom:
    url: "https://example.com"
    body: |
      {
        "text": "[ALERT_TRIGGERED_OR_RESOLVED]: [ENDPOINT_NAME] - [ALERT_DESCRIPTION]"
      }
endpoints:
  - name: website
    url: https://twin.sh/health
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
		t.Fatal("Custom alerting config shouldn't have been nil")
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
    body: "[ALERT_TRIGGERED_OR_RESOLVED]: [ENDPOINT_NAME] - [ALERT_DESCRIPTION]"
endpoints:
  - name: website
    url: https://twin.sh/health
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
		t.Fatal("Custom alerting config shouldn't have been nil")
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
}

func TestParseAndValidateConfigBytesWithCustomAlertingConfigAndOneCustomPlaceholderValue(t *testing.T) {
	config, err := parseAndValidateConfigBytes([]byte(`
alerting:
  custom:
    placeholders:
      ALERT_TRIGGERED_OR_RESOLVED:
        TRIGGERED: "partial_outage"
    url: "https://example.com"
    body: "[ALERT_TRIGGERED_OR_RESOLVED]: [ENDPOINT_NAME] - [ALERT_DESCRIPTION]"
endpoints:
  - name: website
    url: https://twin.sh/health
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
		t.Fatal("Custom alerting config shouldn't have been nil")
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
}

func TestParseAndValidateConfigBytesWithInvalidEndpointName(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
endpoints:
  - name: ""
    url: https://twin.sh/health
    conditions:
      - "[STATUS] == 200"
`))
	if err == nil {
		t.Error("should've returned an error")
	}
}

func TestParseAndValidateConfigBytesWithInvalidStorageConfig(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(`
storage:
  type: sqlite
endpoints:
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
endpoints:
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
endpoints:
  - name: website
    url: https://twin.sh/health
    conditions:
      - "[STATUS] == 200"
`))
	if err == nil {
		t.Error("should've returned an error")
	}
}

func TestParseAndValidateConfigBytesWithValidSecurityConfig(t *testing.T) {
	const expectedUsername = "admin"
	const expectedPasswordHash = "JDJhJDEwJHRiMnRFakxWazZLdXBzRERQazB1TE8vckRLY05Yb1hSdnoxWU0yQ1FaYXZRSW1McmladDYu"
	config, err := parseAndValidateConfigBytes([]byte(fmt.Sprintf(`debug: true
security:
  basic:
    username: "%s"
    password-bcrypt-base64: "%s"
endpoints:
  - name: website
    url: https://twin.sh/health
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
	if config.Security.Basic.PasswordBcryptHashBase64Encoded != expectedPasswordHash {
		t.Errorf("config.Security.Basic.PasswordBcryptHashBase64Encoded should've been %s, but was %s", expectedPasswordHash, config.Security.Basic.PasswordBcryptHashBase64Encoded)
	}
}

func TestParseAndValidateConfigBytesWithLiteralDollarSign(t *testing.T) {
	os.Setenv("GATUS_TestParseAndValidateConfigBytesWithLiteralDollarSign", "whatever")
	config, err := parseAndValidateConfigBytes([]byte(`
endpoints:
  - name: website
    url: https://twin.sh/health
    conditions:
      - "[BODY] == $$GATUS_TestParseAndValidateConfigBytesWithLiteralDollarSign"
      - "[BODY] == $GATUS_TestParseAndValidateConfigBytesWithLiteralDollarSign"
`))
	if err != nil {
		t.Error("expected no error, got", err.Error())
	}
	if config == nil {
		t.Fatal("Config shouldn't have been nil")
	}
	if config.Endpoints[0].URL != "https://twin.sh/health" {
		t.Errorf("URL should have been %s", "https://twin.sh/health")
	}
	if config.Endpoints[0].Conditions[0] != "[BODY] == $GATUS_TestParseAndValidateConfigBytesWithLiteralDollarSign" {
		t.Errorf("Condition should have been %s", "[BODY] == $GATUS_TestParseAndValidateConfigBytesWithLiteralDollarSign")
	}
	if config.Endpoints[0].Conditions[1] != "[BODY] == whatever" {
		t.Errorf("Condition should have been %s", "[BODY] == whatever")
	}
}

func TestParseAndValidateConfigBytesWithNoEndpoints(t *testing.T) {
	_, err := parseAndValidateConfigBytes([]byte(``))
	if err != ErrNoEndpointInConfig {
		t.Error("The error returned should have been of type ErrNoEndpointInConfig")
	}
}

func TestGetAlertingProviderByAlertType(t *testing.T) {
	alertingConfig := &alerting.Config{
		Custom:      &custom.AlertProvider{},
		Discord:     &discord.AlertProvider{},
		Email:       &email.AlertProvider{},
		GitHub:      &github.AlertProvider{},
		GoogleChat:  &googlechat.AlertProvider{},
		Matrix:      &matrix.AlertProvider{},
		Mattermost:  &mattermost.AlertProvider{},
		Messagebird: &messagebird.AlertProvider{},
		Ntfy:        &ntfy.AlertProvider{},
		Opsgenie:    &opsgenie.AlertProvider{},
		PagerDuty:   &pagerduty.AlertProvider{},
		Pushover:    &pushover.AlertProvider{},
		Slack:       &slack.AlertProvider{},
		Telegram:    &telegram.AlertProvider{},
		Twilio:      &twilio.AlertProvider{},
		Teams:       &teams.AlertProvider{},
	}
	scenarios := []struct {
		alertType alert.Type
		expected  provider.AlertProvider
	}{
		{alertType: alert.TypeCustom, expected: alertingConfig.Custom},
		{alertType: alert.TypeDiscord, expected: alertingConfig.Discord},
		{alertType: alert.TypeEmail, expected: alertingConfig.Email},
		{alertType: alert.TypeGitHub, expected: alertingConfig.GitHub},
		{alertType: alert.TypeGoogleChat, expected: alertingConfig.GoogleChat},
		{alertType: alert.TypeMatrix, expected: alertingConfig.Matrix},
		{alertType: alert.TypeMattermost, expected: alertingConfig.Mattermost},
		{alertType: alert.TypeMessagebird, expected: alertingConfig.Messagebird},
		{alertType: alert.TypeNtfy, expected: alertingConfig.Ntfy},
		{alertType: alert.TypeOpsgenie, expected: alertingConfig.Opsgenie},
		{alertType: alert.TypePagerDuty, expected: alertingConfig.PagerDuty},
		{alertType: alert.TypePushover, expected: alertingConfig.Pushover},
		{alertType: alert.TypeSlack, expected: alertingConfig.Slack},
		{alertType: alert.TypeTelegram, expected: alertingConfig.Telegram},
		{alertType: alert.TypeTwilio, expected: alertingConfig.Twilio},
		{alertType: alert.TypeTeams, expected: alertingConfig.Teams},
	}
	for _, scenario := range scenarios {
		t.Run(string(scenario.alertType), func(t *testing.T) {
			if alertingConfig.GetAlertingProviderByAlertType(scenario.alertType) != scenario.expected {
				t.Errorf("expected %s configuration", scenario.alertType)
			}
		})
	}
}

package config

import (
	"fmt"
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
      - "$STATUS == 200"
  - name: github
    url: https://api.github.com/healthz
    conditions:
      - "$STATUS != 400"
      - "$STATUS != 500"
`))
	if err != nil {
		t.Error("No error should've been returned")
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
      - "$STATUS == 200"
`))
	if err != nil {
		t.Error("No error should've been returned")
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
      - "$STATUS == 200"
`))
	if err != nil {
		t.Error("No error should've been returned")
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

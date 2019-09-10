package config

import (
	"fmt"
	"testing"
	"time"
)

func TestParseConfigBytes(t *testing.T) {
	config := ParseConfigBytes([]byte(`
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    interval: 15s
    failure-threshold: 3
    conditions:
      - "$STATUS == 200"
  - name: github
    url: https://api.github.com/healthz
    conditions:
      - "$STATUS != 400"
      - "$STATUS != 500"
`))
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
	if config.Services[0].FailureThreshold != 3 {
		t.Errorf("FailureThreshold should have been %d", 3)
	}
	if config.Services[1].FailureThreshold != 1 {
		t.Errorf("FailureThreshold should have been %d, because it is the default value", 1)
	}
	if len(config.Services[0].Conditions) != 1 {
		t.Errorf("There should have been %d conditions", 1)
	}
	if len(config.Services[1].Conditions) != 2 {
		t.Errorf("There should have been %d conditions", 2)
	}
}

func TestParseConfigBytesDefault(t *testing.T) {
	config := ParseConfigBytes([]byte(`
services:
  - name: twinnation
    url: https://twinnation.org/actuator/health
    conditions:
      - "$STATUS == 200"
`))
	if config.Services[0].Url != "https://twinnation.org/actuator/health" {
		t.Errorf("URL should have been %s", "https://twinnation.org/actuator/health")
	}
	if config.Services[0].Interval != 10*time.Second {
		t.Errorf("Interval should have been %s, because it is the default value", 10*time.Second)
	}
	if config.Services[0].FailureThreshold != 1 {
		t.Errorf("FailureThreshold should have been %d, because it is the default value", 1)
	}
}

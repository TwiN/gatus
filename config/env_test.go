package config

import (
	"os"
	"testing"
)

func TestParseAndValidateConfigBytes_EnvDefault(t *testing.T) {
	// Create a temporary YAML config with environment variable using default value syntax
	yamlConfig := `
endpoints:
  - name: test-endpoint
    group: core
    url: "http://example.com"
    interval: 5m
    conditions:
      - "[STATUS] == ${EXPECTED_STATUS:-201}"
`

	// Ensure the environment variable is NOT set
	os.Unsetenv("EXPECTED_STATUS")

	// Parse the configuration
	cfg, err := parseAndValidateConfigBytes([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify that the default value was applied
	expectedCondition := "[STATUS] == 201"
	if len(cfg.Endpoints) == 0 {
		t.Fatal("No endpoints parsed")
	}
	actualCondition := string(cfg.Endpoints[0].Conditions[0])

	if actualCondition != expectedCondition {
		t.Errorf("Expected condition '%s', got '%s'", expectedCondition, actualCondition)
	}
}

func TestParseAndValidateConfigBytes_EnvSet(t *testing.T) {
	// Create a temporary YAML config with environment variable using default value syntax
	yamlConfig := `
endpoints:
  - name: test-endpoint-set
    group: core
    url: "http://example.com"
    interval: 5m
    conditions:
      - "[STATUS] == ${EXPECTED_STATUS_SET:-201}"
`

	// Ensure the environment variable IS set
	os.Setenv("EXPECTED_STATUS_SET", "200")
	defer os.Unsetenv("EXPECTED_STATUS_SET")

	// Parse the configuration
	cfg, err := parseAndValidateConfigBytes([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	// Verify that the env value was applied (override default)
	expectedCondition := "[STATUS] == 200"
	if len(cfg.Endpoints) == 0 {
		t.Fatal("No endpoints parsed")
	}
	actualCondition := string(cfg.Endpoints[0].Conditions[0])

	if actualCondition != expectedCondition {
		t.Errorf("Expected condition '%s', got '%s'", expectedCondition, actualCondition)
	}
}

package config

import (
	"os"
	"testing"
)

func TestExpandEnv(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		env      map[string]string
		expected string
	}{
		{
			name:     "simple variable with default value when unset",
			input:    "${UNSET_VAR:-default_value}",
			env:      map[string]string{},
			expected: "default_value",
		},
		{
			name:     "simple variable with default value when set",
			input:    "${SET_VAR:-default_value}",
			env:      map[string]string{"SET_VAR": "actual_value"},
			expected: "actual_value",
		},
		{
			name:     "simple variable with default value when empty",
			input:    "${EMPTY_VAR:-default_value}",
			env:      map[string]string{"EMPTY_VAR": ""},
			expected: "default_value",
		},
		{
			name:     "variable without default when set",
			input:    "${SET_VAR}",
			env:      map[string]string{"SET_VAR": "value"},
			expected: "value",
		},
		{
			name:     "variable without default when unset",
			input:    "${UNSET_VAR}",
			env:      map[string]string{},
			expected: "",
		},
		{
			name:     "dollar VAR syntax when set",
			input:    "$SET_VAR",
			env:      map[string]string{"SET_VAR": "value"},
			expected: "value",
		},
		{
			name:     "dollar VAR syntax when unset",
			input:    "$UNSET_VAR",
			env:      map[string]string{},
			expected: "",
		},
		{
			name:     "mixed text and variable with default",
			input:    "prefix_${VAR:-default}_suffix",
			env:      map[string]string{},
			expected: "prefix_default_suffix",
		},
		{
			name:     "multiple variables with defaults",
			input:    "${VAR1:-default1} ${VAR2:-default2}",
			env:      map[string]string{"VAR1": "value1"},
			expected: "value1 default2",
		},
		{
			name:     "default value with special characters",
			input:    "${VAR:-http://localhost:8080}",
			env:      map[string]string{},
			expected: "http://localhost:8080",
		},
		{
			name:     "default value with spaces",
			input:    "${VAR:-default value with spaces}",
			env:      map[string]string{},
			expected: "default value with spaces",
		},
		{
			name:     "empty default value",
			input:    "${VAR:-}",
			env:      map[string]string{},
			expected: "",
		},
		{
			name:     "colon in default value",
			input:    "${VAR:-value:with:colons}",
			env:      map[string]string{},
			expected: "value:with:colons",
		},
		{
			name:     "literal dollar sign not affected",
			input:    "price is $$50",
			env:      map[string]string{},
			expected: "price is $50",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}
			// Clear any variables that should be unset
			testVars := []string{"UNSET_VAR", "SET_VAR", "EMPTY_VAR", "VAR", "VAR1", "VAR2"}
			for _, v := range testVars {
				if _, ok := tt.env[v]; !ok {
					os.Unsetenv(v)
				}
			}

			result := expandEnv(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnv(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExpandEnvInConfig(t *testing.T) {
	// Test that expandEnv works correctly in the config parsing pipeline
	os.Setenv("TEST_URL", "https://example.com")
	os.Setenv("TEST_NAME", "test-endpoint")
	defer os.Unsetenv("TEST_URL")
	defer os.Unsetenv("TEST_NAME")

	configYAML := `
endpoints:
  - name: ${TEST_NAME}
    url: ${TEST_URL}
    conditions:
      - "[STATUS] == 200"
  - name: with-default
    url: ${UNSET_URL:-https://default.com}
    conditions:
      - "[STATUS] == 200"
`

	config, err := parseAndValidateConfigBytes([]byte(configYAML))
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	if len(config.Endpoints) != 2 {
		t.Fatalf("Expected 2 endpoints, got %d", len(config.Endpoints))
	}

	// Check first endpoint with environment variables
	if config.Endpoints[0].Name != "test-endpoint" {
		t.Errorf("Expected endpoint name 'test-endpoint', got %q", config.Endpoints[0].Name)
	}
	if config.Endpoints[0].URL != "https://example.com" {
		t.Errorf("Expected endpoint URL 'https://example.com', got %q", config.Endpoints[0].URL)
	}

	// Check second endpoint with default value
	if config.Endpoints[1].Name != "with-default" {
		t.Errorf("Expected endpoint name 'with-default', got %q", config.Endpoints[1].Name)
	}
	if config.Endpoints[1].URL != "https://default.com" {
		t.Errorf("Expected endpoint URL 'https://default.com', got %q", config.Endpoints[1].URL)
	}
}

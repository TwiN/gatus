package security

import "testing"

func TestMetricsConfig_isProtected(t *testing.T) {
	type Scenario struct {
		Name     string
		Config   *MetricsConfig
		Expected bool
	}
	scenarios := []Scenario{
		{
			Name:     "empty",
			Config:   &MetricsConfig{},
			Expected: false,
		},
		{
			Name:     "protected",
			Config:   &MetricsConfig{Protected: true},
			Expected: true,
		},
		{
			Name:     "tokens",
			Config:   &MetricsConfig{Tokens: []string{"token"}},
			Expected: true,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if isProtected := scenario.Config.isProtected(); isProtected != scenario.Expected {
				t.Errorf("expected %v, got %v", scenario.Expected, isProtected)
			}
		})
	}
}

func TestMetricsConfig_isValid(t *testing.T) {
	type Scenario struct {
		Name        string
		Config      *MetricsConfig
		HasProvider bool
		ExpectValid bool
	}
	scenarios := []Scenario{
		{
			Name:        "empty",
			Config:      &MetricsConfig{},
			HasProvider: false,
			ExpectValid: true,
		},
		{
			Name:        "protected-with-provider",
			Config:      &MetricsConfig{Protected: true},
			HasProvider: true,
			ExpectValid: true,
		},
		{
			Name:        "protected-without-provider",
			Config:      &MetricsConfig{Protected: true},
			HasProvider: false,
			ExpectValid: false,
		},
		{
			Name:        "protected-with-tokens-without-provider",
			Config:      &MetricsConfig{Protected: true, Tokens: []string{"token"}},
			HasProvider: false,
			ExpectValid: true,
		},
		{
			Name:        "tokens-without-provider",
			Config:      &MetricsConfig{Tokens: []string{"token"}},
			HasProvider: false,
			ExpectValid: true,
		},
		{
			Name:        "empty-token",
			Config:      &MetricsConfig{Tokens: []string{"token", ""}},
			HasProvider: true,
			ExpectValid: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if isValid := scenario.Config.isValid(scenario.HasProvider); isValid != scenario.ExpectValid {
				t.Errorf("expected %v, got %v", scenario.ExpectValid, isValid)
			}
		})
	}
}

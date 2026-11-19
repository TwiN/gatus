package signalgrid

import (
	"net/url"
	"testing"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func boolPtr(value bool) *bool {
	return &value
}

func TestAlertProvider_Validate(t *testing.T) {
	scenarios := []struct {
		name     string
		provider AlertProvider
		expected bool
	}{
		{
			name: "valid",
			provider: AlertProvider{
				DefaultConfig: Config{
					ClientKey: "client-key",
					Channel:   "channel",
				},
			},
			expected: true,
		},
		{
			name: "missing-client-key",
			provider: AlertProvider{
				DefaultConfig: Config{
					Channel: "channel",
				},
			},
			expected: false,
		},
		{
			name: "missing-channel",
			provider: AlertProvider{
				DefaultConfig: Config{
					ClientKey: "client-key",
				},
			},
			expected: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			err := scenario.provider.Validate()

			if (err == nil) != scenario.expected {
				t.Errorf(
					"expected valid=%t, got error=%v",
					scenario.expected,
					err,
				)
			}
		})
	}
}

func TestAlertProvider_buildRequestPayload_Triggered(t *testing.T) {
	provider := AlertProvider{}

	description := "Production API"

	cfg := &Config{
		ClientKey: "client-key",
		Channel:   "channel",
		Critical:  boolPtr(true),
	}

	ep := &endpoint.Endpoint{
		Name: "api",
	}

	alertConfig := &alert.Alert{
		Description:      &description,
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}

	result := &endpoint.Result{
		ConditionResults: []*endpoint.ConditionResult{
			{
				Condition: "[CONNECTED] == true",
				Success:   false,
			},
			{
				Condition: "[STATUS] == 200",
				Success:   false,
			},
		},
	}

	payload := provider.buildRequestPayload(
		cfg,
		ep,
		alertConfig,
		result,
		false,
	)

	assertPayloadValue(t, payload, "client_key", "client-key")
	assertPayloadValue(t, payload, "channel", "channel")
	assertPayloadValue(t, payload, "title", "Gatus: api")
	assertPayloadValue(t, payload, "type", "CRIT")
	assertPayloadValue(t, payload, "critical", "true")

	expectedBody := "An alert for `api` has been triggered due to having failed 3 time(s) in a row" +
		" with the following description: Production API" +
		"\n✕ - [CONNECTED] == true" +
		"\n✕ - [STATUS] == 200"

	assertPayloadValue(t, payload, "body", expectedBody)
}

func TestAlertProvider_buildRequestPayload_Resolved(t *testing.T) {
	provider := AlertProvider{}

	cfg := &Config{
		ClientKey: "client-key",
		Channel:   "channel",
		Critical:  boolPtr(true),
	}

	ep := &endpoint.Endpoint{
		Name: "api",
	}

	alertConfig := &alert.Alert{
		FailureThreshold: 3,
		SuccessThreshold: 2,
	}

	result := &endpoint.Result{
		ConditionResults: []*endpoint.ConditionResult{
			{
				Condition: "[CONNECTED] == true",
				Success:   true,
			},
			{
				Condition: "[STATUS] == 200",
				Success:   true,
			},
		},
	}

	payload := provider.buildRequestPayload(
		cfg,
		ep,
		alertConfig,
		result,
		true,
	)

	assertPayloadValue(t, payload, "type", "SUCCESS")

	// Resolved notifications should never be sent as critical notifications.
	assertPayloadValue(t, payload, "critical", "false")

	expectedBody := "An alert for `api` has been resolved after passing successfully 2 time(s) in a row" +
		"\n✓ - [CONNECTED] == true" +
		"\n✓ - [STATUS] == 200"

	assertPayloadValue(t, payload, "body", expectedBody)
}

func TestAlertProvider_buildRequestPayload_NonCritical(t *testing.T) {
	provider := AlertProvider{}

	cfg := &Config{
		ClientKey: "client-key",
		Channel:   "channel",
		Critical:  boolPtr(false),
	}

	payload := provider.buildRequestPayload(
		cfg,
		&endpoint.Endpoint{Name: "api"},
		&alert.Alert{
			FailureThreshold: 1,
		},
		&endpoint.Result{},
		false,
	)

	assertPayloadValue(t, payload, "type", "CRIT")
	assertPayloadValue(t, payload, "critical", "false")
}

func TestAlertProvider_GetDefaultAlert(t *testing.T) {
	provider := AlertProvider{
		DefaultAlert: &alert.Alert{},
	}

	if provider.GetDefaultAlert() != provider.DefaultAlert {
		t.Error("expected default alert to be returned")
	}
}

func TestAlertProvider_GetConfig(t *testing.T) {
	provider := AlertProvider{
		DefaultConfig: Config{
			ClientKey: "default-client-key",
			Channel:   "default-channel",
			Critical:  boolPtr(true),
		},
	}

	t.Run("default-config", func(t *testing.T) {
		cfg, err := provider.GetConfig("", &alert.Alert{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.ClientKey != "default-client-key" {
			t.Errorf(
				"expected client key %q, got %q",
				"default-client-key",
				cfg.ClientKey,
			)
		}

		if cfg.Channel != "default-channel" {
			t.Errorf(
				"expected channel %q, got %q",
				"default-channel",
				cfg.Channel,
			)
		}

		if !cfg.IsCritical() {
			t.Error("expected critical to be true")
		}
	})

	t.Run("provider-override", func(t *testing.T) {
		inputAlert := alert.Alert{
			ProviderOverride: map[string]any{
				"client-key": "override-client-key",
				"channel":    "override-channel",
				"critical":   false,
			},
		}

		cfg, err := provider.GetConfig("", &inputAlert)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.ClientKey != "override-client-key" {
			t.Errorf(
				"expected client key %q, got %q",
				"override-client-key",
				cfg.ClientKey,
			)
		}

		if cfg.Channel != "override-channel" {
			t.Errorf(
				"expected channel %q, got %q",
				"override-channel",
				cfg.Channel,
			)
		}

		if cfg.IsCritical() {
			t.Error("expected critical override to be false")
		}

		if err := provider.ValidateOverrides("", &inputAlert); err != nil {
			t.Errorf("unexpected validation error: %v", err)
		}
	})
}

func assertPayloadValue(
	t *testing.T,
	payload url.Values,
	key string,
	expected string,
) {
	t.Helper()

	actual := payload.Get(key)

	if actual != expected {
		t.Errorf(
			"expected %s=%q, got %q",
			key,
			expected,
			actual,
		)
	}
}

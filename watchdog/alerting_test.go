package watchdog

import (
	"os"
	"testing"

	"github.com/TwiN/gatus/v3/alerting"
	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/alerting/provider/custom"
	"github.com/TwiN/gatus/v3/alerting/provider/pagerduty"
	"github.com/TwiN/gatus/v3/config"
	"github.com/TwiN/gatus/v3/core"
)

func TestHandleAlerting(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twin.sh/health",
				Method: "GET",
			},
		},
	}
	enabled := true
	endpoint := &core.Endpoint{
		URL: "https://example.com",
		Alerts: []*alert.Alert{
			{
				Type:             alert.TypeCustom,
				Enabled:          &enabled,
				FailureThreshold: 2,
				SuccessThreshold: 3,
				SendOnResolved:   &enabled,
				Triggered:        false,
			},
		},
	}

	verify(t, endpoint, 0, 0, false, "The alert shouldn't start triggered")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 1, 0, false, "The alert shouldn't have triggered")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 2, 0, true, "The alert should've triggered")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 3, 0, true, "The alert should still be triggered")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 4, 0, true, "The alert should still be triggered")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 1, true, "The alert should still be triggered (because endpoint.Alerts[0].SuccessThreshold is 3)")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 2, true, "The alert should still be triggered (because endpoint.Alerts[0].SuccessThreshold is 3)")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 3, false, "The alert should've been resolved")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 4, false, "The alert should no longer be triggered")
}

func TestHandleAlertingWhenAlertingConfigIsNil(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()
	HandleAlerting(nil, nil, nil, true)
}

func TestHandleAlertingWithBadAlertProvider(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	enabled := true
	endpoint := &core.Endpoint{
		URL: "http://example.com",
		Alerts: []*alert.Alert{
			{
				Type:             alert.TypeCustom,
				Enabled:          &enabled,
				FailureThreshold: 1,
				SuccessThreshold: 1,
				SendOnResolved:   &enabled,
				Triggered:        false,
			},
		},
	}

	verify(t, endpoint, 0, 0, false, "The alert shouldn't start triggered")
	HandleAlerting(endpoint, &core.Result{Success: false}, &alerting.Config{}, false)
	verify(t, endpoint, 1, 0, false, "The alert shouldn't have triggered")
	HandleAlerting(endpoint, &core.Result{Success: false}, &alerting.Config{}, false)
	verify(t, endpoint, 2, 0, false, "The alert shouldn't have triggered, because the provider wasn't configured properly")
}

func TestHandleAlertingWhenTriggeredAlertIsAlmostResolvedButendpointStartFailingAgain(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twin.sh/health",
				Method: "GET",
			},
		},
	}
	enabled := true
	endpoint := &core.Endpoint{
		URL: "http://example.com",
		Alerts: []*alert.Alert{
			{
				Type:             alert.TypeCustom,
				Enabled:          &enabled,
				FailureThreshold: 2,
				SuccessThreshold: 3,
				SendOnResolved:   &enabled,
				Triggered:        true,
			},
		},
		NumberOfFailuresInARow: 1,
	}

	// This test simulate an alert that was already triggered
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 2, 0, true, "The alert was already triggered at the beginning of this test")
}

func TestHandleAlertingWhenTriggeredAlertIsResolvedButSendOnResolvedIsFalse(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twin.sh/health",
				Method: "GET",
			},
		},
	}
	enabled := true
	disabled := false
	endpoint := &core.Endpoint{
		URL: "http://example.com",
		Alerts: []*alert.Alert{
			{
				Type:             alert.TypeCustom,
				Enabled:          &enabled,
				FailureThreshold: 1,
				SuccessThreshold: 1,
				SendOnResolved:   &disabled,
				Triggered:        true,
			},
		},
		NumberOfFailuresInARow: 1,
	}

	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 1, false, "The alert should've been resolved")
}

func TestHandleAlertingWhenTriggeredAlertIsResolvedPagerDuty(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			PagerDuty: &pagerduty.AlertProvider{
				IntegrationKey: "00000000000000000000000000000000",
			},
		},
	}
	enabled := true
	endpoint := &core.Endpoint{
		URL: "http://example.com",
		Alerts: []*alert.Alert{
			{
				Type:             alert.TypePagerDuty,
				Enabled:          &enabled,
				FailureThreshold: 1,
				SuccessThreshold: 1,
				SendOnResolved:   &enabled,
				Triggered:        false,
			},
		},
		NumberOfFailuresInARow: 0,
	}

	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 1, 0, true, "")

	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 1, false, "The alert should've been resolved")
}

func TestHandleAlertingWithProviderThatReturnsAnError(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twin.sh/health",
				Method: "GET",
			},
		},
	}
	enabled := true
	endpoint := &core.Endpoint{
		URL: "http://example.com",
		Alerts: []*alert.Alert{
			{
				Type:             alert.TypeCustom,
				Enabled:          &enabled,
				FailureThreshold: 2,
				SuccessThreshold: 2,
				SendOnResolved:   &enabled,
				Triggered:        false,
			},
		},
	}

	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 1, 0, false, "")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 2, 0, false, "The alert should have failed to trigger, because the alert provider is returning an error")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 3, 0, false, "The alert should still not be triggered, because the alert provider is still returning an error")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 4, 0, false, "The alert should still not be triggered, because the alert provider is still returning an error")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 5, 0, true, "The alert should've been triggered because the alert provider is no longer returning an error")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 1, true, "The alert should've still been triggered")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 2, false, "The alert should've been resolved DESPITE THE ALERT PROVIDER RETURNING AN ERROR. See Alert.Triggered for further explanation.")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")

	// Make sure that everything's working as expected after a rough patch
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 1, 0, false, "")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 2, 0, true, "The alert should have triggered")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 1, true, "The alert should still be triggered")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 2, false, "The alert should have been resolved")
}

func TestHandleAlertingWithProviderThatOnlyReturnsErrorOnResolve(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twin.sh/health",
				Method: "GET",
			},
		},
	}
	enabled := true
	endpoint := &core.Endpoint{
		URL: "http://example.com",
		Alerts: []*alert.Alert{
			{
				Type:             alert.TypeCustom,
				Enabled:          &enabled,
				FailureThreshold: 1,
				SuccessThreshold: 1,
				SendOnResolved:   &enabled,
				Triggered:        false,
			},
		},
	}

	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 1, 0, true, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 1, false, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 1, 0, true, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 1, false, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")

	// Make sure that everything's working as expected after a rough patch
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 1, 0, true, "")
	HandleAlerting(endpoint, &core.Result{Success: false}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 2, 0, true, "")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 1, false, "")
	HandleAlerting(endpoint, &core.Result{Success: true}, cfg.Alerting, cfg.Debug)
	verify(t, endpoint, 0, 2, false, "")
}

func verify(t *testing.T, endpoint *core.Endpoint, expectedNumberOfFailuresInARow, expectedNumberOfSuccessInARow int, expectedTriggered bool, expectedTriggeredReason string) {
	if endpoint.NumberOfFailuresInARow != expectedNumberOfFailuresInARow {
		t.Fatalf("endpoint.NumberOfFailuresInARow should've been %d, got %d", expectedNumberOfFailuresInARow, endpoint.NumberOfFailuresInARow)
	}
	if endpoint.NumberOfSuccessesInARow != expectedNumberOfSuccessInARow {
		t.Fatalf("endpoint.NumberOfSuccessesInARow should've been %d, got %d", expectedNumberOfSuccessInARow, endpoint.NumberOfSuccessesInARow)
	}
	if endpoint.Alerts[0].Triggered != expectedTriggered {
		if len(expectedTriggeredReason) != 0 {
			t.Fatal(expectedTriggeredReason)
		} else {
			if expectedTriggered {
				t.Fatal("The alert should've been triggered")
			} else {
				t.Fatal("The alert shouldn't have been triggered")
			}
		}
	}
}

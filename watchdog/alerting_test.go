package watchdog

import (
	"os"
	"testing"

	"github.com/TwinProduction/gatus/alerting"
	"github.com/TwinProduction/gatus/alerting/provider/custom"
	"github.com/TwinProduction/gatus/alerting/provider/pagerduty"
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
)

func TestHandleAlerting(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twinnation.org/health",
				Method: "GET",
			},
		},
	}
	config.Set(cfg)
	service := &core.Service{
		URL: "http://example.com",
		Alerts: []*core.Alert{
			{
				Type:             core.CustomAlert,
				Enabled:          true,
				FailureThreshold: 2,
				SuccessThreshold: 3,
				SendOnResolved:   true,
				Triggered:        false,
			},
		},
	}

	verify(t, service, 0, 0, false, "The alert shouldn't start triggered")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 1, 0, false, "The alert shouldn't have triggered")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 2, 0, true, "The alert should've triggered")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 3, 0, true, "The alert should still be triggered")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 4, 0, true, "The alert should still be triggered")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 1, true, "The alert should still be triggered (because service.Alerts[0].SuccessThreshold is 3)")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 2, true, "The alert should still be triggered (because service.Alerts[0].SuccessThreshold is 3)")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 3, false, "The alert should've been resolved")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 4, false, "The alert should no longer be triggered")
}

func TestHandleAlertingWhenAlertingConfigIsNil(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug:    true,
		Alerting: nil,
	}
	config.Set(cfg)
	HandleAlerting(nil, nil)
}

func TestHandleAlertingWithBadAlertProvider(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Alerting: &alerting.Config{},
	}
	config.Set(cfg)
	service := &core.Service{
		URL: "http://example.com",
		Alerts: []*core.Alert{
			{
				Type:             core.CustomAlert,
				Enabled:          true,
				FailureThreshold: 1,
				SuccessThreshold: 1,
				SendOnResolved:   true,
				Triggered:        false,
			},
		},
	}

	verify(t, service, 0, 0, false, "The alert shouldn't start triggered")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 1, 0, false, "The alert shouldn't have triggered")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 2, 0, false, "The alert shouldn't have triggered, because the provider wasn't configured properly")
}

func TestHandleAlertingWhenTriggeredAlertIsAlmostResolvedButServiceStartFailingAgain(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twinnation.org/health",
				Method: "GET",
			},
		},
	}
	config.Set(cfg)
	service := &core.Service{
		URL: "http://example.com",
		Alerts: []*core.Alert{
			{
				Type:             core.CustomAlert,
				Enabled:          true,
				FailureThreshold: 2,
				SuccessThreshold: 3,
				SendOnResolved:   true,
				Triggered:        true,
			},
		},
		NumberOfFailuresInARow: 1,
	}

	// This test simulate an alert that was already triggered
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 2, 0, true, "The alert was already triggered at the beginning of this test")
}

func TestHandleAlertingWhenTriggeredAlertIsResolvedButSendOnResolvedIsFalse(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twinnation.org/health",
				Method: "GET",
			},
		},
	}
	config.Set(cfg)
	service := &core.Service{
		URL: "http://example.com",
		Alerts: []*core.Alert{
			{
				Type:             core.CustomAlert,
				Enabled:          true,
				FailureThreshold: 1,
				SuccessThreshold: 1,
				SendOnResolved:   false,
				Triggered:        true,
			},
		},
		NumberOfFailuresInARow: 1,
	}

	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 1, false, "The alert should've been resolved")
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
	config.Set(cfg)
	service := &core.Service{
		URL: "http://example.com",
		Alerts: []*core.Alert{
			{
				Type:             core.PagerDutyAlert,
				Enabled:          true,
				FailureThreshold: 1,
				SuccessThreshold: 1,
				SendOnResolved:   true,
				Triggered:        false,
			},
		},
		NumberOfFailuresInARow: 0,
	}

	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 1, 0, true, "")

	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 1, false, "The alert should've been resolved")
}

func TestHandleAlertingWithProviderThatReturnsAnError(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twinnation.org/health",
				Method: "GET",
			},
		},
	}
	config.Set(cfg)
	service := &core.Service{
		URL: "http://example.com",
		Alerts: []*core.Alert{
			{
				Type:             core.CustomAlert,
				Enabled:          true,
				FailureThreshold: 2,
				SuccessThreshold: 2,
				SendOnResolved:   true,
				Triggered:        false,
			},
		},
	}

	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 1, 0, false, "")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 2, 0, false, "The alert should have failed to trigger, because the alert provider is returning an error")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 3, 0, false, "The alert should still not be triggered, because the alert provider is still returning an error")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 4, 0, false, "The alert should still not be triggered, because the alert provider is still returning an error")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 5, 0, true, "The alert should've been triggered because the alert provider is no longer returning an error")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 1, true, "The alert should've still been triggered")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 2, false, "The alert should've been resolved DESPITE THE ALERT PROVIDER RETURNING AN ERROR. See Alert.Triggered for further explanation.")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")

	// Make sure that everything's working as expected after a rough patch
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 1, 0, false, "")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 2, 0, true, "The alert should have triggered")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 1, true, "The alert should still be triggered")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 2, false, "The alert should have been resolved")
}

func TestHandleAlertingWithProviderThatOnlyReturnsErrorOnResolve(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Debug: true,
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				URL:    "https://twinnation.org/health",
				Method: "GET",
			},
		},
	}
	config.Set(cfg)
	service := &core.Service{
		URL: "http://example.com",
		Alerts: []*core.Alert{
			{
				Type:             core.CustomAlert,
				Enabled:          true,
				FailureThreshold: 1,
				SuccessThreshold: 1,
				SendOnResolved:   true,
				Triggered:        false,
			},
		},
	}

	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 1, 0, true, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 1, false, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 1, 0, true, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 1, false, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")

	// Make sure that everything's working as expected after a rough patch
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 1, 0, true, "")
	HandleAlerting(service, &core.Result{Success: false})
	verify(t, service, 2, 0, true, "")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 1, false, "")
	HandleAlerting(service, &core.Result{Success: true})
	verify(t, service, 0, 2, false, "")
}

func verify(t *testing.T, service *core.Service, expectedNumberOfFailuresInARow, expectedNumberOfSuccessInARow int, expectedTriggered bool, expectedTriggeredReason string) {
	if service.NumberOfFailuresInARow != expectedNumberOfFailuresInARow {
		t.Fatalf("service.NumberOfFailuresInARow should've been %d, got %d", expectedNumberOfFailuresInARow, service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != expectedNumberOfSuccessInARow {
		t.Fatalf("service.NumberOfSuccessesInARow should've been %d, got %d", expectedNumberOfSuccessInARow, service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered != expectedTriggered {
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

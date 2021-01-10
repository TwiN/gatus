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

	if service.NumberOfFailuresInARow != 0 {
		t.Fatal("service.NumberOfFailuresInARow should've started at 0, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've started at 0, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert shouldn't start triggered")
	}

	HandleAlerting(service, &core.Result{Success: false})
	if service.NumberOfFailuresInARow != 1 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 0 to 1, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert shouldn't have triggered")
	}

	HandleAlerting(service, &core.Result{Success: false})
	if service.NumberOfFailuresInARow != 2 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 1 to 2, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if !service.Alerts[0].Triggered {
		t.Fatal("The alert should've triggered")
	}

	HandleAlerting(service, &core.Result{Success: false})
	if service.NumberOfFailuresInARow != 3 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 2 to 3, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if !service.Alerts[0].Triggered {
		t.Fatal("The alert should still show as triggered")
	}

	HandleAlerting(service, &core.Result{Success: false})
	if service.NumberOfFailuresInARow != 4 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 3 to 4, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if !service.Alerts[0].Triggered {
		t.Fatal("The alert should still show as triggered")
	}

	HandleAlerting(service, &core.Result{Success: true})
	if service.NumberOfFailuresInARow != 0 {
		t.Fatal("service.NumberOfFailuresInARow should've reset to 0, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 1 {
		t.Fatal("service.NumberOfSuccessesInARow should've increased from 0 to 1, got", service.NumberOfSuccessesInARow)
	}
	if !service.Alerts[0].Triggered {
		t.Fatal("The alert should still be triggered (because service.Alerts[0].SuccessThreshold is 3)")
	}

	HandleAlerting(service, &core.Result{Success: true})
	if service.NumberOfFailuresInARow != 0 {
		t.Fatal("service.NumberOfFailuresInARow should've stayed at 0, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 2 {
		t.Fatal("service.NumberOfSuccessesInARow should've increased from 1 to 2, got", service.NumberOfSuccessesInARow)
	}
	if !service.Alerts[0].Triggered {
		t.Fatal("The alert should still be triggered")
	}

	HandleAlerting(service, &core.Result{Success: true})
	if service.NumberOfFailuresInARow != 0 {
		t.Fatal("service.NumberOfFailuresInARow should've stayed at 0, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 3 {
		t.Fatal("service.NumberOfSuccessesInARow should've increased from 2 to 3, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert should not be triggered")
	}

	HandleAlerting(service, &core.Result{Success: true})
	if service.NumberOfFailuresInARow != 0 {
		t.Fatal("service.NumberOfFailuresInARow should've stayed at 0, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 4 {
		t.Fatal("service.NumberOfSuccessesInARow should've increased from 3 to 4, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert should no longer be triggered")
	}
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

	if service.NumberOfFailuresInARow != 0 {
		t.Fatal("service.NumberOfFailuresInARow should've started at 0, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've started at 0, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert shouldn't start triggered")
	}

	HandleAlerting(service, &core.Result{Success: false})
	if service.NumberOfFailuresInARow != 1 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 0 to 1, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert shouldn't have triggered")
	}

	HandleAlerting(service, &core.Result{Success: false})
	if service.NumberOfFailuresInARow != 2 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 1 to 2, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert shouldn't have triggered, because the provider wasn't configured properly")
	}
}

func TestHandleAlertingWithoutSendingAlertOnResolve(t *testing.T) {
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
				SendOnResolved:   false,
				Triggered:        false,
			},
		},
	}

	if service.NumberOfFailuresInARow != 0 {
		t.Fatal("service.NumberOfFailuresInARow should've started at 0, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've started at 0, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert shouldn't start triggered")
	}

	HandleAlerting(service, &core.Result{Success: false})
	if service.NumberOfFailuresInARow != 1 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 0 to 1, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert shouldn't have triggered")
	}

	HandleAlerting(service, &core.Result{Success: false})
	if service.NumberOfFailuresInARow != 2 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 1 to 2, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert shouldn't have triggered, because the provider wasn't configured properly")
	}
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
	if service.NumberOfFailuresInARow != 2 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 1 to 2, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if !service.Alerts[0].Triggered {
		t.Fatal("The alert was already triggered at the beginning of this test")
	}
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
	if service.NumberOfFailuresInARow != 1 {
		t.Fatal("service.NumberOfFailuresInARow should've increased from 0 to 1, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 0 {
		t.Fatal("service.NumberOfSuccessesInARow should've stayed at 0, got", service.NumberOfSuccessesInARow)
	}
	if !service.Alerts[0].Triggered {
		t.Fatal("The alert should've been triggered")
	}

	HandleAlerting(service, &core.Result{Success: true})
	if service.NumberOfFailuresInARow != 0 {
		t.Fatal("service.NumberOfFailuresInARow should've decreased from 1 to 0, got", service.NumberOfFailuresInARow)
	}
	if service.NumberOfSuccessesInARow != 1 {
		t.Fatal("service.NumberOfSuccessesInARow should've increased from 0 to 1, got", service.NumberOfSuccessesInARow)
	}
	if service.Alerts[0].Triggered {
		t.Fatal("The alert shouldn't be triggered anymore")
	}
}

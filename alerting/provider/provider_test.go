package provider

import (
	"testing"

	"github.com/TwinProduction/gatus/alerting/alert"
)

func TestParseWithDefaultAlert(t *testing.T) {
	type Scenario struct {
		Name                                            string
		DefaultAlert, ServiceAlert, ExpectedOutputAlert *alert.Alert
	}
	enabled := true
	disabled := false
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []Scenario{
		{
			Name: "service-alert-type-only",
			DefaultAlert: &alert.Alert{
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
			},
			ServiceAlert: &alert.Alert{
				Type: alert.TypeDiscord,
			},
			ExpectedOutputAlert: &alert.Alert{
				Type:             alert.TypeDiscord,
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
			},
		},
		{
			Name: "service-alert-overwrites-default-alert",
			DefaultAlert: &alert.Alert{
				Enabled:          &disabled,
				SendOnResolved:   &disabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
			},
			ServiceAlert: &alert.Alert{
				Type:             alert.TypeTelegram,
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &secondDescription,
				FailureThreshold: 6,
				SuccessThreshold: 11,
			},
			ExpectedOutputAlert: &alert.Alert{
				Type:             alert.TypeTelegram,
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &secondDescription,
				FailureThreshold: 6,
				SuccessThreshold: 11,
			},
		},
		{
			Name: "service-alert-partially-overwrites-default-alert",
			DefaultAlert: &alert.Alert{
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
			},
			ServiceAlert: &alert.Alert{
				Type:             alert.TypeDiscord,
				Enabled:          nil,
				SendOnResolved:   nil,
				FailureThreshold: 6,
				SuccessThreshold: 11,
			},
			ExpectedOutputAlert: &alert.Alert{
				Type:             alert.TypeDiscord,
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &firstDescription,
				FailureThreshold: 6,
				SuccessThreshold: 11,
			},
		},
		{
			Name: "default-alert-type-should-be-ignored",
			DefaultAlert: &alert.Alert{
				Type:             alert.TypeTelegram,
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
			},
			ServiceAlert: &alert.Alert{
				Type: alert.TypeDiscord,
			},
			ExpectedOutputAlert: &alert.Alert{
				Type:             alert.TypeDiscord,
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
			},
		},
		{
			Name: "no-default-alert",
			DefaultAlert: &alert.Alert{
				Type:             alert.TypeDiscord,
				Enabled:          nil,
				SendOnResolved:   nil,
				Description:      &firstDescription,
				FailureThreshold: 2,
				SuccessThreshold: 5,
			},
			ServiceAlert:        nil,
			ExpectedOutputAlert: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			ParseWithDefaultAlert(scenario.DefaultAlert, scenario.ServiceAlert)
			if scenario.ExpectedOutputAlert == nil {
				if scenario.ServiceAlert != nil {
					t.Fail()
				}
				return
			}
			if scenario.ServiceAlert.IsEnabled() != scenario.ExpectedOutputAlert.IsEnabled() {
				t.Errorf("expected ServiceAlert.IsEnabled() to be %v, got %v", scenario.ExpectedOutputAlert.IsEnabled(), scenario.ServiceAlert.IsEnabled())
			}
			if scenario.ServiceAlert.IsSendingOnResolved() != scenario.ExpectedOutputAlert.IsSendingOnResolved() {
				t.Errorf("expected ServiceAlert.IsSendingOnResolved() to be %v, got %v", scenario.ExpectedOutputAlert.IsSendingOnResolved(), scenario.ServiceAlert.IsSendingOnResolved())
			}
			if scenario.ServiceAlert.GetDescription() != scenario.ExpectedOutputAlert.GetDescription() {
				t.Errorf("expected ServiceAlert.GetDescription() to be %v, got %v", scenario.ExpectedOutputAlert.GetDescription(), scenario.ServiceAlert.GetDescription())
			}
			if scenario.ServiceAlert.FailureThreshold != scenario.ExpectedOutputAlert.FailureThreshold {
				t.Errorf("expected ServiceAlert.FailureThreshold to be %v, got %v", scenario.ExpectedOutputAlert.FailureThreshold, scenario.ServiceAlert.FailureThreshold)
			}
			if scenario.ServiceAlert.SuccessThreshold != scenario.ExpectedOutputAlert.SuccessThreshold {
				t.Errorf("expected ServiceAlert.SuccessThreshold to be %v, got %v", scenario.ExpectedOutputAlert.SuccessThreshold, scenario.ServiceAlert.SuccessThreshold)
			}
		})
	}
}

package provider

import (
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
)

func TestParseWithDefaultAlert(t *testing.T) {
	type Scenario struct {
		Name                                             string
		DefaultAlert, EndpointAlert, ExpectedOutputAlert *alert.Alert
	}
	enabled := true
	disabled := false
	firstDescription := "description-1"
	secondDescription := "description-2"
	scenarios := []Scenario{
		{
			Name: "endpoint-alert-type-only",
			DefaultAlert: &alert.Alert{
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
				MinimumReminderInterval: 30 * time.Second,
			},
			EndpointAlert: &alert.Alert{
				Type: alert.TypeDiscord,
			},
			ExpectedOutputAlert: &alert.Alert{
				Type:             alert.TypeDiscord,
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
				MinimumReminderInterval: 30 * time.Second,
			},
		},
		{
			Name: "endpoint-alert-overwrites-default-alert",
			DefaultAlert: &alert.Alert{
				Enabled:          &disabled,
				SendOnResolved:   &disabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
			},
			EndpointAlert: &alert.Alert{
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
			Name: "endpoint-alert-partially-overwrites-default-alert",
			DefaultAlert: &alert.Alert{
				Enabled:          &enabled,
				SendOnResolved:   &enabled,
				Description:      &firstDescription,
				FailureThreshold: 5,
				SuccessThreshold: 10,
			},
			EndpointAlert: &alert.Alert{
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
			EndpointAlert: &alert.Alert{
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
			EndpointAlert:       nil,
			ExpectedOutputAlert: nil,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			MergeProviderDefaultAlertIntoEndpointAlert(scenario.DefaultAlert, scenario.EndpointAlert)
			if scenario.ExpectedOutputAlert == nil {
				if scenario.EndpointAlert != nil {
					t.Fail()
				}
				return
			}
			if scenario.EndpointAlert.IsEnabled() != scenario.ExpectedOutputAlert.IsEnabled() {
				t.Errorf("expected EndpointAlert.IsEnabled() to be %v, got %v", scenario.ExpectedOutputAlert.IsEnabled(), scenario.EndpointAlert.IsEnabled())
			}
			if scenario.EndpointAlert.IsSendingOnResolved() != scenario.ExpectedOutputAlert.IsSendingOnResolved() {
				t.Errorf("expected EndpointAlert.IsSendingOnResolved() to be %v, got %v", scenario.ExpectedOutputAlert.IsSendingOnResolved(), scenario.EndpointAlert.IsSendingOnResolved())
			}
			if scenario.EndpointAlert.GetDescription() != scenario.ExpectedOutputAlert.GetDescription() {
				t.Errorf("expected EndpointAlert.GetDescription() to be %v, got %v", scenario.ExpectedOutputAlert.GetDescription(), scenario.EndpointAlert.GetDescription())
			}
			if scenario.EndpointAlert.FailureThreshold != scenario.ExpectedOutputAlert.FailureThreshold {
				t.Errorf("expected EndpointAlert.FailureThreshold to be %v, got %v", scenario.ExpectedOutputAlert.FailureThreshold, scenario.EndpointAlert.FailureThreshold)
			}
			if scenario.EndpointAlert.SuccessThreshold != scenario.ExpectedOutputAlert.SuccessThreshold {
				t.Errorf("expected EndpointAlert.SuccessThreshold to be %v, got %v", scenario.ExpectedOutputAlert.SuccessThreshold, scenario.EndpointAlert.SuccessThreshold)
			}
			if int(scenario.EndpointAlert.MinimumReminderInterval) != int(scenario.ExpectedOutputAlert.MinimumReminderInterval) {
				t.Errorf("expected EndpointAlert.MinimumReminderInterval to be %v, got %v", scenario.ExpectedOutputAlert.MinimumReminderInterval, scenario.EndpointAlert.MinimumReminderInterval)
			}
		})
	}
}

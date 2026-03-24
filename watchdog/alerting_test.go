package watchdog

import (
	"os"
	"testing"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/alerting/provider/clickup"
	"github.com/TwiN/gatus/v5/alerting/provider/custom"
	"github.com/TwiN/gatus/v5/alerting/provider/datadog"
	"github.com/TwiN/gatus/v5/alerting/provider/discord"
	"github.com/TwiN/gatus/v5/alerting/provider/email"
	"github.com/TwiN/gatus/v5/alerting/provider/ifttt"
	"github.com/TwiN/gatus/v5/alerting/provider/line"
	"github.com/TwiN/gatus/v5/alerting/provider/matrix"
	"github.com/TwiN/gatus/v5/alerting/provider/mattermost"
	"github.com/TwiN/gatus/v5/alerting/provider/messagebird"
	"github.com/TwiN/gatus/v5/alerting/provider/newrelic"
	"github.com/TwiN/gatus/v5/alerting/provider/pagerduty"
	"github.com/TwiN/gatus/v5/alerting/provider/plivo"
	"github.com/TwiN/gatus/v5/alerting/provider/pushover"
	"github.com/TwiN/gatus/v5/alerting/provider/signl4"
	"github.com/TwiN/gatus/v5/alerting/provider/slack"
	"github.com/TwiN/gatus/v5/alerting/provider/teams"
	"github.com/TwiN/gatus/v5/alerting/provider/telegram"
	"github.com/TwiN/gatus/v5/alerting/provider/twilio"
	"github.com/TwiN/gatus/v5/alerting/provider/vonage"
	"github.com/TwiN/gatus/v5/alerting/provider/zapier"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

func TestHandleAlerting(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				DefaultConfig: custom.Config{
					URL:    "https://twin.sh/health",
					Method: "GET",
				},
			},
		},
	}
	enabled := true
	ep := &endpoint.Endpoint{
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

	verify(t, ep, 0, 0, false, "The alert shouldn't start triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 1, 0, false, "The alert shouldn't have triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 2, 0, true, "The alert should've triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 3, 0, true, "The alert should still be triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 4, 0, true, "The alert should still be triggered")
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 1, true, "The alert should still be triggered (because endpoint.Alerts[0].SuccessThreshold is 3)")
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 2, true, "The alert should still be triggered (because endpoint.Alerts[0].SuccessThreshold is 3)")
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 3, false, "The alert should've been resolved")
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 4, false, "The alert should no longer be triggered")
}

func TestHandleAlertingWhenAlertingConfigIsNil(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()
	HandleAlerting(nil, nil, nil)
}

func TestHandleAlertingWithBadAlertProvider(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	enabled := true
	ep := &endpoint.Endpoint{
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

	verify(t, ep, 0, 0, false, "The alert shouldn't start triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, &alerting.Config{})
	verify(t, ep, 1, 0, false, "The alert shouldn't have triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, &alerting.Config{})
	verify(t, ep, 2, 0, false, "The alert shouldn't have triggered, because the provider wasn't configured properly")
}

func TestHandleAlertingWhenTriggeredAlertIsAlmostResolvedButendpointStartFailingAgain(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				DefaultConfig: custom.Config{
					URL:    "https://twin.sh/health",
					Method: "GET",
				},
			},
		},
	}
	enabled := true
	ep := &endpoint.Endpoint{
		URL: "https://example.com",
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
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 2, 0, true, "The alert was already triggered at the beginning of this test")
}

func TestHandleAlertingWhenTriggeredAlertIsResolvedButSendOnResolvedIsFalse(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				DefaultConfig: custom.Config{
					URL:    "https://twin.sh/health",
					Method: "GET",
				},
			},
		},
	}
	enabled := true
	disabled := false
	ep := &endpoint.Endpoint{
		URL: "https://example.com",
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

	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 1, false, "The alert should've been resolved")
}

func TestHandleAlertingWhenTriggeredAlertIsResolvedPagerDuty(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Alerting: &alerting.Config{
			PagerDuty: &pagerduty.AlertProvider{
				DefaultConfig: pagerduty.Config{
					IntegrationKey: "00000000000000000000000000000000",
				},
			},
		},
	}
	enabled := true
	ep := &endpoint.Endpoint{
		URL: "https://example.com",
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

	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 1, 0, true, "")

	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 1, false, "The alert should've been resolved")
}

func TestHandleAlertingWhenTriggeredAlertIsResolvedPushover(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Alerting: &alerting.Config{
			Pushover: &pushover.AlertProvider{
				DefaultConfig: pushover.Config{
					ApplicationToken: "000000000000000000000000000000",
					UserKey:          "000000000000000000000000000000",
				},
			},
		},
	}
	enabled := true
	ep := &endpoint.Endpoint{
		URL: "https://example.com",
		Alerts: []*alert.Alert{
			{
				Type:             alert.TypePushover,
				Enabled:          &enabled,
				FailureThreshold: 1,
				SuccessThreshold: 1,
				SendOnResolved:   &enabled,
				Triggered:        false,
			},
		},
		NumberOfFailuresInARow: 0,
	}

	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 1, 0, true, "")

	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 1, false, "The alert should've been resolved")
}

func TestHandleAlertingWithProviderThatReturnsAnError(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()
	enabled := true
	scenarios := []struct {
		Name           string
		AlertingConfig *alerting.Config
		AlertType      alert.Type
	}{
		{
			Name:      "custom",
			AlertType: alert.TypeCustom,
			AlertingConfig: &alerting.Config{
				Custom: &custom.AlertProvider{
					DefaultConfig: custom.Config{
						URL:    "https://twin.sh/health",
						Method: "GET",
					},
				},
			},
		},
		{
			Name:      "datadog",
			AlertType: alert.TypeDatadog,
			AlertingConfig: &alerting.Config{
				Datadog: &datadog.AlertProvider{
					DefaultConfig: datadog.Config{
						APIKey: "test-key",
					},
				},
			},
		},
		{
			Name:      "discord",
			AlertType: alert.TypeDiscord,
			AlertingConfig: &alerting.Config{
				Discord: &discord.AlertProvider{
					DefaultConfig: discord.Config{
						WebhookURL: "https://example.com",
					},
				},
			},
		},
		{
			Name:      "email",
			AlertType: alert.TypeEmail,
			AlertingConfig: &alerting.Config{
				Email: &email.AlertProvider{
					DefaultConfig: email.Config{
						From:     "from@example.com",
						Password: "hunter2",
						Host:     "mail.example.com",
						Port:     587,
						To:       "to@example.com",
					},
				},
			},
		},
		{
			Name:      "ifttt",
			AlertType: alert.TypeIFTTT,
			AlertingConfig: &alerting.Config{
				IFTTT: &ifttt.AlertProvider{
					DefaultConfig: ifttt.Config{
						WebhookKey: "test-key",
						EventName:  "test-event",
					},
				},
			},
		},
		{
			Name:      "line",
			AlertType: alert.TypeLine,
			AlertingConfig: &alerting.Config{
				Line: &line.AlertProvider{
					DefaultConfig: line.Config{
						ChannelAccessToken: "test-token",
						UserIDs:            []string{"test-user"},
					},
				},
			},
		},
		{
			Name:      "mattermost",
			AlertType: alert.TypeMattermost,
			AlertingConfig: &alerting.Config{
				Mattermost: &mattermost.AlertProvider{
					DefaultConfig: mattermost.Config{
						WebhookURL: "https://example.com",
					},
				},
			},
		},
		{
			Name:      "messagebird",
			AlertType: alert.TypeMessagebird,
			AlertingConfig: &alerting.Config{
				Messagebird: &messagebird.AlertProvider{
					DefaultConfig: messagebird.Config{
						AccessKey:  "1",
						Originator: "2",
						Recipients: "3",
					},
				},
			},
		},
		{
			Name:      "newrelic",
			AlertType: alert.TypeNewRelic,
			AlertingConfig: &alerting.Config{
				NewRelic: &newrelic.AlertProvider{
					DefaultConfig: newrelic.Config{
						InsertKey: "test-key",
						AccountID: "test-account",
					},
				},
			},
		},
		{
			Name:      "pagerduty",
			AlertType: alert.TypePagerDuty,
			AlertingConfig: &alerting.Config{
				PagerDuty: &pagerduty.AlertProvider{
					DefaultConfig: pagerduty.Config{
						IntegrationKey: "00000000000000000000000000000000",
					},
				},
			},
		},
		{
			Name:      "plivo",
			AlertType: alert.TypePlivo,
			AlertingConfig: &alerting.Config{
				Plivo: &plivo.AlertProvider{
					DefaultConfig: plivo.Config{
						AuthID:    "test-id",
						AuthToken: "test-token",
						From:      "test-from",
						To:        []string{"test-to"},
					},
				},
			},
		},
		{
			Name:      "pushover",
			AlertType: alert.TypePushover,
			AlertingConfig: &alerting.Config{
				Pushover: &pushover.AlertProvider{
					DefaultConfig: pushover.Config{
						ApplicationToken: "000000000000000000000000000000",
						UserKey:          "000000000000000000000000000000",
					},
				},
			},
		},
		{
			Name:      "signl4",
			AlertType: alert.TypeSIGNL4,
			AlertingConfig: &alerting.Config{
				SIGNL4: &signl4.AlertProvider{
					DefaultConfig: signl4.Config{
						TeamSecret: "test-secret",
					},
				},
			},
		},
		{
			Name:      "slack",
			AlertType: alert.TypeSlack,
			AlertingConfig: &alerting.Config{
				Slack: &slack.AlertProvider{
					DefaultConfig: slack.Config{
						WebhookURL: "https://example.com",
					},
				},
			},
		},
		{
			Name:      "teams",
			AlertType: alert.TypeTeams,
			AlertingConfig: &alerting.Config{
				Teams: &teams.AlertProvider{
					DefaultConfig: teams.Config{
						WebhookURL: "https://example.com",
					},
				},
			},
		},
		{
			Name:      "telegram",
			AlertType: alert.TypeTelegram,
			AlertingConfig: &alerting.Config{
				Telegram: &telegram.AlertProvider{
					DefaultConfig: telegram.Config{
						Token: "1",
						ID:    "2",
					},
				},
			},
		},
		{
			Name:      "twilio",
			AlertType: alert.TypeTwilio,
			AlertingConfig: &alerting.Config{
				Twilio: &twilio.AlertProvider{
					DefaultConfig: twilio.Config{
						SID:   "1",
						Token: "2",
						From:  "3",
						To:    "4",
					},
				},
			},
		},
		{
			Name:      "vonage",
			AlertType: alert.TypeVonage,
			AlertingConfig: &alerting.Config{
				Vonage: &vonage.AlertProvider{
					DefaultConfig: vonage.Config{
						APIKey:    "test-key",
						APISecret: "test-secret",
						From:      "test-from",
						To:        []string{"test-to"},
					},
				},
			},
		},
		{
			Name:      "zapier",
			AlertType: alert.TypeZapier,
			AlertingConfig: &alerting.Config{
				Zapier: &zapier.AlertProvider{
					DefaultConfig: zapier.Config{
						WebhookURL: "https://example.com",
					},
				},
			},
		},
		{
			Name:      "matrix",
			AlertType: alert.TypeMatrix,
			AlertingConfig: &alerting.Config{
				Matrix: &matrix.AlertProvider{
					DefaultConfig: matrix.Config{
						ServerURL:      "https://example.com",
						AccessToken:    "1",
						InternalRoomID: "!a:example.com",
					},
				},
			},
		},
		{
			Name:      "clickup",
			AlertType: alert.TypeClickUp,
			AlertingConfig: &alerting.Config{
				ClickUp: &clickup.AlertProvider{
					DefaultConfig: clickup.Config{
						ListID: "test-list-id",
						Token:  "test-token",
					},
				},
			},
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			ep := &endpoint.Endpoint{
				URL: "https://example.com",
				Alerts: []*alert.Alert{
					{
						Type:             scenario.AlertType,
						Enabled:          &enabled,
						FailureThreshold: 2,
						SuccessThreshold: 2,
						SendOnResolved:   &enabled,
						Triggered:        false,
					},
				},
			}
			_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
			HandleAlerting(ep, &endpoint.Result{Success: false}, scenario.AlertingConfig)
			verify(t, ep, 1, 0, false, "")
			HandleAlerting(ep, &endpoint.Result{Success: false}, scenario.AlertingConfig)
			verify(t, ep, 2, 0, false, "The alert should have failed to trigger, because the alert provider is returning an error")
			HandleAlerting(ep, &endpoint.Result{Success: false}, scenario.AlertingConfig)
			verify(t, ep, 3, 0, false, "The alert should still not be triggered, because the alert provider is still returning an error")
			HandleAlerting(ep, &endpoint.Result{Success: false}, scenario.AlertingConfig)
			verify(t, ep, 4, 0, false, "The alert should still not be triggered, because the alert provider is still returning an error")
			_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")
			HandleAlerting(ep, &endpoint.Result{Success: false}, scenario.AlertingConfig)
			verify(t, ep, 5, 0, true, "The alert should've been triggered because the alert provider is no longer returning an error")
			HandleAlerting(ep, &endpoint.Result{Success: true}, scenario.AlertingConfig)
			verify(t, ep, 0, 1, true, "The alert should've still been triggered")
			_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
			HandleAlerting(ep, &endpoint.Result{Success: true}, scenario.AlertingConfig)
			verify(t, ep, 0, 2, false, "The alert should've been resolved DESPITE THE ALERT PROVIDER RETURNING AN ERROR. See Alert.Triggered for further explanation.")
			_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")

			// Make sure that everything's working as expected after a rough patch
			HandleAlerting(ep, &endpoint.Result{Success: false}, scenario.AlertingConfig)
			verify(t, ep, 1, 0, false, "")
			HandleAlerting(ep, &endpoint.Result{Success: false}, scenario.AlertingConfig)
			verify(t, ep, 2, 0, true, "The alert should have triggered")
			HandleAlerting(ep, &endpoint.Result{Success: true}, scenario.AlertingConfig)
			verify(t, ep, 0, 1, true, "The alert should still be triggered")
			HandleAlerting(ep, &endpoint.Result{Success: true}, scenario.AlertingConfig)
			verify(t, ep, 0, 2, false, "The alert should have been resolved")
		})
	}

}

func TestHandleAlertingWithProviderThatOnlyReturnsErrorOnResolve(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				DefaultConfig: custom.Config{
					URL:    "https://twin.sh/health",
					Method: "GET",
				},
			},
		},
	}
	enabled := true
	ep := &endpoint.Endpoint{
		URL: "https://example.com",
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

	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 1, 0, true, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 1, false, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 1, 0, true, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "true")
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 1, false, "")
	_ = os.Setenv("MOCK_ALERT_PROVIDER_ERROR", "false")

	// Make sure that everything's working as expected after a rough patch
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 1, 0, true, "")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 2, 0, true, "")
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 1, false, "")
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
	verify(t, ep, 0, 2, false, "")
}

func TestHandleAlertingWithMinimumReminderInterval(t *testing.T) {
	_ = os.Setenv("MOCK_ALERT_PROVIDER", "true")
	defer os.Clearenv()

	cfg := &config.Config{
		Alerting: &alerting.Config{
			Custom: &custom.AlertProvider{
				DefaultConfig: custom.Config{
					URL:    "https://twin.sh/health",
					Method: "GET",
				},
			},
		},
	}
	enabled := true
	ep := &endpoint.Endpoint{
		URL: "https://example.com",
		Alerts: []*alert.Alert{
			{
				Type:                    alert.TypeCustom,
				Enabled:                 &enabled,
				FailureThreshold:        2,
				SuccessThreshold:        3,
				SendOnResolved:          &enabled,
				Triggered:               false,
				MinimumReminderInterval: 5 * time.Minute,
			},
		},
	}

	verify(t, ep, 0, 0, false, "The alert shouldn't start triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 1, 0, false, "The alert shouldn't have triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 2, 0, true, "The alert should've triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 3, 0, true, "The alert should still be triggered")
	HandleAlerting(ep, &endpoint.Result{Success: false}, cfg.Alerting)
	verify(t, ep, 4, 0, true, "The alert should still be triggered")
	HandleAlerting(ep, &endpoint.Result{Success: true}, cfg.Alerting)
}

func verify(t *testing.T, ep *endpoint.Endpoint, expectedNumberOfFailuresInARow, expectedNumberOfSuccessInARow int, expectedTriggered bool, expectedTriggeredReason string) {
	if ep.NumberOfFailuresInARow != expectedNumberOfFailuresInARow {
		t.Errorf("endpoint.NumberOfFailuresInARow should've been %d, got %d", expectedNumberOfFailuresInARow, ep.NumberOfFailuresInARow)
	}
	if ep.NumberOfSuccessesInARow != expectedNumberOfSuccessInARow {
		t.Errorf("endpoint.NumberOfSuccessesInARow should've been %d, got %d", expectedNumberOfSuccessInARow, ep.NumberOfSuccessesInARow)
	}
	if ep.Alerts[0].Triggered != expectedTriggered {
		if len(expectedTriggeredReason) != 0 {
			t.Error(expectedTriggeredReason)
		} else {
			if expectedTriggered {
				t.Error("The alert should've been triggered")
			} else {
				t.Error("The alert shouldn't have been triggered")
			}
		}
	}
}

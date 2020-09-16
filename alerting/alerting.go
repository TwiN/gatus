package alerting

import (
	"encoding/json"
	"fmt"
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"log"
)

// Handle takes care of alerts to resolve and alerts to trigger based on result success or failure
func Handle(service *core.Service, result *core.Result) {
	cfg := config.Get()
	if cfg.Alerting == nil {
		return
	}
	if result.Success {
		handleAlertsToResolve(service, result, cfg)
	} else {
		handleAlertsToTrigger(service, result, cfg)
	}
}

func handleAlertsToTrigger(service *core.Service, result *core.Result, cfg *config.Config) {
	service.NumberOfSuccessesInARow = 0
	service.NumberOfFailuresInARow++
	for _, alert := range service.Alerts {
		// If the alert hasn't been triggered, move to the next one
		if !alert.Enabled || alert.Threshold != service.NumberOfFailuresInARow {
			continue
		}
		if alert.Triggered {
			if cfg.Debug {
				log.Printf("[alerting][handleAlertsToTrigger] Alert with description='%s' has already been triggered, skipping", alert.Description)
			}
			continue
		}
		var alertProvider *core.CustomAlertProvider
		if alert.Type == core.SlackAlert {
			if len(cfg.Alerting.Slack) > 0 {
				log.Printf("[alerting][handleAlertsToTrigger] Sending Slack alert because alert with description='%s' has been triggered", alert.Description)
				alertProvider = core.CreateSlackCustomAlertProvider(cfg.Alerting.Slack, service, alert, result, false)
			} else {
				log.Printf("[alerting][handleAlertsToTrigger] Not sending Slack alert despite being triggered, because there is no Slack webhook configured")
			}
		} else if alert.Type == core.PagerDutyAlert {
			if len(cfg.Alerting.PagerDuty) > 0 {
				log.Printf("[alerting][handleAlertsToTrigger] Sending PagerDuty alert because alert with description='%s' has been triggered", alert.Description)
				alertProvider = core.CreatePagerDutyCustomAlertProvider(cfg.Alerting.PagerDuty, "trigger", "", service, fmt.Sprintf("TRIGGERED: %s - %s", service.Name, alert.Description))
			} else {
				log.Printf("[alerting][handleAlertsToTrigger] Not sending PagerDuty alert despite being triggered, because PagerDuty isn't configured properly")
			}
		} else if alert.Type == core.TwilioAlert {
			if cfg.Alerting.Twilio != nil && cfg.Alerting.Twilio.IsValid() {
				log.Printf("[alerting][handleAlertsToTrigger] Sending Twilio alert because alert with description='%s' has been triggered", alert.Description)
				alertProvider = core.CreateTwilioCustomAlertProvider(cfg.Alerting.Twilio, fmt.Sprintf("TRIGGERED: %s - %s", service.Name, alert.Description))
			} else {
				log.Printf("[alerting][handleAlertsToTrigger] Not sending Twilio alert despite being triggered, because Twilio config settings missing")
			}
		} else if alert.Type == core.CustomAlert {
			if cfg.Alerting.Custom != nil && cfg.Alerting.Custom.IsValid() {
				log.Printf("[alerting][handleAlertsToTrigger] Sending custom alert because alert with description='%s' has been triggered", alert.Description)
				alertProvider = &core.CustomAlertProvider{
					Url:     cfg.Alerting.Custom.Url,
					Method:  cfg.Alerting.Custom.Method,
					Body:    cfg.Alerting.Custom.Body,
					Headers: cfg.Alerting.Custom.Headers,
				}
			} else {
				log.Printf("[alerting][handleAlertsToTrigger] Not sending custom alert despite being triggered, because there is no custom url configured")
			}
		}
		if alertProvider != nil {
			// TODO: retry on error
			var err error
			if alert.Type == core.PagerDutyAlert {
				var body []byte
				body, err = alertProvider.Send(service.Name, alert.Description, true)
				if err == nil {
					var response pagerDutyResponse
					err = json.Unmarshal(body, &response)
					if err != nil {
						log.Printf("[alerting][handleAlertsToTrigger] Ran into error unmarshaling pager duty response: %s", err.Error())
					} else {
						alert.ResolveKey = response.DedupKey
					}
				}
			} else {
				_, err = alertProvider.Send(service.Name, alert.Description, false)
			}
			if err != nil {
				log.Printf("[alerting][handleAlertsToTrigger] Ran into error sending an alert: %s", err.Error())
			} else {
				alert.Triggered = true
			}
		}
	}
}

func handleAlertsToResolve(service *core.Service, result *core.Result, cfg *config.Config) {
	service.NumberOfSuccessesInARow++
	for _, alert := range service.Alerts {
		if !alert.Enabled || !alert.Triggered || alert.SuccessBeforeResolved > service.NumberOfSuccessesInARow {
			continue
		}
		alert.Triggered = false
		if !alert.SendOnResolved {
			continue
		}
		var alertProvider *core.CustomAlertProvider
		if alert.Type == core.SlackAlert {
			if len(cfg.Alerting.Slack) > 0 {
				log.Printf("[alerting][handleAlertsToResolve] Sending Slack alert because alert with description='%s' has been resolved", alert.Description)
				alertProvider = core.CreateSlackCustomAlertProvider(cfg.Alerting.Slack, service, alert, result, true)
			} else {
				log.Printf("[alerting][handleAlertsToResolve] Not sending Slack alert despite being resolved, because there is no Slack webhook configured")
			}
		} else if alert.Type == core.PagerDutyAlert {
			if len(cfg.Alerting.PagerDuty) > 0 {
				log.Printf("[alerting][handleAlertsToResolve] Sending PagerDuty alert because alert with description='%s' has been resolved", alert.Description)
				alertProvider = core.CreatePagerDutyCustomAlertProvider(cfg.Alerting.PagerDuty, "resolve", alert.ResolveKey, service, fmt.Sprintf("RESOLVED: %s - %s", service.Name, alert.Description))
			} else {
				log.Printf("[alerting][handleAlertsToResolve] Not sending PagerDuty alert despite being resolved, because PagerDuty isn't configured properly")
			}
		} else if alert.Type == core.TwilioAlert {
			if cfg.Alerting.Twilio != nil && cfg.Alerting.Twilio.IsValid() {
				log.Printf("[alerting][handleAlertsToResolve] Sending Twilio alert because alert with description='%s' has been resolved", alert.Description)
				alertProvider = core.CreateTwilioCustomAlertProvider(cfg.Alerting.Twilio, fmt.Sprintf("RESOLVED: %s - %s", service.Name, alert.Description))
			} else {
				log.Printf("[alerting][handleAlertsToResolve] Not sending Twilio alert despite being resolved, because Twilio isn't configured properly")
			}
		} else if alert.Type == core.CustomAlert {
			if cfg.Alerting.Custom != nil && cfg.Alerting.Custom.IsValid() {
				log.Printf("[alerting][handleAlertsToResolve] Sending custom alert because alert with description='%s' has been resolved", alert.Description)
				alertProvider = &core.CustomAlertProvider{
					Url:     cfg.Alerting.Custom.Url,
					Method:  cfg.Alerting.Custom.Method,
					Body:    cfg.Alerting.Custom.Body,
					Headers: cfg.Alerting.Custom.Headers,
				}
			} else {
				log.Printf("[alerting][handleAlertsToResolve] Not sending custom alert despite being resolved, because the custom provider isn't configured properly")
			}
		}
		if alertProvider != nil {
			// TODO: retry on error
			_, err := alertProvider.Send(service.Name, alert.Description, true)
			if err != nil {
				log.Printf("[alerting][handleAlertsToResolve] Ran into error sending an alert: %s", err.Error())
			} else {
				if alert.Type == core.PagerDutyAlert {
					alert.ResolveKey = ""
				}
			}
		}
	}
	service.NumberOfFailuresInARow = 0
}

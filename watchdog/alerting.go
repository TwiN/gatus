package watchdog

import (
	"encoding/json"
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"log"
)

// HandleAlerting takes care of alerts to resolve and alerts to trigger based on result success or failure
func HandleAlerting(service *core.Service, result *core.Result) {
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
		if !alert.Enabled || alert.FailureThreshold != service.NumberOfFailuresInARow {
			continue
		}
		if alert.Triggered {
			if cfg.Debug {
				log.Printf("[watchdog][handleAlertsToTrigger] Alert with description='%s' has already been triggered, skipping", alert.Description)
			}
			continue
		}
		alertProvider := config.GetAlertingProviderByAlertType(cfg, alert.Type)
		if alertProvider != nil && alertProvider.IsValid() {
			log.Printf("[watchdog][handleAlertsToTrigger] Sending %s alert because alert with description='%s' has been triggered", alert.Type, alert.Description)
			customAlertProvider := alertProvider.ToCustomAlertProvider(service, alert, result, false)
			// TODO: retry on error
			var err error
			// We need to extract the DedupKey from PagerDuty's response
			if alert.Type == core.PagerDutyAlert {
				var body []byte
				body, err = customAlertProvider.Send(service.Name, alert.Description, false)
				if err == nil {
					var response pagerDutyResponse
					err = json.Unmarshal(body, &response)
					if err != nil {
						log.Printf("[watchdog][handleAlertsToTrigger] Ran into error unmarshaling pager duty response: %s", err.Error())
					} else {
						alert.ResolveKey = response.DedupKey
					}
				}
			} else {
				// All other alert types don't need to extract anything from the body, so we can just send the request right away
				_, err = customAlertProvider.Send(service.Name, alert.Description, false)
			}
			if err != nil {
				log.Printf("[watchdog][handleAlertsToTrigger] Ran into error sending an alert: %s", err.Error())
			} else {
				alert.Triggered = true
			}

		} else {
			log.Printf("[watchdog][handleAlertsToResolve] Not sending alert of type=%s despite being triggered, because the provider wasn't configured properly", alert.Type)
		}
	}
}

func handleAlertsToResolve(service *core.Service, result *core.Result, cfg *config.Config) {
	service.NumberOfSuccessesInARow++
	for _, alert := range service.Alerts {
		if !alert.Enabled || !alert.Triggered || alert.SuccessThreshold > service.NumberOfSuccessesInARow {
			continue
		}
		alert.Triggered = false
		if !alert.SendOnResolved {
			continue
		}
		alertProvider := config.GetAlertingProviderByAlertType(cfg, alert.Type)
		if alertProvider != nil && alertProvider.IsValid() {
			log.Printf("[watchdog][handleAlertsToResolve] Sending %s alert because alert with description='%s' has been resolved", alert.Type, alert.Description)
			customAlertProvider := alertProvider.ToCustomAlertProvider(service, alert, result, true)
			// TODO: retry on error
			_, err := customAlertProvider.Send(service.Name, alert.Description, true)
			if err != nil {
				log.Printf("[watchdog][handleAlertsToResolve] Ran into error sending an alert: %s", err.Error())
			} else {
				if alert.Type == core.PagerDutyAlert {
					alert.ResolveKey = ""
				}
			}
		} else {
			log.Printf("[watchdog][handleAlertsToResolve] Not sending alert of type=%s despite being resolved, because the provider wasn't configured properly", alert.Type)
		}
	}
	service.NumberOfFailuresInARow = 0
}

type pagerDutyResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	DedupKey string `json:"dedup_key"`
}

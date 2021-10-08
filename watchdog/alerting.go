package watchdog

import (
	"encoding/json"
	"log"

	"github.com/TwiN/gatus/v3/alerting"
	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/core"
)

// HandleAlerting takes care of alerts to resolve and alerts to trigger based on result success or failure
func HandleAlerting(service *core.Service, result *core.Result, alertingConfig *alerting.Config, debug bool) {
	if alertingConfig == nil {
		return
	}
	if result.Success {
		handleAlertsToResolve(service, result, alertingConfig, debug)
	} else {
		handleAlertsToTrigger(service, result, alertingConfig, debug)
	}
}

func handleAlertsToTrigger(service *core.Service, result *core.Result, alertingConfig *alerting.Config, debug bool) {
	service.NumberOfSuccessesInARow = 0
	service.NumberOfFailuresInARow++
	for _, serviceAlert := range service.Alerts {
		// If the serviceAlert hasn't been triggered, move to the next one
		if !serviceAlert.IsEnabled() || serviceAlert.FailureThreshold > service.NumberOfFailuresInARow {
			continue
		}
		if serviceAlert.Triggered {
			if debug {
				log.Printf("[watchdog][handleAlertsToTrigger] Alert for service=%s with description='%s' has already been TRIGGERED, skipping", service.Name, serviceAlert.GetDescription())
			}
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(serviceAlert.Type)
		if alertProvider != nil && alertProvider.IsValid() {
			log.Printf("[watchdog][handleAlertsToTrigger] Sending %s serviceAlert because serviceAlert for service=%s with description='%s' has been TRIGGERED", serviceAlert.Type, service.Name, serviceAlert.GetDescription())
			customAlertProvider := alertProvider.ToCustomAlertProvider(service, serviceAlert, result, false)
			// TODO: retry on error
			var err error
			// We need to extract the DedupKey from PagerDuty's response
			if serviceAlert.Type == alert.TypePagerDuty {
				var body []byte
				if body, err = customAlertProvider.Send(service.Name, serviceAlert.GetDescription(), false); err == nil {
					var response pagerDutyResponse
					if err = json.Unmarshal(body, &response); err != nil {
						log.Printf("[watchdog][handleAlertsToTrigger] Ran into error unmarshaling pagerduty response: %s", err.Error())
					} else {
						serviceAlert.ResolveKey = response.DedupKey
					}
				}
			} else {
				// All other serviceAlert types don't need to extract anything from the body, so we can just send the request right away
				_, err = customAlertProvider.Send(service.Name, serviceAlert.GetDescription(), false)
			}
			if err != nil {
				log.Printf("[watchdog][handleAlertsToTrigger] Failed to send an serviceAlert for service=%s: %s", service.Name, err.Error())
			} else {
				serviceAlert.Triggered = true
			}
		} else {
			log.Printf("[watchdog][handleAlertsToResolve] Not sending serviceAlert of type=%s despite being TRIGGERED, because the provider wasn't configured properly", serviceAlert.Type)
		}
	}
}

func handleAlertsToResolve(service *core.Service, result *core.Result, alertingConfig *alerting.Config, debug bool) {
	service.NumberOfSuccessesInARow++
	for _, serviceAlert := range service.Alerts {
		if !serviceAlert.IsEnabled() || !serviceAlert.Triggered || serviceAlert.SuccessThreshold > service.NumberOfSuccessesInARow {
			continue
		}
		// Even if the serviceAlert provider returns an error, we still set the serviceAlert's Triggered variable to false.
		// Further explanation can be found on Alert's Triggered field.
		serviceAlert.Triggered = false
		if !serviceAlert.IsSendingOnResolved() {
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(serviceAlert.Type)
		if alertProvider != nil && alertProvider.IsValid() {
			log.Printf("[watchdog][handleAlertsToResolve] Sending %s serviceAlert because serviceAlert for service=%s with description='%s' has been RESOLVED", serviceAlert.Type, service.Name, serviceAlert.GetDescription())
			customAlertProvider := alertProvider.ToCustomAlertProvider(service, serviceAlert, result, true)
			// TODO: retry on error
			_, err := customAlertProvider.Send(service.Name, serviceAlert.GetDescription(), true)
			if err != nil {
				log.Printf("[watchdog][handleAlertsToResolve] Failed to send an serviceAlert for service=%s: %s", service.Name, err.Error())
			} else {
				if serviceAlert.Type == alert.TypePagerDuty {
					serviceAlert.ResolveKey = ""
				}
			}
		} else {
			log.Printf("[watchdog][handleAlertsToResolve] Not sending serviceAlert of type=%s despite being RESOLVED, because the provider wasn't configured properly", serviceAlert.Type)
		}
	}
	service.NumberOfFailuresInARow = 0
}

type pagerDutyResponse struct {
	Status   string `json:"status"`
	Message  string `json:"message"`
	DedupKey string `json:"dedup_key"`
}

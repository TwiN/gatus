package watchdog

import (
	"errors"
	"log"
	"os"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/endpoint/result"
)

// HandleAlerting takes care of alerts to resolve and alerts to trigger based on result success or failure
func HandleAlerting(ep *endpoint.Endpoint, result *result.Result, alertingConfig *alerting.Config, debug bool) {
	if alertingConfig == nil {
		return
	}
	if result.Success {
		handleAlertsToResolve(ep, result, alertingConfig, debug)
	} else {
		handleAlertsToTrigger(ep, result, alertingConfig, debug)
	}
}

func handleAlertsToTrigger(ep *endpoint.Endpoint, result *result.Result, alertingConfig *alerting.Config, debug bool) {
	ep.NumberOfSuccessesInARow = 0
	ep.NumberOfFailuresInARow++
	for _, endpointAlert := range ep.Alerts {
		// If the alert hasn't been triggered, move to the next one
		if !endpointAlert.IsEnabled() || endpointAlert.FailureThreshold > ep.NumberOfFailuresInARow {
			continue
		}
		if endpointAlert.Triggered {
			if debug {
				log.Printf("[watchdog.handleAlertsToTrigger] Alert for endpoint=%s with description='%s' has already been TRIGGERED, skipping", ep.Name, endpointAlert.GetDescription())
			}
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			log.Printf("[watchdog.handleAlertsToTrigger] Sending %s alert because alert for endpoint=%s with description='%s' has been TRIGGERED", endpointAlert.Type, ep.Name, endpointAlert.GetDescription())
			var err error
			if os.Getenv("MOCK_ALERT_PROVIDER") == "true" {
				if os.Getenv("MOCK_ALERT_PROVIDER_ERROR") == "true" {
					err = errors.New("error")
				}
			} else {
				err = alertProvider.Send(ep, endpointAlert, result, false)
			}
			if err != nil {
				log.Printf("[watchdog.handleAlertsToTrigger] Failed to send an alert for endpoint=%s: %s", ep.Name, err.Error())
			} else {
				endpointAlert.Triggered = true
			}
		} else {
			log.Printf("[watchdog.handleAlertsToResolve] Not sending alert of type=%s despite being TRIGGERED, because the provider wasn't configured properly", endpointAlert.Type)
		}
	}
}

func handleAlertsToResolve(ep *endpoint.Endpoint, result *result.Result, alertingConfig *alerting.Config, debug bool) {
	ep.NumberOfSuccessesInARow++
	for _, endpointAlert := range ep.Alerts {
		if !endpointAlert.IsEnabled() || !endpointAlert.Triggered || endpointAlert.SuccessThreshold > ep.NumberOfSuccessesInARow {
			continue
		}
		// Even if the alert provider returns an error, we still set the alert's Triggered variable to false.
		// Further explanation can be found on Alert's Triggered field.
		endpointAlert.Triggered = false
		if !endpointAlert.IsSendingOnResolved() {
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			log.Printf("[watchdog.handleAlertsToResolve] Sending %s alert because alert for endpoint=%s with description='%s' has been RESOLVED", endpointAlert.Type, ep.Name, endpointAlert.GetDescription())
			err := alertProvider.Send(ep, endpointAlert, result, true)
			if err != nil {
				log.Printf("[watchdog.handleAlertsToResolve] Failed to send an alert for endpoint=%s: %s", ep.Name, err.Error())
			}
		} else {
			log.Printf("[watchdog.handleAlertsToResolve] Not sending alert of type=%s despite being RESOLVED, because the provider wasn't configured properly", endpointAlert.Type)
		}
	}
	ep.NumberOfFailuresInARow = 0
}

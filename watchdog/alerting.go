package watchdog

import (
	"errors"
	"log"
	"os"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/core"
)

// HandleAlerting takes care of alerts to resolve and alerts to trigger based on result success or failure
func HandleAlerting(endpoint *core.Endpoint, result *core.Result, alertingConfig *alerting.Config, debug bool) {
	if alertingConfig == nil {
		return
	}
	if result.Success {
		handleAlertsToResolve(endpoint, result, alertingConfig, debug)
	} else {
		handleAlertsToTrigger(endpoint, result, alertingConfig, debug)
	}
}

func handleAlertsToTrigger(endpoint *core.Endpoint, result *core.Result, alertingConfig *alerting.Config, debug bool) {
	endpoint.NumberOfSuccessesInARow = 0
	endpoint.NumberOfFailuresInARow++
	for _, endpointAlert := range endpoint.Alerts {
		// If the alert hasn't been triggered, move to the next one
		if !endpointAlert.IsEnabled() {
			continue
		}
		shouldSendAlert := false
		// Check if endpointAlert is set to trigger alerts consecutively
		if endpointAlert.Consecutive {
			// If NumberOfFailuresInARow is a multiple of FailureThreshold, then we should send an alert
			if endpoint.NumberOfFailuresInARow%endpointAlert.FailureThreshold == 0 {
				shouldSendAlert = true
				log.Printf("[watchdog][handleAlertsToTrigger] Consecutive alerting enabled. Alert triggered for endpoint=%s as it meets the failure threshold multiple", endpoint.Name)
			}
		} else if !endpointAlert.Consecutive && (endpointAlert.FailureThreshold == endpoint.NumberOfFailuresInARow || endpointAlert.FailureThreshold < endpoint.NumberOfFailuresInARow) && !endpointAlert.Triggered {
			// For non-consecutive alerts, send the alert once when FailureThreshold equals NumberOfFailuresInARow
			shouldSendAlert = true
		}
		// If we shouldn't send an alert based on the above conditions, move to the next iteration
		if !shouldSendAlert {
			if debug {
				log.Printf("[watchdog][handleAlertsToTrigger] Alert for endpoint=%s with description='%s' should not be triggered, skipping", endpoint.Name, endpointAlert.GetDescription())
			}
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			log.Printf("[watchdog][handleAlertsToTrigger] Sending %s alert because alert for endpoint=%s with description='%s' has been TRIGGERED", endpointAlert.Type, endpoint.Name, endpointAlert.GetDescription())
			var err error
			if os.Getenv("MOCK_ALERT_PROVIDER") == "true" {
				if os.Getenv("MOCK_ALERT_PROVIDER_ERROR") == "true" {
					err = errors.New("error")
				}
			} else {
				err = alertProvider.Send(endpoint, endpointAlert, result, false)
			}
			if err != nil {
				log.Printf("[watchdog][handleAlertsToTrigger] Failed to send an alert for endpoint=%s: %s", endpoint.Name, err.Error())
			} else {
				endpointAlert.Triggered = true
			}
		} else {
			log.Printf("[watchdog][handleAlertsToResolve] Not sending alert of type=%s despite being TRIGGERED, because the provider wasn't configured properly", endpointAlert.Type)
		}
	}
}

func handleAlertsToResolve(endpoint *core.Endpoint, result *core.Result, alertingConfig *alerting.Config, debug bool) {
	endpoint.NumberOfSuccessesInARow++
	for _, endpointAlert := range endpoint.Alerts {
		if !endpointAlert.IsEnabled() || !endpointAlert.Triggered || endpointAlert.SuccessThreshold > endpoint.NumberOfSuccessesInARow {
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
			log.Printf("[watchdog][handleAlertsToResolve] Sending %s alert because alert for endpoint=%s with description='%s' has been RESOLVED", endpointAlert.Type, endpoint.Name, endpointAlert.GetDescription())
			err := alertProvider.Send(endpoint, endpointAlert, result, true)
			if err != nil {
				log.Printf("[watchdog][handleAlertsToResolve] Failed to send an alert for endpoint=%s: %s", endpoint.Name, err.Error())
			}
		} else {
			log.Printf("[watchdog][handleAlertsToResolve] Not sending alert of type=%s despite being RESOLVED, because the provider wasn't configured properly", endpointAlert.Type)
		}
	}
	endpoint.NumberOfFailuresInARow = 0
}

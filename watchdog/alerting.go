package watchdog

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage/store"
)

// HandleAlerting takes care of alerts to resolve and alerts to trigger based on result success or failure
func HandleAlerting(ep *endpoint.Endpoint, result *endpoint.Result, alertingConfig *alerting.Config, debug bool) {
	if alertingConfig == nil {
		return
	}
	if result.Success {
		handleAlertsToResolve(ep, result, alertingConfig, debug)
	} else {
		handleAlertsToTrigger(ep, result, alertingConfig, debug)
	}
}

func handleAlertsToTrigger(ep *endpoint.Endpoint, result *endpoint.Result, alertingConfig *alerting.Config, debug bool) {
	ep.NumberOfSuccessesInARow = 0
	ep.NumberOfFailuresInARow++
	for _, endpointAlert := range ep.Alerts {
		// If the alert hasn't been triggered, move to the next one
		if !endpointAlert.IsEnabled() || endpointAlert.FailureThreshold > ep.NumberOfFailuresInARow {
			continue
		}
		// Determine if an initial alert should be sent
		sendInitialAlert := !endpointAlert.Triggered
		// Determine if a reminder should be sent
		var lastReminder time.Duration
		if lr, ok := ep.LastReminderSent[endpointAlert.Type]; ok && !lr.IsZero() {
			lastReminder = time.Since(lr)
		}
		sendReminder := endpointAlert.Triggered && endpointAlert.RepeatInterval != nil &&
			*endpointAlert.RepeatInterval > 0 && (lastReminder == 0 || lastReminder >= *endpointAlert.RepeatInterval)
		// If neither initial alert nor reminder needs to be sent, skip to the next alert
		if !sendInitialAlert && !sendReminder {
			if debug {
				log.Printf("[watchdog.handleAlertsToTrigger] Alert %s for endpoint=%s with description='%s' is not due for triggering (interval: %s last: %s), skipping",
					endpointAlert.Type, ep.Name, endpointAlert.GetDescription(), endpointAlert.RepeatInterval, lastReminder)
			}
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			var err error
			alertType := "reminder"
			if sendInitialAlert {
				alertType = "initial"
			}
			log.Printf("[watchdog.handleAlertsToTrigger] Sending %s %s alert because alert for endpoint=%s with description='%s' has been TRIGGERED", alertType, endpointAlert.Type, ep.Name, endpointAlert.GetDescription())
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
				// Mark initial alert as triggered and update last reminder time
				if sendInitialAlert {
					endpointAlert.Triggered = true
				}
				ep.LastReminderSent[endpointAlert.Type] = time.Now()
				if err := store.Get().UpsertTriggeredEndpointAlert(ep, endpointAlert); err != nil {
					log.Printf("[watchdog.handleAlertsToTrigger] Failed to persist triggered endpoint alert for endpoint with key=%s: %s", ep.Key(), err.Error())
				}
			}
		} else {
			log.Printf("[watchdog.handleAlertsToTrigger] Not sending alert of type=%s despite being TRIGGERED, because the provider wasn't configured properly", endpointAlert.Type)
		}
	}
}

func handleAlertsToResolve(ep *endpoint.Endpoint, result *endpoint.Result, alertingConfig *alerting.Config, debug bool) {
	ep.NumberOfSuccessesInARow++
	for _, endpointAlert := range ep.Alerts {
		isStillBelowSuccessThreshold := endpointAlert.SuccessThreshold > ep.NumberOfSuccessesInARow
		if isStillBelowSuccessThreshold && endpointAlert.IsEnabled() && endpointAlert.Triggered {
			// Persist NumberOfSuccessesInARow
			if err := store.Get().UpsertTriggeredEndpointAlert(ep, endpointAlert); err != nil {
				log.Printf("[watchdog.handleAlertsToResolve] Failed to update triggered endpoint alert for endpoint with key=%s: %s", ep.Key(), err.Error())
			}
		}
		if !endpointAlert.IsEnabled() || !endpointAlert.Triggered || isStillBelowSuccessThreshold {
			continue
		}
		// Even if the alert provider returns an error, we still set the alert's Triggered variable to false.
		// Further explanation can be found on Alert's Triggered field.
		endpointAlert.Triggered = false
		if err := store.Get().DeleteTriggeredEndpointAlert(ep, endpointAlert); err != nil {
			log.Printf("[watchdog.handleAlertsToResolve] Failed to delete persisted triggered endpoint alert for endpoint with key=%s: %s", ep.Key(), err.Error())
		}
		if !endpointAlert.IsSendingOnResolved() {
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			log.Printf("[watchdog.handleAlertsToResolve] Sending %s alert because alert for endpoint with key=%s with description='%s' has been RESOLVED", endpointAlert.Type, ep.Key(), endpointAlert.GetDescription())
			err := alertProvider.Send(ep, endpointAlert, result, true)
			if err != nil {
				log.Printf("[watchdog.handleAlertsToResolve] Failed to send an alert for endpoint with key=%s: %s", ep.Key(), err.Error())
			}
		} else {
			log.Printf("[watchdog.handleAlertsToResolve] Not sending alert of type=%s despite being RESOLVED, because the provider wasn't configured properly", endpointAlert.Type)
		}
		ep.LastReminderSent[endpointAlert.Type] = time.Now()
	}
	ep.NumberOfFailuresInARow = 0
}

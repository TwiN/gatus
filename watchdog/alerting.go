package watchdog

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/logr"
)

// HandleAlerting takes care of alerts to resolve and alerts to trigger based on result success or failure
func HandleAlerting(ep *endpoint.Endpoint, result *endpoint.Result, alertingConfig *alerting.Config) {
	if alertingConfig == nil {
		return
	}
	if result.Success {
		handleAlertsToResolve(ep, result, alertingConfig)
	} else {
		handleAlertsToTrigger(ep, result, alertingConfig)
	}
}

func handleAlertsToTrigger(ep *endpoint.Endpoint, result *endpoint.Result, alertingConfig *alerting.Config) {
	ep.NumberOfSuccessesInARow = 0
	ep.NumberOfFailuresInARow++
	// Store the current LastReminderSent time so all alert providers use the same reference time for reminder checks
	// This is important in case there are multiple alerts: if the first one sends a reminder, it would update the value
	// of ep.LastReminderSent (since ep is a pointer), so the second one would never send a reminder, even if it was due.
	// By storing the value in a local variable, we ensure all alerts use the same reference
	lastReminderSent := ep.LastReminderSent
	for _, endpointAlert := range ep.Alerts {
		// If the alert hasn't been triggered, move to the next one
		if !endpointAlert.IsEnabled() || endpointAlert.FailureThreshold > ep.NumberOfFailuresInARow {
			continue
		}
		// Determine if an initial alert should be sent
		sendInitialAlert := !endpointAlert.Triggered
		// Determine if a reminder should be sent
		sendReminder := endpointAlert.Triggered && endpointAlert.MinimumReminderInterval > 0 && time.Since(lastReminderSent) >= endpointAlert.MinimumReminderInterval
		// If neither initial alert nor reminder needs to be sent, skip to the next alert
		if !sendInitialAlert && !sendReminder {
			logr.Debugf("[watchdog.handleAlertsToTrigger] Alert for endpoint=%s with description='%s' is not due for triggering or reminding, skipping", ep.Name, endpointAlert.GetDescription())
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			logr.Infof("[watchdog.handleAlertsToTrigger] Sending %s alert because alert for endpoint with key=%s with description='%s' has been TRIGGERED", endpointAlert.Type, ep.Key(), endpointAlert.GetDescription())
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
				logr.Errorf("[watchdog.handleAlertsToTrigger] Failed to send an alert for endpoint with key=%s: %s", ep.Key(), err.Error())
			} else {
				// Mark initial alert as triggered and update last reminder time
				if sendInitialAlert {
					endpointAlert.Triggered = true
				}
				ep.LastReminderSent = time.Now()
				if err := store.Get().UpsertTriggeredEndpointAlert(ep, endpointAlert); err != nil {
					logr.Errorf("[watchdog.handleAlertsToTrigger] Failed to persist triggered endpoint alert for endpoint with key=%s: %s", ep.Key(), err.Error())
				}
			}
		} else {
			logr.Warnf("[watchdog.handleAlertsToTrigger] Not sending alert of type=%s endpoint with key=%s despite being TRIGGERED, because the provider wasn't configured properly", endpointAlert.Type, ep.Key())
		}
	}
}

func handleAlertsToResolve(ep *endpoint.Endpoint, result *endpoint.Result, alertingConfig *alerting.Config) {
	ep.NumberOfSuccessesInARow++
	for _, endpointAlert := range ep.Alerts {
		isStillBelowSuccessThreshold := endpointAlert.SuccessThreshold > ep.NumberOfSuccessesInARow
		if isStillBelowSuccessThreshold && endpointAlert.IsEnabled() && endpointAlert.Triggered {
			// Persist NumberOfSuccessesInARow
			if err := store.Get().UpsertTriggeredEndpointAlert(ep, endpointAlert); err != nil {
				logr.Errorf("[watchdog.handleAlertsToResolve] Failed to update triggered endpoint alert for endpoint with key=%s: %s", ep.Key(), err.Error())
			}
		}
		if !endpointAlert.IsEnabled() || !endpointAlert.Triggered || isStillBelowSuccessThreshold {
			continue
		}
		// Even if the alert provider returns an error, we still set the alert's Triggered variable to false.
		// Further explanation can be found on Alert's Triggered field.
		endpointAlert.Triggered = false
		if err := store.Get().DeleteTriggeredEndpointAlert(ep, endpointAlert); err != nil {
			logr.Errorf("[watchdog.handleAlertsToResolve] Failed to delete persisted triggered endpoint alert for endpoint with key=%s: %s", ep.Key(), err.Error())
		}
		if !endpointAlert.IsSendingOnResolved() {
			logr.Debugf("[watchdog.handleAlertsToResolve] Not sending request to provider of alert with type=%s for endpoint with key=%s despite being RESOLVED, because send-on-resolved is set to false", endpointAlert.Type, ep.Key())
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			logr.Infof("[watchdog.handleAlertsToResolve] Sending %s alert because alert for endpoint with key=%s with description='%s' has been RESOLVED", endpointAlert.Type, ep.Key(), endpointAlert.GetDescription())
			err := alertProvider.Send(ep, endpointAlert, result, true)
			if err != nil {
				logr.Errorf("[watchdog.handleAlertsToResolve] Failed to send an alert for endpoint with key=%s: %s", ep.Key(), err.Error())
			}
		} else {
			logr.Warnf("[watchdog.handleAlertsToResolve] Not sending alert of type=%s for endpoint with key=%s despite being RESOLVED, because the provider wasn't configured properly", endpointAlert.Type, ep.Key())
		}
	}
	ep.NumberOfFailuresInARow = 0
}

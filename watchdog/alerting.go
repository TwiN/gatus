package watchdog

import (
	"errors"
	"log/slog"
	"os"
	"time"

	"github.com/TwiN/gatus/v5/alerting"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage/store"
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
			slog.Debug("Alert not due for triggering or reminding", "endpoint", ep.Name, "description", endpointAlert.GetDescription())
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			slog.Info("Sending alert", "type", endpointAlert.Type, "endpoint", ep.Name, "description", endpointAlert.GetDescription())
			var err error
			alertType := "reminder"
			if sendInitialAlert {
				alertType = "initial"
			}
			slog.Info("Sending alert", "type", endpointAlert.Type, "alert_type", alertType, "endpoint", ep.Name, "description", endpointAlert.GetDescription())
			if os.Getenv("MOCK_ALERT_PROVIDER") == "true" {
				if os.Getenv("MOCK_ALERT_PROVIDER_ERROR") == "true" {
					err = errors.New("error")
				}
			} else {
				err = alertProvider.Send(ep, endpointAlert, result, false)
			}
			if err != nil {
				slog.Error("Failed to send alert", "type", endpointAlert.Type, "endpoint", ep.Name, "description", endpointAlert.GetDescription(), "error", err.Error())
			} else {
				// Mark initial alert as triggered and update last reminder time
				if sendInitialAlert {
					endpointAlert.Triggered = true
				}
				ep.LastReminderSent = time.Now()
				if err := store.Get().UpsertTriggeredEndpointAlert(ep, endpointAlert); err != nil {
					slog.Error("Failed to persist triggered endpoint alert", "endpoint", ep.Name, "description", endpointAlert.GetDescription(), "error", err.Error())
				}
			}
		} else {
			slog.Warn("Not sending alert because provider is not configured", "type", endpointAlert.Type, "endpoint", ep.Name, "description", endpointAlert.GetDescription())
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
				slog.Error("Failed to update triggered endpoint alert", "endpoint", ep.Name, "description", endpointAlert.GetDescription(), "error", err.Error())
			}
		}
		if !endpointAlert.IsEnabled() || !endpointAlert.Triggered || isStillBelowSuccessThreshold {
			continue
		}
		// Even if the alert provider returns an error, we still set the alert's Triggered variable to false.
		// Further explanation can be found on Alert's Triggered field.
		endpointAlert.Triggered = false
		if err := store.Get().DeleteTriggeredEndpointAlert(ep, endpointAlert); err != nil {
			slog.Error("Failed to delete persisted triggered endpoint alert", "endpoint", ep.Name, "description", endpointAlert.GetDescription(), "error", err.Error())
		}
		if !endpointAlert.IsSendingOnResolved() {
			slog.Debug("Not sending alert on resolved because send-on-resolved is false", "type", endpointAlert.Type, "endpoint", ep.Name)
			continue
		}
		alertProvider := alertingConfig.GetAlertingProviderByAlertType(endpointAlert.Type)
		if alertProvider != nil {
			slog.Info("Sending resolved alert", "type", endpointAlert.Type, "endpoint", ep.Name, "description", endpointAlert.GetDescription())
			err := alertProvider.Send(ep, endpointAlert, result, true)
			if err != nil {
				slog.Error("Failed to send resolved alert", "type", endpointAlert.Type, "endpoint", ep.Name, "description", endpointAlert.GetDescription(), "error", err.Error())
			}
		} else {
			slog.Warn("Not sending resolved alert because provider is not configured", "type", endpointAlert.Type, "endpoint", ep.Name)
		}
	}
	ep.NumberOfFailuresInARow = 0
}

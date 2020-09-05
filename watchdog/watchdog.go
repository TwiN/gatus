package watchdog

import (
	"encoding/json"
	"fmt"
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/metric"
	"log"
	"sync"
	"time"
)

var (
	serviceResults = make(map[string][]*core.Result)

	// serviceResultsMutex is used to prevent concurrent map access
	serviceResultsMutex sync.RWMutex

	// monitoringMutex is used to prevent multiple services from being evaluated at the same time.
	// Without this, conditions using response time may become inaccurate.
	monitoringMutex sync.Mutex
)

// GetJsonEncodedServiceResults returns a list of the last 20 results for each services encoded using json.Marshal.
// The reason why the encoding is done here is because we use a mutex to prevent concurrent map access.
func GetJsonEncodedServiceResults() ([]byte, error) {
	serviceResultsMutex.RLock()
	data, err := json.Marshal(serviceResults)
	serviceResultsMutex.RUnlock()
	return data, err
}

// Monitor loops over each services and starts a goroutine to monitor each services separately
func Monitor(cfg *config.Config) {
	for _, service := range cfg.Services {
		go monitor(service)
		// To prevent multiple requests from running at the same time
		time.Sleep(1111 * time.Millisecond)
	}
}

// monitor monitors a single service in a loop
func monitor(service *core.Service) {
	cfg := config.Get()
	for {
		// By placing the lock here, we prevent multiple services from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		monitoringMutex.Lock()
		if cfg.Debug {
			log.Printf("[watchdog][monitor] Monitoring serviceName=%s", service.Name)
		}
		result := service.EvaluateConditions()
		metric.PublishMetricsForService(service, result)
		serviceResultsMutex.Lock()
		serviceResults[service.Name] = append(serviceResults[service.Name], result)
		if len(serviceResults[service.Name]) > 20 {
			serviceResults[service.Name] = serviceResults[service.Name][1:]
		}
		serviceResultsMutex.Unlock()
		var extra string
		if !result.Success {
			extra = fmt.Sprintf("responseBody=%s", result.Body)
		}
		log.Printf(
			"[watchdog][monitor] Monitored serviceName=%s; success=%v; errors=%d; requestDuration=%s; %s",
			service.Name,
			result.Success,
			len(result.Errors),
			result.Duration.Round(time.Millisecond),
			extra,
		)
		handleAlerting(service, result)
		if cfg.Debug {
			log.Printf("[watchdog][monitor] Waiting for interval=%s before monitoring serviceName=%s again", service.Interval, service.Name)
		}
		monitoringMutex.Unlock()
		time.Sleep(service.Interval)
	}
}

func handleAlerting(service *core.Service, result *core.Result) {
	cfg := config.Get()
	if cfg.Alerting == nil {
		return
	}
	if result.Success {
		if service.NumberOfFailuresInARow > 0 {
			for _, alert := range service.Alerts {
				if !alert.Enabled || !alert.SendOnResolved || alert.Threshold > service.NumberOfFailuresInARow {
					continue
				}
				var alertProvider *core.CustomAlertProvider
				if alert.Type == core.SlackAlert {
					if len(cfg.Alerting.Slack) > 0 {
						log.Printf("[watchdog][monitor] Sending Slack alert because alert with description=%s has been resolved", alert.Description)
						alertProvider = core.CreateSlackCustomAlertProvider(cfg.Alerting.Slack, service, alert, result, true)
					} else {
						log.Printf("[watchdog][monitor] Not sending Slack alert despite being triggered, because there is no Slack webhook configured")
					}
				} else if alert.Type == core.TwilioAlert {
					if cfg.Alerting.Twilio != nil && cfg.Alerting.Twilio.IsValid() {
						log.Printf("[watchdog][monitor] Sending Twilio alert because alert with description=%s has been resolved", alert.Description)
						alertProvider = core.CreateTwilioCustomAlertProvider(cfg.Alerting.Twilio, fmt.Sprintf("RESOLVED: %s - %s", service.Name, alert.Description))
					} else {
						log.Printf("[watchdog][monitor] Not sending Twilio alert despite being resolved, because Twilio isn't configured properly")
					}
				} else if alert.Type == core.CustomAlert {
					if cfg.Alerting.Custom != nil && cfg.Alerting.Custom.IsValid() {
						log.Printf("[watchdog][monitor] Sending custom alert because alert with description=%s has been resolved", alert.Description)
						alertProvider = &core.CustomAlertProvider{
							Url:     cfg.Alerting.Custom.Url,
							Method:  cfg.Alerting.Custom.Method,
							Body:    cfg.Alerting.Custom.Body,
							Headers: cfg.Alerting.Custom.Headers,
						}
					} else {
						log.Printf("[watchdog][monitor] Not sending custom alert despite being resolved, because the custom provider isn't configured properly")
					}
				}
				if alertProvider != nil {
					err := alertProvider.Send(service.Name, alert.Description, true)
					if err != nil {
						log.Printf("[watchdog][monitor] Ran into error sending an alert: %s", err.Error())
					}
				}
			}
		}
		service.NumberOfFailuresInARow = 0
	} else {
		service.NumberOfFailuresInARow++
		for _, alert := range service.Alerts {
			// If the alert hasn't been triggered, move to the next one
			if !alert.Enabled || alert.Threshold != service.NumberOfFailuresInARow {
				continue
			}
			var alertProvider *core.CustomAlertProvider
			if alert.Type == core.SlackAlert {
				if len(cfg.Alerting.Slack) > 0 {
					log.Printf("[watchdog][monitor] Sending Slack alert because alert with description=%s has been triggered", alert.Description)
					alertProvider = core.CreateSlackCustomAlertProvider(cfg.Alerting.Slack, service, alert, result, false)
				} else {
					log.Printf("[watchdog][monitor] Not sending Slack alert despite being triggered, because there is no Slack webhook configured")
				}
			} else if alert.Type == core.TwilioAlert {
				if cfg.Alerting.Twilio != nil && cfg.Alerting.Twilio.IsValid() {
					log.Printf("[watchdog][monitor] Sending Twilio alert because alert with description=%s has been triggered", alert.Description)
					alertProvider = core.CreateTwilioCustomAlertProvider(cfg.Alerting.Twilio, fmt.Sprintf("TRIGGERED: %s - %s", service.Name, alert.Description))
				} else {
					log.Printf("[watchdog][monitor] Not sending Twilio alert despite being triggered, because Twilio config settings missing")
				}
			} else if alert.Type == core.CustomAlert {
				if cfg.Alerting.Custom != nil && cfg.Alerting.Custom.IsValid() {
					log.Printf("[watchdog][monitor] Sending custom alert because alert with description=%s has been triggered", alert.Description)
					alertProvider = &core.CustomAlertProvider{
						Url:     cfg.Alerting.Custom.Url,
						Method:  cfg.Alerting.Custom.Method,
						Body:    cfg.Alerting.Custom.Body,
						Headers: cfg.Alerting.Custom.Headers,
					}
				} else {
					log.Printf("[watchdog][monitor] Not sending custom alert despite being triggered, because there is no custom url configured")
				}
			}
			if alertProvider != nil {
				err := alertProvider.Send(service.Name, alert.Description, false)
				if err != nil {
					log.Printf("[watchdog][monitor] Ran into error sending an alert: %s", err.Error())
				}
			}
		}
	}
}

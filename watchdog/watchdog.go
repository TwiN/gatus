package watchdog

import (
	"encoding/base64"
	"fmt"
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/metric"
	"log"
	"net/url"
	"sync"
	"time"
)

var (
	serviceResults = make(map[string][]*core.Result)
	rwLock         sync.RWMutex
)

// GetServiceResults returns a list of the last 20 results for each services
func GetServiceResults() *map[string][]*core.Result {
	return &serviceResults
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
	for {
		// By placing the lock here, we prevent multiple services from being monitored at the exact same time, which
		// could cause performance issues and return inaccurate results
		rwLock.Lock()
		log.Printf("[watchdog][monitor] Monitoring serviceName=%s", service.Name)
		result := service.EvaluateConditions()
		metric.PublishMetricsForService(service, result)
		serviceResults[service.Name] = append(serviceResults[service.Name], result)
		if len(serviceResults[service.Name]) > 20 {
			serviceResults[service.Name] = serviceResults[service.Name][1:]
		}
		rwLock.Unlock()
		var extra string
		if !result.Success {
			extra = fmt.Sprintf("responseBody=%s", result.Body)
		}
		log.Printf(
			"[watchdog][monitor] Finished monitoring serviceName=%s; errors=%d; requestDuration=%s; %s",
			service.Name,
			len(result.Errors),
			result.Duration.Round(time.Millisecond),
			extra,
		)

		handleAlerting(service, result)

		log.Printf("[watchdog][monitor] Waiting for interval=%s before monitoring serviceName=%s", service.Interval, service.Name)
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
				if !alert.Enabled || !alert.SendOnResolved || alert.Threshold < service.NumberOfFailuresInARow {
					continue
				}
				// TODO
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
					alertProvider = &core.CustomAlertProvider{
						Url:     cfg.Alerting.Slack,
						Method:  "POST",
						Body:    fmt.Sprintf(`{"text":"*[Gatus]*\n*service:* %s\n*description:* %s"}`, service.Name, alert.Description),
						Headers: map[string]string{"Content-Type": "application/json"},
					}
				} else {
					log.Printf("[watchdog][monitor] Not sending Slack alert despite being triggered, because there is no Slack webhook configured")
				}
			} else if alert.Type == core.TwilioAlert {
				if cfg.Alerting.Twilio != nil && cfg.Alerting.Twilio.IsValid() {
					log.Printf("[watchdog][monitor] Sending Twilio alert because alert with description=%s has been triggered", alert.Description)
					alertProvider = &core.CustomAlertProvider{
						Url:    fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", cfg.Alerting.Twilio.SID),
						Method: "POST",
						Body: url.Values{
							"To":   {cfg.Alerting.Twilio.To},
							"From": {cfg.Alerting.Twilio.From},
							"Body": {fmt.Sprintf("%s - %s", service.Name, alert.Description)},
						}.Encode(),
						Headers: map[string]string{
							"Content-Type":  "application/x-www-form-urlencoded",
							"Authorization": fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.Alerting.Twilio.SID, cfg.Alerting.Twilio.Token)))),
						},
					}
				} else {
					log.Printf("[watchdog][monitor] Not sending Twilio alert despite being triggered, because twilio config settings missing")
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
				err := alertProvider.Send(service.Name, alert.Description)
				if err != nil {
					log.Printf("[watchdog][monitor] Ran into error sending an alert: %s", err.Error())
				}
			}
		}
	}
}

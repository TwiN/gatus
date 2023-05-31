package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/core"
	"github.com/google/uuid"
)

// AlertProvider is the configuration necessary for sending an alert using GitLab
type AlertProvider struct {
	WebhookURL       string `yaml:"webhook-url"`       // The webhook url provided by GitLab
	AuthorizationKey string `yaml:"authorization-key"` // The authorization key provided by GitLab

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Severity can be one of: critical, high, medium, low, info, unknown. Defaults to critical
	Severity string `yaml:"severity,omitempty"`
	// MonitoringTool overrides the name sent to gitlab. Defaults to gatus
	MonitoringTool string `yaml:"monitoring-tool,omitempty"`
	// EnvironmentName is the name of the associated GitLab environment. Required to display alerts on a dashboard.
	EnvironmentName string `yaml:"environment-name,omitempty"`
	// Service affected. Defaults to endpoint display name
	Service string `yaml:"service,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if len(provider.AuthorizationKey) == 0 || len(provider.WebhookURL) == 0 {
		return false
	}
	// Validate format of the repository URL
	_, err := url.Parse(provider.WebhookURL)
	if err != nil {
		return false
	}
	return true
}

// Send creates an issue in the designed RepositoryURL if the resolved parameter passed is false,
// or closes the relevant issue(s) if the resolved parameter passed is true.
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	if len(alert.ResolveKey) == 0 {
		alert.ResolveKey = uuid.NewString()
	}

	buffer := bytes.NewBuffer(provider.buildAlertBody(endpoint, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, provider.WebhookURL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", provider.AuthorizationKey))
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return err
}

type AlertBody struct {
	Title                 string `json:"title,omitempty"`                   // The title of the alert.
	Description           string `json:"description,omitempty"`             // A high-level summary of the problem.
	StartTime             string `json:"start_time,omitempty"`              // The time of the alert. If none is provided, a current time is used.
	EndTime               string `json:"end_time,omitempty"`                // The resolution time of the alert. If provided, the alert is resolved.
	Service               string `json:"service,omitempty"`                 // The affected service.
	MonitoringTool        string `json:"monitoring_tool,omitempty"`         // The name of the associated monitoring tool.
	Hosts                 string `json:"hosts,omitempty"`                   // One or more hosts, as to where this incident occurred.
	Severity              string `json:"severity,omitempty"`                // The severity of the alert. Case-insensitive. Can be one of: critical, high, medium, low, info, unknown. Defaults to critical if missing or value is not in this list.
	Fingerprint           string `json:"fingerprint,omitempty"`             // The unique identifier of the alert. This can be used to group occurrences of the same alert.
	GitlabEnvironmentName string `json:"gitlab_environment_name,omitempty"` // The name of the associated GitLab environment. Required to display alerts on a dashboard.
}

func (provider *AlertProvider) monitoringTool() string {
	if len(provider.MonitoringTool) > 0 {
		return provider.MonitoringTool
	}
	return "gatus"
}

func (provider *AlertProvider) service(endpoint *core.Endpoint) string {
	if len(provider.Service) > 0 {
		return provider.Service
	}
	return endpoint.DisplayName()
}

// buildAlertBody builds the body of the alert
func (provider *AlertProvider) buildAlertBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) []byte {
	body := AlertBody{
		Title:                 fmt.Sprintf("alert(%s): %s", provider.monitoringTool(), provider.service(endpoint)),
		StartTime:             result.Timestamp.Format(time.RFC3339),
		Service:               provider.service(endpoint),
		MonitoringTool:        provider.monitoringTool(),
		Hosts:                 endpoint.URL,
		GitlabEnvironmentName: provider.EnvironmentName,
		Severity:              provider.Severity,
		Fingerprint:           alert.ResolveKey,
	}
	if resolved {
		body.EndTime = result.Timestamp.Format(time.RFC3339)
	}

	var results string
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = ":white_check_mark:"
		} else {
			prefix = ":x:"
		}
		results += fmt.Sprintf("- %s - `%s`\n", prefix, conditionResult.Condition)
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = ":\n> " + alertDescription
	}
	var message string
	if resolved {
		message = fmt.Sprintf("An alert for *%s* has been resolved after passing successfully %d time(s) in a row", endpoint.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for *%s* has been triggered due to having failed %d time(s) in a row", endpoint.DisplayName(), alert.FailureThreshold)
	}
	body.Description = message + description + "\n\n## Condition results\n" + results

	json, _ := json.Marshal(body)
	return json
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

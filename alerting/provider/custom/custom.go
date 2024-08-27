package custom

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

// AlertProvider is the configuration necessary for sending an alert using a custom HTTP request
// Technically, all alert providers should be reachable using the custom alert provider
type AlertProvider struct {
	URL          string                       `yaml:"url"`
	Method       string                       `yaml:"method,omitempty"`
	Body         string                       `yaml:"body,omitempty"`
	Headers      map[string]string            `yaml:"headers,omitempty"`
	Placeholders map[string]map[string]string `yaml:"placeholders,omitempty"`

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client,omitempty"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if provider.ClientConfig == nil {
		provider.ClientConfig = client.GetDefaultConfig()
	}
	return len(provider.URL) > 0 && provider.ClientConfig != nil
}

// GetAlertStatePlaceholderValue returns the Placeholder value for ALERT_TRIGGERED_OR_RESOLVED if configured
func (provider *AlertProvider) GetAlertStatePlaceholderValue(resolved bool) string {
	status := "TRIGGERED"
	if resolved {
		status = "RESOLVED"
	}
	if _, ok := provider.Placeholders["ALERT_TRIGGERED_OR_RESOLVED"]; ok {
		if val, ok := provider.Placeholders["ALERT_TRIGGERED_OR_RESOLVED"][status]; ok {
			return val
		}
	}
	return status
}

func (provider *AlertProvider) buildHTTPRequest(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) *http.Request {
	body, url, method := provider.Body, provider.URL, provider.Method
	body = strings.ReplaceAll(body, "[ALERT_DESCRIPTION]", alert.GetDescription())
	url = strings.ReplaceAll(url, "[ALERT_DESCRIPTION]", alert.GetDescription())
	body = strings.ReplaceAll(body, "[ENDPOINT_NAME]", ep.Name)
	url = strings.ReplaceAll(url, "[ENDPOINT_NAME]", ep.Name)
	body = strings.ReplaceAll(body, "[ENDPOINT_GROUP]", ep.Group)
	url = strings.ReplaceAll(url, "[ENDPOINT_GROUP]", ep.Group)
	body = strings.ReplaceAll(body, "[ENDPOINT_URL]", ep.URL)
	url = strings.ReplaceAll(url, "[ENDPOINT_URL]", ep.URL)
	body = strings.ReplaceAll(body, "[RESULT_ERRORS]", strings.Join(result.Errors, ","))
	url = strings.ReplaceAll(url, "[RESULT_ERRORS]", strings.Join(result.Errors, ","))
	if resolved {
		body = strings.ReplaceAll(body, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(true))
		url = strings.ReplaceAll(url, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(true))
	} else {
		body = strings.ReplaceAll(body, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(false))
		url = strings.ReplaceAll(url, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(false))
	}
	if len(method) == 0 {
		method = http.MethodGet
	}
	bodyBuffer := bytes.NewBuffer([]byte(body))
	request, _ := http.NewRequest(method, url, bodyBuffer)
	for k, v := range provider.Headers {
		request.Header.Set(k, v)
	}
	return request
}

func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	request := provider.buildHTTPRequest(ep, alert, result, resolved)
	response, err := client.GetHTTPClient(provider.ClientConfig).Do(request)
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

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

package custom

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/client"
	"github.com/TwiN/gatus/v3/core"
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
	ClientConfig *client.Config `yaml:"client"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if provider.ClientConfig == nil {
		provider.ClientConfig = client.GetDefaultConfig()
	}
	return len(provider.URL) > 0 && provider.ClientConfig != nil
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
func (provider *AlertProvider) ToCustomAlertProvider(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) *AlertProvider {
	return provider
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

func (provider *AlertProvider) buildHTTPRequest(endpointName, alertDescription string, resolved bool) *http.Request {
	body := provider.Body
	providerURL := provider.URL
	method := provider.Method

	if strings.Contains(body, "[ALERT_DESCRIPTION]") {
		body = strings.ReplaceAll(body, "[ALERT_DESCRIPTION]", alertDescription)
	}
	if strings.Contains(body, "[SERVICE_NAME]") { // XXX: Remove this in v4.0.0
		body = strings.ReplaceAll(body, "[SERVICE_NAME]", endpointName)
	}
	if strings.Contains(body, "[ENDPOINT_NAME]") {
		body = strings.ReplaceAll(body, "[ENDPOINT_NAME]", endpointName)
	}
	if strings.Contains(body, "[ALERT_TRIGGERED_OR_RESOLVED]") {
		if resolved {
			body = strings.ReplaceAll(body, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(true))
		} else {
			body = strings.ReplaceAll(body, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(false))
		}
	}
	if strings.Contains(providerURL, "[ALERT_DESCRIPTION]") {
		providerURL = strings.ReplaceAll(providerURL, "[ALERT_DESCRIPTION]", alertDescription)
	}
	if strings.Contains(providerURL, "[SERVICE_NAME]") { // XXX: Remove this in v4.0.0
		providerURL = strings.ReplaceAll(providerURL, "[SERVICE_NAME]", endpointName)
	}
	if strings.Contains(providerURL, "[ENDPOINT_NAME]") {
		providerURL = strings.ReplaceAll(providerURL, "[ENDPOINT_NAME]", endpointName)
	}
	if strings.Contains(providerURL, "[ALERT_TRIGGERED_OR_RESOLVED]") {
		if resolved {
			providerURL = strings.ReplaceAll(providerURL, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(true))
		} else {
			providerURL = strings.ReplaceAll(providerURL, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(false))
		}
	}
	if len(method) == 0 {
		method = http.MethodGet
	}
	bodyBuffer := bytes.NewBuffer([]byte(body))
	request, _ := http.NewRequest(method, providerURL, bodyBuffer)
	for k, v := range provider.Headers {
		request.Header.Set(k, v)
	}
	return request
}

// Send a request to the alert provider and return the body
func (provider *AlertProvider) Send(endpointName, alertDescription string, resolved bool) ([]byte, error) {
	if os.Getenv("MOCK_ALERT_PROVIDER") == "true" {
		if os.Getenv("MOCK_ALERT_PROVIDER_ERROR") == "true" {
			return nil, errors.New("error")
		}
		return []byte("{}"), nil
	}
	request := provider.buildHTTPRequest(endpointName, alertDescription, resolved)
	response, err := client.GetHTTPClient(provider.ClientConfig).Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode > 399 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("call to provider alert returned status code %d", response.StatusCode)
		}
		return nil, fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return ioutil.ReadAll(response.Body)
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

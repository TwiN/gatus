package custom

import (
	"bytes"
	"fmt"
	"io"
	"maps"
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

// ReplacePlaceholder replaces occurrences of the placeholder in body, url and all headers with content
func (provider *AlertProvider) ReplacePlaceholder(placeholder string, content string, body *string, url *string, headers map[string]string) {
	*body = strings.ReplaceAll(*body, placeholder, content)
	*url = strings.ReplaceAll(*url, placeholder, content)
	for k, v := range headers {
		headers[k] = strings.ReplaceAll(v, placeholder, content)
	}
}

func (provider *AlertProvider) buildHTTPRequest(ep *endpoint.Endpoint, alert *alert.Alert, resolved bool) *http.Request {
	body, url, method, headers := provider.Body, provider.URL, provider.Method, maps.Clone(provider.Headers)
	provider.ReplacePlaceholder("[ALERT_DESCRIPTION]", alert.GetDescription(), &body, &url, headers)
	provider.ReplacePlaceholder("[ENDPOINT_NAME]", ep.Name, &body, &url, headers)
	provider.ReplacePlaceholder("[ENDPOINT_GROUP]", ep.Group, &body, &url, headers)
	provider.ReplacePlaceholder("[ENDPOINT_URL]", ep.URL, &body, &url, headers)
	if resolved {
		provider.ReplacePlaceholder("[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(true), &body, &url, headers)
	} else {
		provider.ReplacePlaceholder("[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(false), &body, &url, headers)
	}
	if len(method) == 0 {
		method = http.MethodGet
	}
	bodyBuffer := bytes.NewBuffer([]byte(body))
	request, _ := http.NewRequest(method, url, bodyBuffer)
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	return request
}

func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	request := provider.buildHTTPRequest(ep, alert, resolved)
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

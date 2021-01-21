package custom

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/TwinProduction/gatus/client"
	"github.com/TwinProduction/gatus/core"
)

// AlertProvider is the configuration necessary for sending an alert using a custom HTTP request
// Technically, all alert providers should be reachable using the custom alert provider
type AlertProvider struct {
	URL      string            `yaml:"url"`
	Method   string            `yaml:"method,omitempty"`
	Insecure bool              `yaml:"insecure,omitempty"`
	Body     string            `yaml:"body,omitempty"`
	Headers  map[string]string `yaml:"headers,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.URL) > 0
}

// ToCustomAlertProvider converts the provider into a custom.AlertProvider
func (provider *AlertProvider) ToCustomAlertProvider(service *core.Service, alert *core.Alert, result *core.Result, resolved bool) *AlertProvider {
	return provider
}

func (provider *AlertProvider) buildHTTPRequest(serviceName, alertDescription string, resolved bool) *http.Request {
	body := provider.Body
	providerURL := provider.URL
	method := provider.Method
	if strings.Contains(body, "[ALERT_DESCRIPTION]") {
		body = strings.ReplaceAll(body, "[ALERT_DESCRIPTION]", alertDescription)
	}
	if strings.Contains(body, "[SERVICE_NAME]") {
		body = strings.ReplaceAll(body, "[SERVICE_NAME]", serviceName)
	}
	if strings.Contains(body, "[ALERT_TRIGGERED_OR_RESOLVED]") {
		if resolved {
			body = strings.ReplaceAll(body, "[ALERT_TRIGGERED_OR_RESOLVED]", "RESOLVED")
		} else {
			body = strings.ReplaceAll(body, "[ALERT_TRIGGERED_OR_RESOLVED]", "TRIGGERED")
		}
	}
	if strings.Contains(providerURL, "[ALERT_DESCRIPTION]") {
		providerURL = strings.ReplaceAll(providerURL, "[ALERT_DESCRIPTION]", alertDescription)
	}
	if strings.Contains(providerURL, "[SERVICE_NAME]") {
		providerURL = strings.ReplaceAll(providerURL, "[SERVICE_NAME]", serviceName)
	}
	if strings.Contains(providerURL, "[ALERT_TRIGGERED_OR_RESOLVED]") {
		if resolved {
			providerURL = strings.ReplaceAll(providerURL, "[ALERT_TRIGGERED_OR_RESOLVED]", "RESOLVED")
		} else {
			providerURL = strings.ReplaceAll(providerURL, "[ALERT_TRIGGERED_OR_RESOLVED]", "TRIGGERED")
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
func (provider *AlertProvider) Send(serviceName, alertDescription string, resolved bool) ([]byte, error) {
	if os.Getenv("MOCK_ALERT_PROVIDER") == "true" {
		if os.Getenv("MOCK_ALERT_PROVIDER_ERROR") == "true" {
			return nil, errors.New("error")
		}
		return []byte("{}"), nil
	}
	request := provider.buildHTTPRequest(serviceName, alertDescription, resolved)
	response, err := client.GetHTTPClient(provider.Insecure).Do(request)
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

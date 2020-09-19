package custom

import (
	"bytes"
	"fmt"
	"github.com/TwinProduction/gatus/client"
	"io/ioutil"
	"net/http"
	"strings"
)

type AlertProvider struct {
	Url     string            `yaml:"url"`
	Method  string            `yaml:"method,omitempty"`
	Body    string            `yaml:"body,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

func (provider *AlertProvider) IsValid() bool {
	return len(provider.Url) > 0
}

func (provider *AlertProvider) buildRequest(serviceName, alertDescription string, resolved bool) *http.Request {
	body := provider.Body
	providerUrl := provider.Url
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
	if strings.Contains(providerUrl, "[ALERT_DESCRIPTION]") {
		providerUrl = strings.ReplaceAll(providerUrl, "[ALERT_DESCRIPTION]", alertDescription)
	}
	if strings.Contains(providerUrl, "[SERVICE_NAME]") {
		providerUrl = strings.ReplaceAll(providerUrl, "[SERVICE_NAME]", serviceName)
	}
	if strings.Contains(providerUrl, "[ALERT_TRIGGERED_OR_RESOLVED]") {
		if resolved {
			providerUrl = strings.ReplaceAll(providerUrl, "[ALERT_TRIGGERED_OR_RESOLVED]", "RESOLVED")
		} else {
			providerUrl = strings.ReplaceAll(providerUrl, "[ALERT_TRIGGERED_OR_RESOLVED]", "TRIGGERED")
		}
	}
	if len(method) == 0 {
		method = "GET"
	}
	bodyBuffer := bytes.NewBuffer([]byte(body))
	request, _ := http.NewRequest(method, providerUrl, bodyBuffer)
	for k, v := range provider.Headers {
		request.Header.Set(k, v)
	}
	return request
}

// Send a request to the alert provider and return the body
func (provider *AlertProvider) Send(serviceName, alertDescription string, resolved bool) ([]byte, error) {
	request := provider.buildRequest(serviceName, alertDescription, resolved)
	response, err := client.GetHttpClient().Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode > 399 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return nil, fmt.Errorf("call to provider alert returned status code %d", response.StatusCode)
		} else {
			return nil, fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
		}
	}
	return ioutil.ReadAll(response.Body)
}

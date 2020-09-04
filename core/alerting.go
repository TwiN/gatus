package core

import (
	"bytes"
	"fmt"
	"github.com/TwinProduction/gatus/client"
	"net/http"
	"strings"
)

type AlertingConfig struct {
	Slack  string               `yaml:"slack"`
	Twilio *TwilioAlertProvider `yaml:"twilio"`
	Custom *CustomAlertProvider `yaml:"custom"`
}

type TwilioAlertProvider struct {
	SID   string `yaml:"sid"`
	Token string `yaml:"token"`
	From  string `yaml:"from"`
	To    string `yaml:"to"`
}

type CustomAlertProvider struct {
	Url     string            `yaml:"url"`
	Method  string            `yaml:"method,omitempty"`
	Body    string            `yaml:"body,omitempty"`
	Headers map[string]string `yaml:"headers,omitempty"`
}

func (provider *CustomAlertProvider) buildRequest(serviceName, alertDescription string) *http.Request {
	body := provider.Body
	url := provider.Url
	if strings.Contains(provider.Body, "[ALERT_DESCRIPTION]") {
		body = strings.ReplaceAll(provider.Body, "[ALERT_DESCRIPTION]", alertDescription)
	}
	if strings.Contains(provider.Body, "[SERVICE_NAME]") {
		body = strings.ReplaceAll(provider.Body, "[SERVICE_NAME]", serviceName)
	}
	if strings.Contains(provider.Url, "[ALERT_DESCRIPTION]") {
		url = strings.ReplaceAll(provider.Url, "[ALERT_DESCRIPTION]", alertDescription)
	}
	if strings.Contains(provider.Url, "[SERVICE_NAME]") {
		url = strings.ReplaceAll(provider.Url, "[SERVICE_NAME]", serviceName)
	}
	bodyBuffer := bytes.NewBuffer([]byte(body))
	request, _ := http.NewRequest(provider.Method, url, bodyBuffer)
	for k, v := range provider.Headers {
		request.Header.Set(k, v)
	}
	return request
}

func (provider *CustomAlertProvider) Send(serviceName, alertDescription string) error {
	request := provider.buildRequest(serviceName, alertDescription)
	response, err := client.GetHttpClient().Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode > 399 {
		return fmt.Errorf("call to provider alert returned status code %d", response.StatusCode)
	}
	return nil
}

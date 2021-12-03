package teams

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/client"
	"github.com/TwiN/gatus/v3/core"
)

// AlertProvider is the configuration necessary for sending an alert using Teams
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.WebhookURL) > 0
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	if os.Getenv("MOCK_ALERT_PROVIDER") == "true" {
		if os.Getenv("MOCK_ALERT_PROVIDER_ERROR") == "true" {
			return errors.New("error")
		}
		return nil
	}
	buffer := bytes.NewBuffer([]byte(provider.buildRequestBody(endpoint, alert, result, resolved)))
	request, err := http.NewRequest(http.MethodPost, provider.WebhookURL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode > 399 {
		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return fmt.Errorf("call to provider alert returned status code %d", response.StatusCode)
		}
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return err
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) string {
	var message, color string
	if resolved {
		message = fmt.Sprintf("An alert for *%s* has been resolved after passing successfully %d time(s) in a row", endpoint.Name, alert.SuccessThreshold)
		color = "#36A64F"
	} else {
		message = fmt.Sprintf("An alert for *%s* has been triggered due to having failed %d time(s) in a row", endpoint.Name, alert.FailureThreshold)
		color = "#DD0000"
	}
	var results string
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "&#x2705;"
		} else {
			prefix = "&#x274C;"
		}
		results += fmt.Sprintf("%s - `%s`<br/>", prefix, conditionResult.Condition)
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = ":\\n> " + alertDescription
	}
	return fmt.Sprintf(`{
  "@type": "MessageCard",
  "@context": "http://schema.org/extensions",
  "themeColor": "%s",
  "title": "&#x1F6A8; Gatus",
  "text": "%s%s",
  "sections": [
    {
      "activityTitle": "URL",
      "text": "%s"
    },
    {
      "activityTitle": "Condition results",
      "text": "%s"
    }
  ]
}`, color, message, description, endpoint.URL, results)
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

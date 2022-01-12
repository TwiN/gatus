package gchat

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/client"
	"github.com/TwiN/gatus/v3/core"
)

// AlertProvider is the configuration necessary for sending an alert using Google chat
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"`
	// Url to your gatus instance
	GatusHost string `yaml:"gatus-url,omitempty"`

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
	return len(provider.WebhookURL) > 0
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer([]byte(provider.buildRequestBody(endpoint, alert, result, resolved)))
	request, err := http.NewRequest(http.MethodPost, provider.WebhookURL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(provider.ClientConfig).Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode > 399 {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	return err
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) string {
	var message, color string
	if resolved {
		color = "#36A64F"
		message = fmt.Sprintf("<font color='%s'>An alert has been resolved after passing successfully %d time(s) in a row</font>", color, alert.SuccessThreshold)
	} else {
		color = "#DD0000"
		message = fmt.Sprintf("<font color='%s'>An alert has been triggered due to having failed %d time(s) in a row</font>", color, alert.FailureThreshold)
	}
	var statusPage string
	statusPage = fmt.Sprintf("%s/endpoints/%s_%s", provider.GatusHost, endpoint.Group, endpoint.Name)
	var results string
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "✅"
		} else {
			prefix = "❌"
		}
		results += fmt.Sprintf("%s   %s<br>", prefix, conditionResult.Condition)
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = ":: " + alertDescription
	}
	return fmt.Sprintf(`{
    "cards": [
  {
    "sections": [
      {
        "widgets": [
          {
            "keyValue": {
              "topLabel": "%s [%s]",
              "content": "%s",
              "contentMultiline": "true",
              "bottomLabel": "%s",
              "icon": "BOOKMARK"
            }
          },
          {
            "keyValue": {
              "topLabel": "Condition results",
              "content": "%s",
              "contentMultiline": "true",
              "bottomLabel": "%s",
              "icon": "DESCRIPTION"
            }
          },
          {
            "buttons": [
              {
                "textButton": {
                  "text": "STATUS",
                  "onClick": {
                    "openLink": {
                      "url": "%s"
                    }
                  }
                }
              },
              {
                "textButton": {
                  "text": "URL",
                  "onClick": {
                    "openLink": {
                      "url": "%s"
                    }
                  }
                }
              }
            ]
          }
        ]
      }
    ]
  }
]
}`, endpoint.Name, endpoint.Group, message, description, results, statusPage, statusPage, endpoint.URL)
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

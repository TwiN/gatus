package googlechat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v4/alerting/alert"
	"github.com/TwiN/gatus/v4/client"
	"github.com/TwiN/gatus/v4/core"
)

// AlertProvider is the configuration necessary for sending an alert using Google chat
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"`

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client,omitempty"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

// Override is a case under which the default integration is overridden
type Override struct {
	Group      string `yaml:"group"`
	WebhookURL string `yaml:"webhook-url"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if provider.ClientConfig == nil {
		provider.ClientConfig = client.GetDefaultConfig()
	}

	registeredGroups := make(map[string]bool)
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" || len(override.WebhookURL) == 0 {
				return false
			}
			registeredGroups[override.Group] = true
		}
	}

	return len(provider.WebhookURL) > 0
}

// Send an alert using the provider
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildRequestBody(endpoint, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, provider.getWebhookURLForGroup(endpoint.Group), buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
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

type Body struct {
	Cards []Cards `json:"cards"`
}

type Cards struct {
	Sections []Sections `json:"sections"`
}

type Sections struct {
	Widgets []Widgets `json:"widgets"`
}

type Widgets struct {
	KeyValue KeyValue  `json:"keyValue,omitempty"`
	Buttons  []Buttons `json:"buttons,omitempty"`
}

type KeyValue struct {
	TopLabel         string `json:"topLabel"`
	Content          string `json:"content"`
	ContentMultiline string `json:"contentMultiline"`
	BottomLabel      string `json:"bottomLabel"`
	Icon             string `json:"icon"`
}

type Buttons struct {
	TextButton TextButton `json:"textButton"`
}

type TextButton struct {
	Text    string  `json:"text"`
	OnClick OnClick `json:"onClick"`
}

type OnClick struct {
	OpenLink OpenLink `json:"openLink"`
}

type OpenLink struct {
	URL string `json:"url"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) []byte {
	var message, color string
	if resolved {
		color = "#36A64F"
		message = fmt.Sprintf("<font color='%s'>An alert has been resolved after passing successfully %d time(s) in a row</font>", color, alert.SuccessThreshold)
	} else {
		color = "#DD0000"
		message = fmt.Sprintf("<font color='%s'>An alert has been triggered due to having failed %d time(s) in a row</font>", color, alert.FailureThreshold)
	}
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
	body, _ := json.Marshal(Body{
		Cards: []Cards{
			{
				Sections: []Sections{
					{
						Widgets: []Widgets{
							{
								KeyValue: KeyValue{
									TopLabel:         endpoint.DisplayName(),
									Content:          message,
									ContentMultiline: "true",
									BottomLabel:      description,
									Icon:             "BOOKMARK",
								},
							},
							{
								KeyValue: KeyValue{
									TopLabel:         "Condition results",
									Content:          results,
									ContentMultiline: "true",
									Icon:             "DESCRIPTION",
								},
							},
							{
								Buttons: []Buttons{
									{
										TextButton: TextButton{
											Text:    "Open",
											OnClick: OnClick{OpenLink: OpenLink{URL: endpoint.URL}},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	return body

	//	return fmt.Sprintf(`{
	//    "cards": [
	//  {
	//    "sections": [
	//      {
	//        "widgets": [
	//          {
	//            "keyValue": {
	//              "topLabel": "%s [%s]",
	//              "content": "%s",
	//              "contentMultiline": "true",
	//              "bottomLabel": "%s",
	//              "icon": "BOOKMARK"
	//            }
	//          },
	//          {
	//            "keyValue": {
	//              "topLabel": "Condition results",
	//              "content": "%s",
	//              "contentMultiline": "true",
	//              "icon": "DESCRIPTION"
	//            }
	//          },
	//          {
	//            "buttons": [
	//              {
	//                "textButton": {
	//                  "text": "URL",
	//                  "onClick": {
	//                    "openLink": {
	//                      "url": "%s"
	//                    }
	//                  }
	//                }
	//              }
	//            ]
	//          }
	//        ]
	//      }
	//    ]
	//  }
	//]
	//}`, endpoint.Name, endpoint.Group, message, description, results, endpoint.URL)
}

// getWebhookURLForGroup returns the appropriate Webhook URL integration to for a given group
func (provider *AlertProvider) getWebhookURLForGroup(group string) string {
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				return override.WebhookURL
			}
		}
	}
	return provider.WebhookURL
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

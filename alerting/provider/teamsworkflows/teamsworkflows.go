package teamsworkflows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

// AlertProvider is the configuration necessary for sending an alert using Teams
type AlertProvider struct {
	WebhookURL string `yaml:"webhook-url"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`

	// Title is the title of the message that will be sent
	Title string `yaml:"title,omitempty"`
}

// Override is a case under which the default integration is overridden
type Override struct {
	Group      string `yaml:"group"`
	WebhookURL string `yaml:"webhook-url"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
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
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildRequestBody(ep, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, provider.getWebhookURLForGroup(ep.Group), buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
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

// AdaptiveCardBody represents the structure of an Adaptive Card
type AdaptiveCardBody struct {
	Type    string      `json:"type"`
	Version string      `json:"version"`
	Body    []CardBody  `json:"body"`
	MSTeams MSTeamsBody `json:"msteams"`
}

// CardBody represents the body of the Adaptive Card
type CardBody struct {
	Type      string       `json:"type"`
	Text      string       `json:"text,omitempty"`
	Wrap      bool         `json:"wrap"`
	Separator bool         `json:"separator,omitempty"`
	Size      string       `json:"size,omitempty"`
	Weight    string       `json:"weight,omitempty"`
	Items     []CardBody   `json:"items,omitempty"`
	Facts     []Fact       `json:"facts,omitempty"`
	FactSet   *FactSetBody `json:"factSet,omitempty"`
	Style     string       `json:"style,omitempty"`
}

// MSTeamsBody represents the msteams options
type MSTeamsBody struct {
	Width string `json:"width"`
}

// FactSetBody represents the FactSet in the Adaptive Card
type FactSetBody struct {
	Type  string `json:"type"`
	Facts []Fact `json:"facts"`
}

// Fact represents an individual fact in the FactSet
type Fact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	var message string
	var themeColor string
	if resolved {
		message = fmt.Sprintf("An alert for **%s** has been resolved after passing successfully %d time(s) in a row.", ep.DisplayName(), alert.SuccessThreshold)
		themeColor = "Good" // green
	} else {
		message = fmt.Sprintf("An alert for **%s** has been triggered due to having failed %d time(s) in a row.", ep.DisplayName(), alert.FailureThreshold)
		themeColor = "Attention" // red
	}

	// Configure default title if it's not provided
	title := "&#x26D1; Gatus"
	if provider.Title != "" {
		title = provider.Title
	}

	// Build the facts from the condition results
	var facts []Fact
	for _, conditionResult := range result.ConditionResults {
		var key string
		if conditionResult.Success {
			key = "&#x2705;"
		} else {
			key = "&#x274C;"
		}
		facts = append(facts, Fact{
			Title: key,
			Value: conditionResult.Condition,
		})
	}

	cardContent := AdaptiveCardBody{
		Type:    "AdaptiveCard",
		Version: "1.4", // Version 1.5 and 1.6 doesn't seem to be supported by Teams as of 27/08/2024
		Body: []CardBody{
			{
				Type:  "Container",
				Style: themeColor,
				Items: []CardBody{
					{
						Type:  "Container",
						Style: "Default",
						Items: []CardBody{
							{
								Type:   "TextBlock",
								Text:   title,
								Size:   "Medium",
								Weight: "Bolder",
							},
							{
								Type: "TextBlock",
								Text: message,
								Wrap: true,
							},
							{
								Type:  "FactSet",
								Facts: facts,
							},
						},
					},
				},
			},
		},
		MSTeams: MSTeamsBody{
			Width: "Full",
		},
	}

	attachment := map[string]interface{}{
		"contentType": "application/vnd.microsoft.card.adaptive",
		"content":     cardContent,
	}

	payload := map[string]interface{}{
		"type":        "message",
		"attachments": []interface{}{attachment},
	}

	bodyAsJSON, _ := json.Marshal(payload)
	return bodyAsJSON
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
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

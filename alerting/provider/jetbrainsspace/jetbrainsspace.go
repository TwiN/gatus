package jetbrainsspace

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

// AlertProvider is the configuration necessary for sending an alert using JetBrains Space
type AlertProvider struct {
	Project   string `yaml:"project"`    // JetBrains Space Project name
	ChannelID string `yaml:"channel-id"` // JetBrains Space Chat Channel ID
	Token     string `yaml:"token"`      // JetBrains Space Bearer Token

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

// Override is a case under which the default integration is overridden
type Override struct {
	Group     string `yaml:"group"`
	ChannelID string `yaml:"channel-id"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	registeredGroups := make(map[string]bool)
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" || len(override.ChannelID) == 0 {
				return false
			}
			registeredGroups[override.Group] = true
		}
	}
	return len(provider.Project) > 0 && len(provider.ChannelID) > 0 && len(provider.Token) > 0
}

// Send an alert using the provider
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildRequestBody(ep, alert, result, resolved))
	url := fmt.Sprintf("https://%s.jetbrains.space/api/http/chats/messages/send-message", provider.Project)
	request, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+provider.Token)
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

type Body struct {
	Channel string  `json:"channel"`
	Content Content `json:"content"`
}

type Content struct {
	ClassName string    `json:"className"`
	Style     string    `json:"style"`
	Sections  []Section `json:"sections,omitempty"`
}

type Section struct {
	ClassName string    `json:"className"`
	Elements  []Element `json:"elements"`
	Header    string    `json:"header"`
}

type Element struct {
	ClassName string    `json:"className"`
	Accessory Accessory `json:"accessory"`
	Style     string    `json:"style"`
	Size      string    `json:"size"`
	Content   string    `json:"content"`
}

type Accessory struct {
	ClassName string `json:"className"`
	Icon      Icon   `json:"icon"`
	Style     string `json:"style"`
}

type Icon struct {
	Icon string `json:"icon"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	body := Body{
		Channel: "id:" + provider.getChannelIDForGroup(ep.Group),
		Content: Content{
			ClassName: "ChatMessage.Block",
			Sections: []Section{{
				ClassName: "MessageSection",
				Elements:  []Element{},
			}},
		},
	}
	if resolved {
		body.Content.Style = "SUCCESS"
		body.Content.Sections[0].Header = fmt.Sprintf("An alert for *%s* has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		body.Content.Style = "WARNING"
		body.Content.Sections[0].Header = fmt.Sprintf("An alert for *%s* has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	for _, conditionResult := range result.ConditionResults {
		icon := "warning"
		style := "WARNING"
		if conditionResult.Success {
			icon = "success"
			style = "SUCCESS"
		}
		body.Content.Sections[0].Elements = append(body.Content.Sections[0].Elements, Element{
			ClassName: "MessageText",
			Accessory: Accessory{
				ClassName: "MessageIcon",
				Icon:      Icon{Icon: icon},
				Style:     style,
			},
			Style:   style,
			Size:    "REGULAR",
			Content: conditionResult.Condition,
		})
	}
	bodyAsJSON, _ := json.Marshal(body)
	return bodyAsJSON
}

// getChannelIDForGroup returns the appropriate channel ID to for a given group override
func (provider *AlertProvider) getChannelIDForGroup(group string) string {
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				return override.ChannelID
			}
		}
	}
	return provider.ChannelID
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

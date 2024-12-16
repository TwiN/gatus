package jetbrainsspace

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrProjectNotSet          = errors.New("project not set")
	ErrChannelIDNotSet        = errors.New("channel-id not set")
	ErrTokenNotSet            = errors.New("token not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	Project   string `yaml:"project"`    // Project name
	ChannelID string `yaml:"channel-id"` // Chat Channel ID
	Token     string `yaml:"token"`      // Bearer Token
}

func (cfg *Config) Validate() error {
	if len(cfg.Project) == 0 {
		return ErrProjectNotSet
	}
	if len(cfg.ChannelID) == 0 {
		return ErrChannelIDNotSet
	}
	if len(cfg.Token) == 0 {
		return ErrTokenNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.Project) > 0 {
		cfg.Project = override.Project
	}
	if len(override.ChannelID) > 0 {
		cfg.ChannelID = override.ChannelID
	}
	if len(override.Token) > 0 {
		cfg.Token = override.Token
	}
}

// AlertProvider is the configuration necessary for sending an alert using JetBrains Space
type AlertProvider struct {
	DefaultConfig Config `yaml:",inline"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

// Override is a case under which the default integration is overridden
type Override struct {
	Group  string `yaml:"group"`
	Config `yaml:",inline"`
}

// Validate the provider's configuration
func (provider *AlertProvider) Validate() error {
	registeredGroups := make(map[string]bool)
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" {
				return ErrDuplicateGroupOverride
			}
			registeredGroups[override.Group] = true
		}
	}
	return provider.DefaultConfig.Validate()
}

// Send an alert using the provider
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(provider.buildRequestBody(cfg, ep, alert, result, resolved))
	url := fmt.Sprintf("https://%s.jetbrains.space/api/http/chats/messages/send-message", cfg.Project)
	request, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+cfg.Token)
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
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	body := Body{
		Channel: "id:" + cfg.ChannelID,
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

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// GetConfig returns the configuration for the provider with the overrides applied
func (provider *AlertProvider) GetConfig(group string, alert *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig
	// Handle group overrides
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				cfg.Merge(&override.Config)
				break
			}
		}
	}
	// Handle alert overrides
	if len(alert.ProviderOverride) != 0 {
		overrideConfig := Config{}
		if err := yaml.Unmarshal(alert.ProviderOverrideAsBytes(), &overrideConfig); err != nil {
			return nil, err
		}
		cfg.Merge(&overrideConfig)
	}
	// Validate the configuration
	err := cfg.Validate()
	return &cfg, err
}

// ValidateOverrides validates the alert's provider override and, if present, the group override
func (provider *AlertProvider) ValidateOverrides(group string, alert *alert.Alert) error {
	_, err := provider.GetConfig(group, alert)
	return err
}

package ntfy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

const (
	DefaultURL      = "https://ntfy.sh"
	DefaultPriority = 3
	TokenPrefix     = "tk_"
)

// AlertProvider is the configuration necessary for sending an alert using Slack
type AlertProvider struct {
	Topic           string `yaml:"topic"`
	URL             string `yaml:"url,omitempty"`              // Defaults to DefaultURL
	Priority        int    `yaml:"priority,omitempty"`         // Defaults to DefaultPriority
	Token           string `yaml:"token,omitempty"`            // Defaults to ""
	Email           string `yaml:"email,omitempty"`            // Defaults to ""
	Click           string `yaml:"click,omitempty"`            // Defaults to ""
	DisableFirebase bool   `yaml:"disable-firebase,omitempty"` // Defaults to false
	DisableCache    bool   `yaml:"disable-cache,omitempty"`    // Defaults to false

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// Overrides is a list of Override that may be prioritized over the default configuration
	Overrides []Override `yaml:"overrides,omitempty"`
}

// Override is a case under which the default integration is overridden
type Override struct {
	Group    string `yaml:"group"`
	Topic    string `yaml:"topic"`
	URL      string `yaml:"url"`
	Priority int    `yaml:"priority"`
	Token    string `yaml:"token"`
	Email    string `yaml:"email"`
	Click    string `yaml:"click"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if len(provider.URL) == 0 {
		provider.URL = DefaultURL
	}
	if provider.Priority == 0 {
		provider.Priority = DefaultPriority
	}
	isTokenValid := true
	if len(provider.Token) > 0 {
		isTokenValid = strings.HasPrefix(provider.Token, TokenPrefix)
	}
	registeredGroups := make(map[string]bool)
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if len(override.Group) == 0 {
				return false
			}
			if _, ok := registeredGroups[override.Group]; ok {
				return false
			}
			if len(override.Token) > 0 && !strings.HasPrefix(override.Token, TokenPrefix) {
				return false
			}
			if override.Priority < 0 || override.Priority >= 6 {
				return false
			}
			registeredGroups[override.Group] = true
		}
	}

	return len(provider.URL) > 0 && len(provider.Topic) > 0 && provider.Priority > 0 && provider.Priority < 6 && isTokenValid
}

// Send an alert using the provider
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	override := provider.getGroupOverride(ep.Group)
	buffer := bytes.NewBuffer(provider.buildRequestBody(ep, alert, result, resolved, override))
	url := provider.getURL(override)
	request, err := http.NewRequest(http.MethodPost, url, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	if token := provider.getToken(override); len(token) > 0 {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	if provider.DisableFirebase {
		request.Header.Set("Firebase", "no")
	}
	if provider.DisableCache {
		request.Header.Set("Cache", "no")
	}
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
	Topic    string   `json:"topic"`
	Title    string   `json:"title"`
	Message  string   `json:"message"`
	Tags     []string `json:"tags"`
	Priority int      `json:"priority"`
	Email    string   `json:"email,omitempty"`
	Click    string   `json:"click,omitempty"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool, override *Override) []byte {
	var message, formattedConditionResults, tag string
	if resolved {
		tag = "white_check_mark"
		message = "An alert has been resolved after passing successfully " + strconv.Itoa(alert.SuccessThreshold) + " time(s) in a row"
	} else {
		tag = "rotating_light"
		message = "An alert has been triggered due to having failed " + strconv.Itoa(alert.FailureThreshold) + " time(s) in a row"
	}
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "🟢"
		} else {
			prefix = "🔴"
		}
		formattedConditionResults += fmt.Sprintf("\n%s %s", prefix, conditionResult.Condition)
	}
	if len(alert.GetDescription()) > 0 {
		message += " with the following description: " + alert.GetDescription()
	}
	message += formattedConditionResults
	body, _ := json.Marshal(Body{
		Topic:    provider.getTopic(override),
		Title:    "Gatus: " + ep.DisplayName(),
		Message:  message,
		Tags:     []string{tag},
		Priority: provider.getPriority(override),
		Email:    provider.getEmail(override),
		Click:    provider.getClick(override),
	})
	return body
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

func (provider *AlertProvider) getGroupOverride(group string) *Override {
	if provider.Overrides != nil {
		for _, override := range provider.Overrides {
			if group == override.Group {
				return &override
			}
		}
	}
	return nil
}

func (provider *AlertProvider) getTopic(override *Override) string {
	if override != nil && len(override.Topic) > 0 {
		return override.Topic
	}
	return provider.Topic
}

func (provider *AlertProvider) getURL(override *Override) string {
	if override != nil && len(override.URL) > 0 {
		return override.URL
	}
	return provider.URL
}

func (provider *AlertProvider) getPriority(override *Override) int {
	if override != nil && override.Priority > 0 {
		return override.Priority
	}
	return provider.Priority
}

func (provider *AlertProvider) getToken(override *Override) string {
	if override != nil && len(override.Token) > 0 {
		return override.Token
	}
	return provider.Token
}

func (provider *AlertProvider) getEmail(override *Override) string {
	if override != nil && len(override.Email) > 0 {
		return override.Email
	}
	return provider.Email
}

func (provider *AlertProvider) getClick(override *Override) string {
	if override != nil && len(override.Click) > 0 {
		return override.Click
	}
	return provider.Click
}

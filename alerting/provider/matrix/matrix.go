package matrix

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

const defaultServerURL = "https://matrix-client.matrix.org"

var (
	ErrAccessTokenNotSet      = errors.New("access-token not set")
	ErrInternalRoomID         = errors.New("internal-room-id not set")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")
)

type Config struct {
	// ServerURL is the custom homeserver to use (optional)
	ServerURL string `yaml:"server-url"`

	// AccessToken is the bot user's access token to send messages
	AccessToken string `yaml:"access-token"`

	// InternalRoomID is the room that the bot user has permissions to send messages to
	InternalRoomID string `yaml:"internal-room-id"`
}

func (cfg *Config) Validate() error {
	if len(cfg.ServerURL) == 0 {
		cfg.ServerURL = defaultServerURL
	}
	if len(cfg.AccessToken) == 0 {
		return ErrAccessTokenNotSet
	}
	if len(cfg.InternalRoomID) == 0 {
		return ErrInternalRoomID
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.ServerURL) > 0 {
		cfg.ServerURL = override.ServerURL
	}
	if len(override.AccessToken) > 0 {
		cfg.AccessToken = override.AccessToken
	}
	if len(override.InternalRoomID) > 0 {
		cfg.InternalRoomID = override.InternalRoomID
	}
}

// AlertProvider is the configuration necessary for sending an alert using Matrix
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
			if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" || len(override.AccessToken) == 0 || len(override.InternalRoomID) == 0 {
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
	buffer := bytes.NewBuffer(provider.buildRequestBody(ep, alert, result, resolved))
	// The Matrix endpoint requires a unique transaction ID for each event sent
	txnId := randStringBytes(24)
	request, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s?access_token=%s",
			cfg.ServerURL,
			url.PathEscape(cfg.InternalRoomID),
			txnId,
			url.QueryEscape(cfg.AccessToken),
		),
		buffer,
	)
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

type Body struct {
	MsgType       string `json:"msgtype"`
	Format        string `json:"format"`
	Body          string `json:"body"`
	FormattedBody string `json:"formatted_body"`
}

// buildRequestBody builds the request body for the provider
func (provider *AlertProvider) buildRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	body, _ := json.Marshal(Body{
		MsgType:       "m.text",
		Format:        "org.matrix.custom.html",
		Body:          buildPlaintextMessageBody(ep, alert, result, resolved),
		FormattedBody: buildHTMLMessageBody(ep, alert, result, resolved),
	})
	return body
}

// buildPlaintextMessageBody builds the message body in plaintext to include in request
func buildPlaintextMessageBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) string {
	var message string
	if resolved {
		message = fmt.Sprintf("An alert for `%s` has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for `%s` has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	var formattedConditionResults string
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "✓"
		} else {
			prefix = "✕"
		}
		formattedConditionResults += fmt.Sprintf("\n%s - %s", prefix, conditionResult.Condition)
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = "\n" + alertDescription
	}
	return fmt.Sprintf("%s%s\n%s", message, description, formattedConditionResults)
}

// buildHTMLMessageBody builds the message body in HTML to include in request
func buildHTMLMessageBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) string {
	var message string
	if resolved {
		message = fmt.Sprintf("An alert for <code>%s</code> has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert for <code>%s</code> has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	var formattedConditionResults string
	if len(result.ConditionResults) > 0 {
		formattedConditionResults = "\n<h5>Condition results</h5><ul>"
		for _, conditionResult := range result.ConditionResults {
			var prefix string
			if conditionResult.Success {
				prefix = "✅"
			} else {
				prefix = "❌"
			}
			formattedConditionResults += fmt.Sprintf("<li>%s - <code>%s</code></li>", prefix, conditionResult.Condition)
		}
		formattedConditionResults += "</ul>"
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = fmt.Sprintf("\n<blockquote>%s</blockquote>", alertDescription)
	}
	return fmt.Sprintf("<h3>%s</h3>%s%s", message, description, formattedConditionResults)
}

func randStringBytes(n int) string {
	// All the compatible characters to use in a transaction ID
	const availableCharacterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = availableCharacterBytes[rand.Intn(len(availableCharacterBytes))]
	}
	return string(b)
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

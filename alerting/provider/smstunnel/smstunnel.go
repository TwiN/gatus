package smstunnel

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

const apiURL = "https://smstunnel.io/api/v1/sms/send"

var (
	ErrAPIKeyNotSet           = errors.New("api-key not set")
	ErrToNotSet               = errors.New("to not set")
	ErrInvalidTo              = errors.New("to must be exactly one phone number in international format, quoted, e.g. \"+14155550123\"")
	ErrDuplicateGroupOverride = errors.New("duplicate group override")

	// internationalNumberPattern matches one phone number in E.164 format
	internationalNumberPattern = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

	// httpClientConfig doesn't follow redirects, because following one would send the API key, the number and the
	// text to wherever it points
	httpClientConfig = func() *client.Config {
		cfg := client.GetDefaultConfig()
		cfg.IgnoreRedirect = true
		return cfg
	}()
)

type Config struct {
	APIKey   string `yaml:"api-key"`
	To       string `yaml:"to"`                  // One phone number in international format
	DeviceID string `yaml:"device-id,omitempty"` // Phone that sends the text. Defaults to the account's default phone
}

func (cfg *Config) Validate() error {
	if len(cfg.APIKey) == 0 {
		return ErrAPIKeyNotSet
	}
	if len(cfg.To) == 0 {
		return ErrToNotSet
	}
	if !internationalNumberPattern.MatchString(cfg.To) {
		return ErrInvalidTo
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.APIKey) > 0 {
		cfg.APIKey = override.APIKey
	}
	if len(override.To) > 0 {
		cfg.To = override.To
	}
	if len(override.DeviceID) > 0 {
		cfg.DeviceID = override.DeviceID
	}
}

// AlertProvider is the configuration necessary for sending an alert using SMSTunnel
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
	for _, override := range provider.Overrides {
		if isAlreadyRegistered := registeredGroups[override.Group]; isAlreadyRegistered || override.Group == "" {
			return ErrDuplicateGroupOverride
		}
		registeredGroups[override.Group] = true
		if len(override.To) > 0 && !internationalNumberPattern.MatchString(override.To) {
			return ErrInvalidTo
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
	payload, err := json.Marshal(provider.buildRequestBody(cfg, ep, alert, resolved))
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", cfg.APIKey)
	request.Header.Set("Idempotency-Key", buildIdempotencyKey(ep, alert, resolved, payload))
	response, err := client.GetHTTPClient(httpClientConfig).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode > 299 {
		// The response body is left out because it may quote the phone number, and this error ends up in the logs
		if response.Header.Get("Idempotent-Replayed") == "true" {
			return fmt.Errorf("call to provider alert returned status code %d for a repeat of an earlier attempt, so no new text was sent", response.StatusCode)
		}
		return fmt.Errorf("call to provider alert returned status code %d", response.StatusCode)
	}
	return nil
}

// buildIdempotencyKey returns the key identifying the text being sent, which is what makes a retry safe.
//
// When Send returns an error for a triggered alert, the watchdog calls it again on the next failed check, including
// after a timeout or a lost response, when the text may already have been accepted. The API answers a key it has
// seen in the last 24 hours with the original response instead of sending another text, so every attempt at a text
// must use the same key, and every new text a different one:
//   - alert.ResolveKey holds a random value for the incident, renewed on the incident's first attempt. It can't be
//     cleared on resolve, because Send isn't called on resolve when send-on-resolved is false
//   - a reminder also uses ep.LastReminderSent, which only moves once a send has succeeded
//   - the payload is part of the key, so that changing the number or the text is a new text rather than a conflict
func buildIdempotencyKey(ep *endpoint.Endpoint, alert *alert.Alert, resolved bool, payload []byte) string {
	isFirstAttempt := !resolved && !alert.Triggered && ep.NumberOfFailuresInARow <= alert.FailureThreshold
	if isFirstAttempt || len(alert.ResolveKey) == 0 {
		alert.ResolveKey = uuid.NewString()
	}
	textType := "triggered"
	if resolved {
		textType = "resolved"
	} else if alert.Triggered {
		textType = "reminder-" + strconv.FormatInt(ep.LastReminderSent.UnixMilli(), 10)
	}
	hash := sha256.New()
	hash.Write([]byte(alert.ResolveKey + "|" + textType + "|"))
	hash.Write(payload)
	return "gatus-" + hex.EncodeToString(hash.Sum(nil))
}

type Body struct {
	To       string `json:"to"`
	Message  string `json:"message"`
	DeviceID string `json:"deviceId,omitempty"`
}

// buildRequestBody builds the request body for the provider.
//
// The message must not contain anything that changes between two checks (e.g. a timestamp or a response time),
// because a retry has to be the same request as the attempt it repeats.
func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, resolved bool) Body {
	state := "TRIGGERED"
	if resolved {
		state = "RESOLVED"
	}
	message := fmt.Sprintf("%s: %s", state, ep.DisplayName())
	if description := alert.GetDescription(); len(description) > 0 {
		message += " - " + description
	}
	return Body{
		To:       cfg.To,
		Message:  message,
		DeviceID: cfg.DeviceID,
	}
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// GetConfig returns the configuration for the provider with the overrides applied
func (provider *AlertProvider) GetConfig(group string, alert *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig
	// Handle group overrides
	for _, override := range provider.Overrides {
		if group == override.Group {
			cfg.Merge(&override.Config)
			break
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

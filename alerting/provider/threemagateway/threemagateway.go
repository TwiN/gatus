package threemagateway

import (
	"encoding"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

const (
	defaultApiBaseUrl = "https://msgapi.threema.ch"

	defaultRecipientType = RecipientTypeId
)

var (
	errApiIdentityMissing   = errors.New("api-identity is required")
	errApiAuthSecretMissing = errors.New("auth-secret is required")
	errRecipientsMissing    = errors.New("at least one recipient is required")

	errRecipientsTooMany           = errors.New("too many recipients for the selected mode")
	errInvalidRecipientTypeForE2EE = errors.New("only recipient type 'id' is supported in e2ee modes")

	errE2EENotImplemented = errors.New("e2ee mode is not implemented yet")

	errInvalidRecipientFormat = errors.New("recipient must be in the format '[<type>:]<value>'")
	errInvalidRecipientType   = fmt.Errorf("invalid recipient type, must be one of: %v", joinKeys(validRecipientTypes, ", "))
	validRecipientTypes       = map[string]RecipientType{
		"id":    RecipientTypeId,
		"phone": RecipientTypePhone,
		"email": RecipientTypeEmail,
	}

	errInvalidThreemaId          = errors.New("invalid id: must be 8 characters long and alphabetic characters must be uppercase")
	errInvalidPhoneNumberFormat  = errors.New("invalid phone number: must contain only digits and may start with '+'")
	errInvalidEmailAddressFormat = errors.New("invalid email address: must contain '@'")
)

func joinKeys[V any](m map[string]V, separator string) string {
	return strings.Join(slices.Collect(maps.Keys(m)), separator)
}

type SendMode int

const (
	SendModeBasic SendMode = iota
	SendModeE2EE
	SendModeE2EEBulk
)

type RecipientType int

const (
	RecipientTypeInvalid RecipientType = iota
	RecipientTypeId
	RecipientTypePhone
	RecipientTypeEmail
)

func parseRecipientType(s string) RecipientType {
	if val, ok := validRecipientTypes[s]; ok {
		return val
	}
	return RecipientTypeInvalid
}

type Recipient struct {
	Value string        `yaml:"-"`
	Type  RecipientType `yaml:"-"`
}

var _ encoding.TextUnmarshaler = (*Recipient)(nil)
var _ encoding.TextMarshaler = (*Recipient)(nil)

func (r *Recipient) UnmarshalText(text []byte) error {
	parts := strings.Split(string(text), ":")
	switch {
	case len(parts) > 2:
		return errInvalidRecipientFormat
	case len(parts) == 2:
		if r.Type = parseRecipientType(parts[0]); r.Type == RecipientTypeInvalid {
			return errInvalidRecipientType
		}
		r.Value = parts[1]
	default:
		r.Type = defaultRecipientType
		r.Value = parts[0]
	}
	return nil
}

func (r Recipient) MarshalText() ([]byte, error) {
	if r.Type == RecipientTypeInvalid {
		return []byte("invalid" + ":" + r.Value), nil
	}
	for key, val := range validRecipientTypes {
		if val == r.Type {
			return []byte(key + ":" + r.Value), nil
		}
	}
	return nil, errInvalidRecipientType
}

func (r *Recipient) Validate() error {
	if len(r.Value) == 0 {
		return errInvalidRecipientFormat
	}
	switch r.Type {
	case RecipientTypeId:
		if err := validateThreemaId(r.Value); err != nil {
			return err
		}
	case RecipientTypePhone:
		r.Value = strings.TrimPrefix(r.Value, "+")
		if !isValidPhoneNumber(r.Value) {
			return errInvalidPhoneNumberFormat
		}
	case RecipientTypeEmail:
		if !strings.Contains(r.Value, "@") {
			return errInvalidEmailAddressFormat
		}
	default:
		return errInvalidRecipientType
	}
	return nil
}

func validateThreemaId(id string) error {
	if len(id) != 8 || strings.ToUpper(id) != id {
		return errInvalidThreemaId
	}
	return nil
}

func isValidPhoneNumber(number string) bool {
	if len(number) == 0 {
		return false
	}
	for _, ch := range number {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

type Config struct {
	ApiBaseUrl    string      `yaml:"api-base-url"`
	ApiIdentity   string      `yaml:"api-identity"`
	Recipients    []Recipient `yaml:"recipients"` // TODO#1470: Remove comment: This is a list to support bulk sending in e2ee-bulk mode once implemented
	ApiAuthSecret string      `yaml:"auth-secret"`
	PrivateKey    string      `yaml:"-,omitempty"` // TODO#1470: Enable in yaml once e2ee modes are implemented

	Mode SendMode `yaml:"-"`
}

func (cfg *Config) Validate() error {
	// Determine and validate mode
	switch {
	case len(cfg.PrivateKey) > 0 && len(cfg.Recipients) <= 1:
		cfg.Mode = SendModeE2EE
		return errE2EENotImplemented
	case len(cfg.PrivateKey) > 0 && len(cfg.Recipients) > 1:
		cfg.Mode = SendModeE2EEBulk
		return errE2EENotImplemented
	default:
		cfg.Mode = SendModeBasic
	}

	// Validate API Base URL
	if len(cfg.ApiBaseUrl) == 0 {
		cfg.ApiBaseUrl = defaultApiBaseUrl
	}

	// Validate API Identity
	if len(cfg.ApiIdentity) == 0 {
		return errApiIdentityMissing
	}
	if err := validateThreemaId(cfg.ApiIdentity); err != nil {
		return fmt.Errorf("api-identity: %w", err)
	}

	// Validate Recipients
	if len(cfg.Recipients) == 0 {
		return errRecipientsMissing
	}
	if cfg.Mode != SendModeE2EEBulk && len(cfg.Recipients) > 1 {
		return errRecipientsTooMany
	}
	for _, recipient := range cfg.Recipients {
		if err := recipient.Validate(); err != nil {
			return fmt.Errorf("recipients: %s: %w", recipient.Value, err)
		}
	}

	// Validate API Key
	if len(cfg.ApiAuthSecret) == 0 {
		return errApiAuthSecretMissing
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.ApiBaseUrl) > 0 {
		cfg.ApiBaseUrl = override.ApiBaseUrl
	}
	if len(override.ApiIdentity) > 0 {
		cfg.ApiIdentity = override.ApiIdentity
	}
	if len(override.Recipients) > 0 {
		cfg.Recipients = override.Recipients
	}
	if len(override.ApiAuthSecret) > 0 {
		cfg.ApiAuthSecret = override.ApiAuthSecret
	}
}

type AlertProvider struct {
	DefaultConfig Config       `yaml:",inline"`
	DefaultAlert  *alert.Alert `yaml:"default-alert,omitempty"`
	Overrides     []Override   `yaml:"overrides,omitempty"`
}

type Override struct {
	Group  string `yaml:"group"`
	Config `yaml:",inline"`
}

func (provider *AlertProvider) Validate() error {
	groupOverrideConfigured := map[string]bool{}
	for _, override := range provider.Overrides {
		if _, exists := groupOverrideConfigured[override.Group]; exists {
			return fmt.Errorf("duplicate override for group: %s", override.Group)
		}
		groupOverrideConfigured[override.Group] = true
	}
	return provider.DefaultConfig.Validate()
}

func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	body := provider.buildMessageBody(ep, alert, result, resolved)
	request, err := provider.prepareRequest(cfg, body)
	if err != nil {
		return err
	}
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	return handleResponse(cfg, response)
}

func (provider *AlertProvider) buildMessageBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) string {
	group := ep.Group
	if len(group) > 0 {
		group += "/"
	}

	var body string
	if resolved {
		body = fmt.Sprintf("✅ *Gatus: %s%s*\nAlert resolved after passing %d checks.", group, ep.Name, alert.SuccessThreshold)
	} else {
		body = fmt.Sprintf("🚨 *Gatus: %s%s*\nAlert triggered after failing %d checks.\nConditions:", group, ep.Name, alert.FailureThreshold)
		for _, conditionResult := range result.ConditionResults {
			icon := "❌"
			if conditionResult.Success {
				icon = "✅"
			}
			body += fmt.Sprintf("\n  %s %s", icon, conditionResult.Condition)
		}
		if len(result.Errors) > 0 {
			body += "\nErrors:"
			for _, err := range result.Errors {
				body += fmt.Sprintf("\n  ❌ %s", err)
			}
		}
	}
	return body
}

func (provider *AlertProvider) prepareRequest(cfg *Config, body string) (*http.Request, error) {
	requestUrl := cfg.ApiBaseUrl
	switch cfg.Mode {
	case SendModeBasic:
		requestUrl += "/send_simple"
	default:
		return nil, errE2EENotImplemented
	}

	data := url.Values{}
	data.Add("from", cfg.ApiIdentity)
	var toKey string
	switch cfg.Recipients[0].Type {
	case RecipientTypeId:
		toKey = "to"
	case RecipientTypePhone:
		toKey = "phone"
	case RecipientTypeEmail:
		toKey = "email"
	default:
		return nil, errInvalidRecipientType
	}
	data.Add(toKey, cfg.Recipients[0].Value)
	data.Add("text", body)
	data.Add("secret", cfg.ApiAuthSecret)

	request, err := http.NewRequest(http.MethodPost, requestUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request, nil
}

func handleResponse(cfg *Config, response *http.Response) error {
	switch response.StatusCode {
	case http.StatusOK:
		switch cfg.Mode {
		case SendModeBasic, SendModeE2EE:
			return nil
		case SendModeE2EEBulk:
			return nil // TODO#1470: Add correct handling once mode is implemented (check success fields in response body)
		}
	case http.StatusBadRequest:
		switch cfg.Mode {
		case SendModeBasic, SendModeE2EE:
			return fmt.Errorf("%s: Invalid recipient(s) or Threema Account not set up for configured mode", response.Status)
		case SendModeE2EEBulk:
			// TODO#1470: Add correct error info once mode is implemented
		}
	case http.StatusUnauthorized:
		return fmt.Errorf("%s: Invalid auth-secret or api-identity", response.Status)
	case http.StatusPaymentRequired:
		return fmt.Errorf("%s: Insufficient credits to send message", response.Status)
	case http.StatusNotFound:
		if cfg.Mode == SendModeBasic {
			return fmt.Errorf("%s: Recipient could not be found", response.Status)
		}
	}
	return fmt.Errorf("Response: %s", response.Status)
}

func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// GetConfig returns the configuration for the provider with the overrides applied
func (provider *AlertProvider) GetConfig(group string, alert *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig
	// Handle group overrides
	if len(provider.Overrides) > 0 {
		for _, override := range provider.Overrides {
			if group == override.Group {
				cfg.Merge(&override.Config)
				break
			}
		}
	}
	// Handle alert overrides
	if len(alert.ProviderOverride) > 0 {
		overrideConfig := Config{}
		if err := yaml.Unmarshal(alert.ProviderOverrideAsBytes(), &overrideConfig); err != nil {
			return nil, err
		}
		cfg.Merge(&overrideConfig)
	}
	err := cfg.Validate()
	return &cfg, err
}

func (provider *AlertProvider) ValidateOverrides(group string, alert *alert.Alert) error {
	_, err := provider.GetConfig(group, alert)
	return err
}

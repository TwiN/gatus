package threemagateway

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

const (
	defaultApiBaseUrl = "https://msgapi.threema.ch"
)

var (
	ErrApiIdentityMissing   = errors.New("api-identity is required")
	ErrApiAuthSecretMissing = errors.New("auth-secret is required")
	ErrRecipientsMissing    = errors.New("at least one recipient is required")

	ErrRecipientsTooMany = errors.New("too many recipients for the selected mode")
)

type Config struct {
	ApiBaseUrl    string      `yaml:"api-base-url"`
	Mode          *SendMode   `yaml:"send-mode"`
	ApiIdentity   string      `yaml:"api-identity"`
	Recipients    []Recipient `yaml:"recipients"` // TODO#1470: Remove comment: This is an array to support bulk sending in e2ee-bulk mode once implemented
	ApiAuthSecret string      `yaml:"auth-secret"`
}

func (cfg *Config) Validate() error {
	// Validate API Base URL
	if len(cfg.ApiBaseUrl) == 0 {
		cfg.ApiBaseUrl = defaultApiBaseUrl
	}

	// Validate Mode
	if cfg.Mode == nil {
		cfg.Mode = &SendMode{}
		cfg.Mode.UnmarshalText([]byte{})
	}
	switch cfg.Mode.Type {
	case ModeTypeInvalid:
		return ErrModeTypeInvalid
	case ModeTypeE2EE, ModeTypeE2EEBulk:
		return ErrNotImplementedMode // TODO#1470: implement E2EE and E2EE-Bulk modes
	}

	// Validate API Identity
	if len(cfg.ApiIdentity) == 0 {
		return ErrApiIdentityMissing
	}
	if err := validateThreemaId(cfg.ApiIdentity); err != nil {
		return fmt.Errorf("api-identity: %w", err)
	}

	// Validate Recipients
	var modeType = cfg.Mode.Type
	if modeType == ModeTypeBasic && len(cfg.Recipients) > 1 { // TODO#1470 Handle non bulk e2ee modes properly once implemented
		return ErrRecipientsTooMany
	} else if len(cfg.Recipients) == 0 {
		return ErrRecipientsMissing
	}
	for _, recipient := range cfg.Recipients {
		if err := recipient.Validate(); err != nil {
			return fmt.Errorf("recipients: %s: %w", recipient.Value, err)
		}
		// TODO#1470: Either support recipient types other than id in e2ee modes or handle the error properly once those modes are implemented
	}

	// Validate API Key
	if len(cfg.ApiAuthSecret) == 0 {
		return ErrApiAuthSecretMissing
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.ApiBaseUrl) > 0 {
		cfg.ApiBaseUrl = override.ApiBaseUrl
	}
	if override.Mode != nil {
		cfg.Mode = override.Mode
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
	// TODO#1464 Validate overrides?
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
	switch cfg.Mode.Type {
	case ModeTypeBasic:
		requestUrl += "/send_simple"
	case ModeTypeE2EE, ModeTypeE2EEBulk:
		return nil, ErrNotImplementedMode // TODO#1470: implement E2EE and E2EE-Bulk modes
	default:
		return nil, ErrNotImplementedMode
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
		return nil, ErrInvalidRecipientType
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
		switch cfg.Mode.Type {
		case ModeTypeBasic, ModeTypeE2EE:
			return nil
		case ModeTypeE2EEBulk:
			return nil // TODO#1470: Add correct handling once mode is implemented (check success fields in response body)
		}
	case http.StatusBadRequest:
		switch cfg.Mode.Type {
		case ModeTypeBasic, ModeTypeE2EE:
			return fmt.Errorf("%s: Invalid recipient or Threema Account not set up for %s mode", response.Status, cfg.Mode.Value)
		case ModeTypeE2EEBulk:
			// TODO#1470: Add correct error info once mode is implemented
		}
	case http.StatusUnauthorized:
		return fmt.Errorf("%s: Invalid auth-secret or api-identity", response.Status)
	case http.StatusPaymentRequired:
		return fmt.Errorf("%s: Insufficient credits to send message", response.Status)
	case http.StatusNotFound:
		if cfg.Mode.Type == ModeTypeBasic {
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

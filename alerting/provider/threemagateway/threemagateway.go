package threemagateway

import (
	"encoding"
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

// TODO#1464: Add tests

const (
	defaultApiUrl = "https://msgapi.threema.ch"
	defaultMode   = "basic"
)

var (
	ErrModeTypeInvalid    = fmt.Errorf("invalid mode, must be one of: %s", joinKeys(validModeTypes, ", "))
	ErrNotImplementedMode = errors.New("configured mode is not implemented")
	validModeTypes        = map[string]ModeType{
		"basic":     ModeTypeBasic,
		"e2ee":      ModeTypeE2EE,
		"e2ee-bulk": ModeTypeE2EEBulk,
	}

	ErrApiIdentityMissing   = errors.New("api-identity is required")
	ErrApiAuthSecretMissing = errors.New("auth-secret is required")
)

type ModeType int // TODO#1464: Maybe move to separate file in package to keep things organized

const (
	ModeTypeInvalid ModeType = iota
	ModeTypeBasic
	ModeTypeE2EE
	ModeTypeE2EEBulk
)

type SendMode struct {
	Value string   `yaml:"-"`
	Type  ModeType `yaml:"-"`
}

var _ encoding.TextUnmarshaler = (*SendMode)(nil)
var _ encoding.TextMarshaler = (*SendMode)(nil)

func (m *SendMode) UnmarshalText(text []byte) error {
	t := string(text)
	if len(t) == 0 {
		t = defaultMode
	}
	m.Value = t
	if val, ok := validModeTypes[t]; ok {
		m.Type = val
		return nil
	}
	m.Type = ModeTypeInvalid
	return ErrModeTypeInvalid
}

func (m SendMode) MarshalText() ([]byte, error) {
	return []byte(m.Value), nil
}

type Config struct {
	ApiUrl        string      `yaml:"api-url"`
	Mode          *SendMode   `yaml:"send-mode"`
	ApiIdentity   string      `yaml:"api-identity"`
	Recipients    []Recipient `yaml:"recipients"`
	ApiAuthSecret string      `yaml:"auth-secret"`
}

func (cfg *Config) Validate() error {
	// Validate API URL
	if len(cfg.ApiUrl) == 0 {
		cfg.ApiUrl = defaultApiUrl
	}

	// Validate Mode
	switch cfg.Mode.Type {
	case ModeTypeInvalid:
		return ErrModeTypeInvalid
	case ModeTypeE2EE, ModeTypeE2EEBulk:
		return ErrNotImplementedMode // TODO#1464: implement E2EE and E2EE-Bulk modes
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
	if (modeType == ModeTypeBasic || modeType == ModeTypeE2EE) && len(cfg.Recipients) > 1 {
		return fmt.Errorf("only one recipient is supported in '%s' mode", cfg.Mode.Value)
	} else if len(cfg.Recipients) == 0 {
		return errors.New("at least one recipient is required")
	}
	for _, recipient := range cfg.Recipients {
		if err := recipient.Validate(); err != nil {
			return fmt.Errorf("recipients: %s: %w", recipient.Value, err)
		}
		if modeType != ModeTypeBasic && recipient.Type == RecipientTypeId {
			return errors.New("recipient type 'id' is only supported in 'basic' mode") // TODO#1464: Maybe add support to fetch and cache IDs in other modes
		}
	}

	// Validate API Key
	if len(cfg.ApiAuthSecret) == 0 {
		return ErrApiAuthSecretMissing
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.ApiUrl) > 0 {
		cfg.ApiUrl = override.ApiUrl
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
	var body string
	if resolved {
		body = fmt.Sprintf("✅ *Gatus: %s/%s*\nAlert resolved after passing %d checks.", ep.Group, ep.Name, alert.SuccessThreshold)
	} else {
		body = fmt.Sprintf("🚨 *Gatus: %s/%s*\nAlert triggered after failing %d checks.", ep.Group, ep.Name, alert.FailureThreshold)
		for _, conditionResult := range result.ConditionResults {
			var icon rune
			if conditionResult.Success {
				icon = '✓'
			} else {
				icon = '✗'
			}
			body += fmt.Sprintf("\n- %c %s", icon, conditionResult.Condition)
		}
		if len(result.Errors) > 0 {
			body += "\nErrors:"
			for _, err := range result.Errors {
				body += fmt.Sprintf("\n- ✗ %s", err)
			}
		}
	}
	return body
}

func (provider *AlertProvider) prepareRequest(cfg *Config, body string) (*http.Request, error) {
	requestUrl := cfg.ApiUrl
	switch cfg.Mode.Type {
	case ModeTypeBasic:
		requestUrl += "/send_simple"
	case ModeTypeE2EE, ModeTypeE2EEBulk:
		return nil, ErrNotImplementedMode // TODO#1464: implement E2EE and E2EE-Bulk modes
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
			return nil // TODO#1464: Add correct handling once mode is implemented (check success fields in response body)
		}
	case http.StatusBadRequest:
		switch cfg.Mode.Type {
		case ModeTypeBasic, ModeTypeE2EE:
			return fmt.Errorf("%s: Invalid recipient or Threema Account not set up for %s mode", response.Status, cfg.Mode.Value)
		case ModeTypeE2EEBulk:
			// TODO#1464: Add correct error info once mode is implemented
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
	return &cfg, cfg.Validate()
}

func (provider *AlertProvider) ValidateOverrides(group string, alert *alert.Alert) error {
	_, err := provider.GetConfig(group, alert)
	return err
}

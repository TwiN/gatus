package dingtalk

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrTokenOrWebhookURLNotSet = errors.New("neither token nor webhook-url is set")
	ErrDuplicateGroupOverride  = errors.New("duplicate group override")
)

type Config struct {
	Token        string         `yaml:"token,omitempty"`
	Secret       string         `yaml:"secret,omitempty"`
	WebhookURL   string         `yaml:"webhook-url,omitempty"`
	Title        string         `yaml:"title,omitempty"`
	ClientConfig *client.Config `yaml:"client,omitempty"`
}

func (cfg *Config) Validate() error {
	if len(cfg.Token) == 0 && len(cfg.WebhookURL) == 0 {
		return ErrTokenOrWebhookURLNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if override.ClientConfig != nil {
		cfg.ClientConfig = override.ClientConfig
	}
	if len(override.Token) > 0 {
		cfg.Token = override.Token
		if len(override.WebhookURL) == 0 {
			cfg.WebhookURL = ""
		}
	}
	if len(override.Secret) > 0 {
		cfg.Secret = override.Secret
	}
	if len(override.WebhookURL) > 0 {
		cfg.WebhookURL = override.WebhookURL
		if len(override.Token) == 0 {
			cfg.Token = ""
		}
	}
	if len(override.Title) > 0 {
		cfg.Title = override.Title
	}
}

// sign computes the HMAC-SHA256 signature for DingTalk webhook URL
func (cfg *Config) sign(timestamp int64) string {
	stringToSign := strconv.FormatInt(timestamp, 10) + "\n" + cfg.Secret
	h := hmac.New(sha256.New, []byte(cfg.Secret))
	h.Write([]byte(stringToSign))
	return url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
}

// GetURL returns the final DingTalk webhook URL, dynamically computing HMAC-SHA256 signature if secret is provided
func (cfg *Config) GetURL() (string, error) {
	baseURL := cfg.WebhookURL
	if len(baseURL) == 0 {
		if len(cfg.Token) == 0 {
			return "", ErrTokenOrWebhookURLNotSet
		}
		baseURL = "https://oapi.dingtalk.com/robot/send?access_token=" + cfg.Token
	}
	if len(cfg.Secret) == 0 {
		return baseURL, nil
	}
	now := time.Now().UnixMilli()
	sign := cfg.sign(now)
	timestamp := strconv.FormatInt(now, 10)
	separator := "?"
	if strings.Contains(baseURL, "?") {
		separator = "&"
	}
	return fmt.Sprintf("%s%stimestamp=%s&sign=%s", baseURL, separator, timestamp, sign), nil
}

// AlertProvider is the configuration necessary for sending an alert using DingTalk
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
	requestURL, err := cfg.GetURL()
	if err != nil {
		return err
	}
	buffer := bytes.NewBuffer(provider.buildRequestBody(cfg, ep, alert, result, resolved))
	request, err := http.NewRequest(http.MethodPost, requestURL, buffer)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.GetHTTPClient(cfg.ClientConfig).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	if response.StatusCode > 399 {
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(body))
	}
	var dingResponse struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body, &dingResponse); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	if dingResponse.ErrCode != 0 {
		return fmt.Errorf("call to dingtalk returned error code %d: %s", dingResponse.ErrCode, dingResponse.ErrMsg)
	}
	return nil
}

type Body struct {
	MsgType  string   `json:"msgtype"`
	Markdown Markdown `json:"markdown"`
}

type Markdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	titlePrefix := "Gatus"
	if len(cfg.Title) > 0 {
		titlePrefix = cfg.Title
	}
	var statusEmoji, statusHeader string
	if resolved {
		statusEmoji = "🟢 [RESOLVED]"
		statusHeader = "🟢 **RESOLVED**"
	} else {
		statusEmoji = "🔴 [TRIGGERED]"
		statusHeader = "🔴 **TRIGGERED**"
	}
	title := fmt.Sprintf("[%s] %s %s", titlePrefix, statusEmoji, ep.DisplayName())
	var buffer strings.Builder
	buffer.WriteString(fmt.Sprintf("### [%s] %s: %s\n\n", titlePrefix, statusHeader, ep.DisplayName()))
	if len(ep.Group) > 0 {
		buffer.WriteString(fmt.Sprintf("- **Group**: `%s`\n", ep.Group))
	}
	if len(ep.URL) > 0 {
		buffer.WriteString(fmt.Sprintf("- **URL**: %s\n", ep.URL))
	}
	nowStr := time.Now().Format("2006-01-02 15:04:05")
	buffer.WriteString(fmt.Sprintf("- **Time**: %s\n", nowStr))
	if resolved {
		buffer.WriteString(fmt.Sprintf("- **Notice**: An alert for *%s* has been resolved after passing successfully %d time(s) in a row.\n", ep.DisplayName(), alert.SuccessThreshold))
	} else {
		buffer.WriteString(fmt.Sprintf("- **Notice**: An alert for *%s* has been triggered due to having failed %d time(s) in a row.\n", ep.DisplayName(), alert.FailureThreshold))
	}
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		buffer.WriteString(fmt.Sprintf("- **Description**: %s\n", alertDescription))
	}
	if len(result.Errors) > 0 {
		buffer.WriteString(fmt.Sprintf("- **Errors**: `%s`\n", strings.Join(result.Errors, ", ")))
	}
	if len(result.ConditionResults) > 0 {
		buffer.WriteString("\n**Conditions**:\n")
		for _, cr := range result.ConditionResults {
			prefix := ":x:"
			if cr.Success {
				prefix = ":white_check_mark:"
			}
			buffer.WriteString(fmt.Sprintf("- %s `%s`\n", prefix, cr.Condition))
		}
	}
	body := Body{
		MsgType: "markdown",
		Markdown: Markdown{
			Title: title,
			Text:  buffer.String(),
		},
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

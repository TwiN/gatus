package bark

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

const (
	DefaultServerURL         = "https://api.day.app"
	maxErrorResponseBodySize = 64 * 1024
)

var (
	ErrDeviceKeyNotSet           = errors.New("device key not set")
	ErrInvalidServerURL          = errors.New("invalid server URL")
	ErrInvalidLevel              = errors.New("invalid notification level")
	ErrGroupOverrideWithoutGroup = errors.New("group override without group")
	ErrDuplicateGroupOverride    = errors.New("duplicate group override")
	ErrCrossOriginRedirect       = errors.New("cross-origin redirect not allowed")
)

type Config struct {
	ServerURL string `yaml:"server-url,omitempty"`
	DeviceKey string `yaml:"device-key,omitempty"`
	Title     string `yaml:"title,omitempty"`
	Group     string `yaml:"group,omitempty"`
	Sound     string `yaml:"sound,omitempty"`
	Icon      string `yaml:"icon,omitempty"`
	URL       string `yaml:"url,omitempty"`
	Level     string `yaml:"level,omitempty"`
}

func (cfg *Config) Validate() error {
	if cfg.ServerURL == "" {
		cfg.ServerURL = DefaultServerURL
	}
	if err := cfg.validateOptionalFields(); err != nil {
		return err
	}
	if cfg.DeviceKey == "" {
		return ErrDeviceKeyNotSet
	}
	return nil
}

func (cfg *Config) validateOptionalFields() error {
	if cfg.ServerURL != "" {
		parsedURL, err := url.Parse(cfg.ServerURL)
		if err != nil || strings.ContainsAny(cfg.ServerURL, "?#") || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
			return ErrInvalidServerURL
		}
		cfg.ServerURL = strings.TrimRight(cfg.ServerURL, "/")
	}
	if cfg.Level != "" && cfg.Level != "critical" && cfg.Level != "active" && cfg.Level != "timeSensitive" && cfg.Level != "passive" {
		return ErrInvalidLevel
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if override == nil {
		return
	}
	if override.ServerURL != "" {
		cfg.ServerURL = override.ServerURL
	}
	if override.DeviceKey != "" {
		cfg.DeviceKey = override.DeviceKey
	}
	if override.Title != "" {
		cfg.Title = override.Title
	}
	if override.Group != "" {
		cfg.Group = override.Group
	}
	if override.Sound != "" {
		cfg.Sound = override.Sound
	}
	if override.Icon != "" {
		cfg.Icon = override.Icon
	}
	if override.URL != "" {
		cfg.URL = override.URL
	}
	if override.Level != "" {
		cfg.Level = override.Level
	}
}

type AlertProvider struct {
	DefaultConfig Config       `yaml:",inline"`
	DefaultAlert  *alert.Alert `yaml:"default-alert,omitempty"`
	Overrides     []Override   `yaml:"overrides,omitempty"`
}

type Override struct {
	Group  string `yaml:"-"`
	Config `yaml:"-"`
}

func (override *Override) UnmarshalYAML(value *yaml.Node) error {
	var selector struct {
		Group string `yaml:"group"`
	}
	if err := value.Decode(&selector); err != nil {
		return err
	}
	configNode := mappingNodeWithoutKey(value, "group")
	var config Config
	if err := configNode.Decode(&config); err != nil {
		return err
	}
	override.Group = selector.Group
	override.Config = config
	return nil
}

func (override Override) MarshalYAML() (any, error) {
	var configNode yaml.Node
	if err := configNode.Encode(override.Config); err != nil {
		return nil, err
	}
	configNode = mappingNodeWithoutKey(&configNode, "group")
	configNode.Content = append([]*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "group"},
		{Kind: yaml.ScalarNode, Value: override.Group},
	}, configNode.Content...)
	return &configNode, nil
}

func mappingNodeWithoutKey(value *yaml.Node, key string) yaml.Node {
	result := *value
	result.Content = make([]*yaml.Node, 0, len(value.Content))
	for i := 0; i+1 < len(value.Content); i += 2 {
		if value.Content[i].Value != key {
			result.Content = append(result.Content, value.Content[i], value.Content[i+1])
		}
	}
	return result
}

type Body struct {
	DeviceKey string `json:"device_key"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	Group     string `json:"group,omitempty"`
	Sound     string `json:"sound,omitempty"`
	Icon      string `json:"icon,omitempty"`
	URL       string `json:"url,omitempty"`
	Level     string `json:"level,omitempty"`
}

func (provider *AlertProvider) Validate() error {
	if err := provider.DefaultConfig.Validate(); err != nil {
		return err
	}
	groups := make(map[string]struct{}, len(provider.Overrides))
	for i := range provider.Overrides {
		override := &provider.Overrides[i]
		if override.Group == "" {
			return ErrGroupOverrideWithoutGroup
		}
		if _, exists := groups[override.Group]; exists {
			return ErrDuplicateGroupOverride
		}
		if err := override.Config.validateOptionalFields(); err != nil {
			return err
		}
		groups[override.Group] = struct{}{}
	}
	return nil
}

func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alertConfig *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alertConfig)
	if err != nil {
		return err
	}
	request, err := http.NewRequest(http.MethodPost, buildPushURL(cfg.ServerURL), bytes.NewReader(provider.buildRequestBody(cfg, ep, alertConfig, result, resolved)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	httpClient := *client.GetHTTPClient(nil)
	sharedRedirectPolicy := httpClient.CheckRedirect
	httpClient.CheckRedirect = func(redirectedRequest *http.Request, previousRequests []*http.Request) error {
		if len(previousRequests) != 0 && !sameOrigin(previousRequests[0].URL, redirectedRequest.URL) {
			return ErrCrossOriginRedirect
		}
		if sharedRedirectPolicy != nil {
			return sharedRedirectPolicy(redirectedRequest, previousRequests)
		}
		if len(previousRequests) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(io.LimitReader(response.Body, maxErrorResponseBodySize+1))
		truncated := len(body) > maxErrorResponseBodySize
		if truncated {
			body = body[:maxErrorResponseBodySize]
		}
		responseBody := sanitizeResponseBody(string(body), cfg.DeviceKey)
		if truncated {
			responseBody += " [truncated]"
		}
		return fmt.Errorf("call to Bark alert provider returned %s: %s", response.Status, responseBody)
	}
	return nil
}

func sanitizeResponseBody(responseBody, deviceKey string) string {
	responseBody = strings.ReplaceAll(responseBody, deviceKey, "[REDACTED]")
	return strings.Map(func(character rune) rune {
		if unicode.IsControl(character) || unicode.In(character, unicode.Zl, unicode.Zp) {
			return ' '
		}
		return character
	}, responseBody)
}

func sameOrigin(first, second *url.URL) bool {
	return strings.EqualFold(first.Scheme, second.Scheme) &&
		strings.EqualFold(first.Hostname(), second.Hostname()) &&
		effectivePort(first) == effectivePort(second)
}

func effectivePort(target *url.URL) string {
	if port := target.Port(); port != "" {
		return port
	}
	if target.Scheme == "http" {
		return "80"
	}
	if target.Scheme == "https" {
		return "443"
	}
	return ""
}

func buildPushURL(serverURL string) string {
	if strings.HasSuffix(serverURL, "/push") {
		return serverURL
	}
	return serverURL + "/push"
}

func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

func (provider *AlertProvider) buildRequestBody(cfg *Config, ep *endpoint.Endpoint, alertConfig *alert.Alert, result *endpoint.Result, resolved bool) []byte {
	title := cfg.Title
	if title == "" {
		title = "Gatus: " + ep.DisplayName()
	}
	var message string
	if resolved {
		message = fmt.Sprintf("An alert has been resolved after passing successfully %d time(s) in a row", alertConfig.SuccessThreshold)
	} else {
		message = fmt.Sprintf("An alert has been triggered due to having failed %d time(s) in a row", alertConfig.FailureThreshold)
	}
	if description := alertConfig.GetDescription(); description != "" {
		message += " with the following description: " + description
	}
	if result != nil {
		for _, conditionResult := range result.ConditionResults {
			prefix := "🔴"
			if conditionResult.Success {
				prefix = "🟢"
			}
			message += fmt.Sprintf("\n%s %s", prefix, conditionResult.Condition)
		}
	}
	body, _ := json.Marshal(Body{
		DeviceKey: cfg.DeviceKey,
		Title:     title,
		Body:      message,
		Group:     cfg.Group,
		Sound:     cfg.Sound,
		Icon:      cfg.Icon,
		URL:       cfg.URL,
		Level:     cfg.Level,
	})
	return body
}

func (provider *AlertProvider) GetConfig(group string, alertConfig *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig
	for i := range provider.Overrides {
		if provider.Overrides[i].Group == group {
			cfg.Merge(&provider.Overrides[i].Config)
			break
		}
	}
	if alertConfig != nil && len(alertConfig.ProviderOverride) != 0 {
		override := Config{}
		if err := yaml.Unmarshal(alertConfig.ProviderOverrideAsBytes(), &override); err != nil {
			return nil, err
		}
		cfg.Merge(&override)
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (provider *AlertProvider) ValidateOverrides(group string, alertConfig *alert.Alert) error {
	_, err := provider.GetConfig(group, alertConfig)
	return err
}

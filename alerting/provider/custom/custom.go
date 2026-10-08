package custom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrURLNotSet = errors.New("url not set")
)

type Config struct {
	URL          string                       `yaml:"url"`
	Method       string                       `yaml:"method,omitempty"`
	Body         string                       `yaml:"body,omitempty"`
	Headers      map[string]string            `yaml:"headers,omitempty"`
	Placeholders map[string]map[string]string `yaml:"placeholders,omitempty"`

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client,omitempty"`
}

func (cfg *Config) Validate() error {
	if len(cfg.URL) == 0 {
		return ErrURLNotSet
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if override.ClientConfig != nil {
		cfg.ClientConfig = override.ClientConfig
	}
	if len(override.URL) > 0 {
		cfg.URL = override.URL
	}
	if len(override.Method) > 0 {
		cfg.Method = override.Method
	}
	if len(override.Body) > 0 {
		cfg.Body = override.Body
	}
	if len(override.Headers) > 0 {
		cfg.Headers = override.Headers
	}
	if len(override.Placeholders) > 0 {
		cfg.Placeholders = override.Placeholders
	}
}

// AlertProvider is the configuration necessary for sending an alert using a custom HTTP request
// Technically, all alert providers should be reachable using the custom alert provider
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
	return provider.DefaultConfig.Validate()
}

func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	request := provider.buildHTTPRequest(cfg, ep, alert, result, resolved)
	response, err := client.GetHTTPClient(cfg.ClientConfig).Do(request)
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

func (provider *AlertProvider) buildHTTPRequest(cfg *Config, ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) *http.Request {
	body, url, method := cfg.Body, cfg.URL, cfg.Method
	// A body sent as JSON gets every value as the content of a JSON string, so that a quote,
	// a backslash or a newline in it (e.g. the response body a failed [BODY] condition shows)
	// cannot break the document. Other bodies get the values as they are.
	jsonBody := hasJSONContentType(cfg.Headers)
	escape := func(s string) string { return s }
	if jsonBody {
		escape = jsonStringContent
	}
	body = strings.ReplaceAll(body, "[ALERT_DESCRIPTION]", escape(alert.GetDescription()))
	url = strings.ReplaceAll(url, "[ALERT_DESCRIPTION]", alert.GetDescription())
	body = strings.ReplaceAll(body, "[ENDPOINT_NAME]", escape(ep.Name))
	url = strings.ReplaceAll(url, "[ENDPOINT_NAME]", ep.Name)
	body = strings.ReplaceAll(body, "[ENDPOINT_GROUP]", escape(ep.Group))
	url = strings.ReplaceAll(url, "[ENDPOINT_GROUP]", ep.Group)
	body = strings.ReplaceAll(body, "[ENDPOINT_URL]", escape(ep.URL))
	url = strings.ReplaceAll(url, "[ENDPOINT_URL]", ep.URL)
	// Outside a JSON body, only the quotes of the errors are escaped, as before.
	resultErrors := strings.ReplaceAll(strings.Join(result.Errors, ","), "\"", "\\\"")
	if jsonBody {
		body = strings.ReplaceAll(body, "[RESULT_ERRORS]", escape(strings.Join(result.Errors, ",")))
	} else {
		body = strings.ReplaceAll(body, "[RESULT_ERRORS]", resultErrors)
	}
	url = strings.ReplaceAll(url, "[RESULT_ERRORS]", resultErrors)

	if len(result.ConditionResults) > 0 && strings.Contains(body, "[RESULT_CONDITIONS]") {
		var formattedConditionResults string
		for index, conditionResult := range result.ConditionResults {
			var prefix string
			if conditionResult.Success {
				prefix = "✅"
			} else {
				prefix = "❌"
			}
			formattedConditionResults += fmt.Sprintf("%s - `%s`", prefix, conditionResult.Condition)
			if index < len(result.ConditionResults)-1 {
				formattedConditionResults += ", "
			}
		}
		body = strings.ReplaceAll(body, "[RESULT_CONDITIONS]", escape(formattedConditionResults))
		url = strings.ReplaceAll(url, "[RESULT_CONDITIONS]", formattedConditionResults)
	}

	if resolved {
		body = strings.ReplaceAll(body, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(cfg, true))
		url = strings.ReplaceAll(url, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(cfg, true))
	} else {
		body = strings.ReplaceAll(body, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(cfg, false))
		url = strings.ReplaceAll(url, "[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(cfg, false))
	}
	if len(method) == 0 {
		method = http.MethodGet
	}
	bodyBuffer := bytes.NewBuffer([]byte(body))
	request, _ := http.NewRequest(method, url, bodyBuffer)
	for k, v := range cfg.Headers {
		request.Header.Set(k, v)
	}
	return request
}

// GetAlertStatePlaceholderValue returns the Placeholder value for ALERT_TRIGGERED_OR_RESOLVED if configured
func (provider *AlertProvider) GetAlertStatePlaceholderValue(cfg *Config, resolved bool) string {
	status := "TRIGGERED"
	if resolved {
		status = "RESOLVED"
	}
	if _, ok := cfg.Placeholders["ALERT_TRIGGERED_OR_RESOLVED"]; ok {
		if val, ok := cfg.Placeholders["ALERT_TRIGGERED_OR_RESOLVED"][status]; ok {
			return val
		}
	}
	return status
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

// hasJSONContentType returns whether the Content-Type header of the request is JSON
// (application/json, or a type with the +json suffix such as application/vnd.api+json)
func hasJSONContentType(headers map[string]string) bool {
	for key, value := range headers {
		if strings.EqualFold(key, "Content-Type") {
			mediaType, _, err := mime.ParseMediaType(value)
			return err == nil && (mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"))
		}
	}
	return false
}

// jsonStringContent returns s escaped as the content of a JSON string, without the surrounding quotes
func jsonStringContent(s string) string {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(s)
	encoded := strings.TrimSuffix(buffer.String(), "\n")
	return encoded[1 : len(encoded)-1]
}

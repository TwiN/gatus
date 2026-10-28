package custom

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	body, method := cfg.Body, cfg.Method
	// Some placeholders resolve to text that the monitored service controls (result errors and conditions), so every value
	// is escaped for the place it is spliced into: URL-encoded in the URL, and escaped in the body for the format of the body.
	// The URL is split at its query so that a value is encoded for the path or for the query, whichever it lands in.
	urlPath, urlQuery, hasQuery := strings.Cut(cfg.URL, "?")
	values := [][2]string{
		{"[ALERT_DESCRIPTION]", alert.GetDescription()},
		{"[ENDPOINT_NAME]", ep.Name},
		{"[ENDPOINT_GROUP]", ep.Group},
		{"[ENDPOINT_URL]", ep.URL},
		{"[RESULT_ERRORS]", strings.Join(result.Errors, ",")},
		{"[ALERT_TRIGGERED_OR_RESOLVED]", provider.GetAlertStatePlaceholderValue(cfg, resolved)},
	}
	if len(result.ConditionResults) > 0 && strings.Contains(body+urlPath+urlQuery, "[RESULT_CONDITIONS]") {
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
		values = append(values, [2]string{"[RESULT_CONDITIONS]", formattedConditionResults})
	}
	// Each template is replaced in a single pass, so that a value that contains the name of a placeholder (the monitored
	// service can make it do so) is not expanded as if it were part of the template.
	format := bodyFormatOf(cfg)
	var bodyPairs, pathPairs, queryPairs []string
	for _, value := range values {
		bodyPairs = append(bodyPairs, value[0], escapeForBody(value[1], format))
		pathPairs = append(pathPairs, value[0], url.PathEscape(value[1]))
		queryPairs = append(queryPairs, value[0], url.QueryEscape(value[1]))
	}
	body = strings.NewReplacer(bodyPairs...).Replace(body)
	requestURL := strings.NewReplacer(pathPairs...).Replace(urlPath)
	if hasQuery {
		requestURL += "?" + strings.NewReplacer(queryPairs...).Replace(urlQuery)
	}
	if len(method) == 0 {
		method = http.MethodGet
	}
	bodyBuffer := bytes.NewBuffer([]byte(body))
	request, _ := http.NewRequest(method, requestURL, bodyBuffer)
	for k, v := range cfg.Headers {
		request.Header.Set(k, v)
	}
	return request
}

type bodyFormat int

const (
	bodyFormatPlain bodyFormat = iota
	bodyFormatJSON
	bodyFormatForm
)

// placeholderNames are the placeholders that can be in a template
var placeholderNames = []string{"[ALERT_DESCRIPTION]", "[ENDPOINT_NAME]", "[ENDPOINT_GROUP]", "[ENDPOINT_URL]", "[RESULT_ERRORS]", "[RESULT_CONDITIONS]", "[ALERT_TRIGGERED_OR_RESOLVED]"}

// bodyFormatOf returns the format of the body of the request. The Content-Type header decides when there is one, and
// otherwise the body is JSON if the template is valid JSON once its placeholders are replaced by a plain value.
func bodyFormatOf(cfg *Config) bodyFormat {
	for name, value := range cfg.Headers {
		if strings.EqualFold(name, "Content-Type") {
			contentType := strings.ToLower(value)
			switch {
			case strings.Contains(contentType, "json"):
				return bodyFormatJSON
			case strings.Contains(contentType, "application/x-www-form-urlencoded"):
				return bodyFormatForm
			}
			return bodyFormatPlain
		}
	}
	stubs := make([]string, 0, 2*len(placeholderNames))
	for _, name := range placeholderNames {
		stubs = append(stubs, name, "x")
	}
	if json.Valid([]byte(strings.NewReplacer(stubs...).Replace(cfg.Body))) {
		return bodyFormatJSON
	}
	return bodyFormatPlain
}

// escapeForBody escapes a placeholder value for the body: so that it can sit between the double quotes of a JSON string,
// or URL-encoded for a form. A body in another format only gets its double quotes escaped, as it always has.
func escapeForBody(value string, format bodyFormat) string {
	switch format {
	case bodyFormatForm:
		return url.QueryEscape(value)
	case bodyFormatJSON:
		var buffer bytes.Buffer
		encoder := json.NewEncoder(&buffer)
		encoder.SetEscapeHTML(false)
		if err := encoder.Encode(value); err != nil {
			return ""
		}
		// Encode wraps the string in double quotes and ends it with a newline
		encoded := strings.TrimSuffix(buffer.String(), "\n")
		return encoded[1 : len(encoded)-1]
	}
	return strings.ReplaceAll(value, "\"", "\\\"")
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

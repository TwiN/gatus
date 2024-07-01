package opsgenie

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
	restAPI = "https://api.opsgenie.com/v2/alerts"
)

type AlertProvider struct {
	// APIKey to use for
	APIKey string `yaml:"api-key"`

	// Priority to be used in Opsgenie alert payload
	//
	// default: P1
	Priority string `yaml:"priority"`

	// Source define source to be used in Opsgenie alert payload
	//
	// default: gatus
	Source string `yaml:"source"`

	// EntityPrefix is a prefix to be used in entity argument in Opsgenie alert payload
	//
	// default: gatus-
	EntityPrefix string `yaml:"entity-prefix"`

	//AliasPrefix is a prefix to be used in alias argument in Opsgenie alert payload
	//
	// default: gatus-healthcheck-
	AliasPrefix string `yaml:"alias-prefix"`

	// Tags to be used in Opsgenie alert payload
	//
	// default: []
	Tags []string `yaml:"tags"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	return len(provider.APIKey) > 0
}

// Send an alert using the provider
//
// Relevant: https://docs.opsgenie.com/docs/alert-api
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	err := provider.createAlert(ep, alert, result, resolved)
	if err != nil {
		return err
	}
	if resolved {
		err = provider.closeAlert(ep, alert)
		if err != nil {
			return err
		}
	}
	if alert.IsSendingOnResolved() {
		if resolved {
			// The alert has been resolved and there's no error, so we can clear the alert's ResolveKey
			alert.ResolveKey = ""
		} else {
			alert.ResolveKey = provider.alias(buildKey(ep))
		}
	}
	return nil
}

func (provider *AlertProvider) createAlert(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	payload := provider.buildCreateRequestBody(ep, alert, result, resolved)
	return provider.sendRequest(restAPI, http.MethodPost, payload)
}

func (provider *AlertProvider) closeAlert(ep *endpoint.Endpoint, alert *alert.Alert) error {
	payload := provider.buildCloseRequestBody(ep, alert)
	url := restAPI + "/" + provider.alias(buildKey(ep)) + "/close?identifierType=alias"
	return provider.sendRequest(url, http.MethodPost, payload)
}

func (provider *AlertProvider) sendRequest(url, method string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error build alert with payload %v: %w", payload, err)
	}
	request, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "GenieKey "+provider.APIKey)
	response, err := client.GetHTTPClient(nil).Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode > 399 {
		rBody, _ := io.ReadAll(response.Body)
		return fmt.Errorf("call to provider alert returned status code %d: %s", response.StatusCode, string(rBody))
	}
	return nil
}

func (provider *AlertProvider) buildCreateRequestBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) alertCreateRequest {
	var message, description string
	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", ep.Name, alert.GetDescription())
		description = fmt.Sprintf("An alert for *%s* has been resolved after passing successfully %d time(s) in a row", ep.DisplayName(), alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("%s - %s", ep.Name, alert.GetDescription())
		description = fmt.Sprintf("An alert for *%s* has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	}
	if ep.Group != "" {
		message = fmt.Sprintf("[%s] %s", ep.Group, message)
	}
	var formattedConditionResults string
	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "▣"
		} else {
			prefix = "▢"
		}
		formattedConditionResults += fmt.Sprintf("%s - `%s`\n", prefix, conditionResult.Condition)
	}
	description = description + "\n" + formattedConditionResults
	key := buildKey(ep)
	details := map[string]string{
		"endpoint:url":    ep.URL,
		"endpoint:group":  ep.Group,
		"result:hostname": result.Hostname,
		"result:ip":       result.IP,
		"result:dns_code": result.DNSRCode,
		"result:errors":   strings.Join(result.Errors, ","),
	}
	for k, v := range details {
		if v == "" {
			delete(details, k)
		}
	}
	if result.HTTPStatus > 0 {
		details["result:http_status"] = strconv.Itoa(result.HTTPStatus)
	}
	return alertCreateRequest{
		Message:     message,
		Description: description,
		Source:      provider.source(),
		Priority:    provider.priority(),
		Alias:       provider.alias(key),
		Entity:      provider.entity(key),
		Tags:        provider.Tags,
		Details:     details,
	}
}

func (provider *AlertProvider) buildCloseRequestBody(ep *endpoint.Endpoint, alert *alert.Alert) alertCloseRequest {
	return alertCloseRequest{
		Source: buildKey(ep),
		Note:   fmt.Sprintf("RESOLVED: %s - %s", ep.Name, alert.GetDescription()),
	}
}

func (provider *AlertProvider) source() string {
	source := provider.Source
	if source == "" {
		return "gatus"
	}
	return source
}

func (provider *AlertProvider) alias(key string) string {
	alias := provider.AliasPrefix
	if alias == "" {
		alias = "gatus-healthcheck-"
	}
	return alias + key
}

func (provider *AlertProvider) entity(key string) string {
	alias := provider.EntityPrefix
	if alias == "" {
		alias = "gatus-"
	}
	return alias + key
}

func (provider *AlertProvider) priority() string {
	priority := provider.Priority
	if priority == "" {
		return "P1"
	}
	return priority
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

func buildKey(ep *endpoint.Endpoint) string {
	name := toKebabCase(ep.Name)
	if ep.Group == "" {
		return name
	}
	return toKebabCase(ep.Group) + "-" + name
}

func toKebabCase(val string) string {
	return strings.ToLower(strings.ReplaceAll(val, " ", "-"))
}

type alertCreateRequest struct {
	Message     string            `json:"message"`
	Priority    string            `json:"priority"`
	Source      string            `json:"source"`
	Entity      string            `json:"entity"`
	Alias       string            `json:"alias"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags,omitempty"`
	Details     map[string]string `json:"details"`
}

type alertCloseRequest struct {
	Source string `json:"source"`
	Note   string `json:"note"`
}

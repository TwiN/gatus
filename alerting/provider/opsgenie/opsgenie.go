package opsgenie

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/TwiN/gatus/v3/alerting/alert"
	"github.com/TwiN/gatus/v3/client"
	"github.com/TwiN/gatus/v3/core"
	"io"
	"net/http"
	"strings"
)

const (
	restAPI = "https://api.opsgenie.com/v2/alerts"
)

type opsgenieAlertRequest struct {
	Message     string   `json:"message"`
	Priority    string   `json:"priority"`
	Source      string   `json:"source"`
	Entity      string   `json:"entity"`
	Alias       string   `json:"alias"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
}

type opsgenieAlertResponse struct {
	Result    string  `json:"result"`
	Took      float64 `json:"took"`
	RequestId string  `json:"requestId"`
}

type AlertProvider struct {
	APIKey string `yaml:"api-key"`

	//Priority define priority to be used in opsgenie alert payload
	// defaults: P1
	Priority string `yaml:"priority"`

	//EntityPrefix is a prefix to be used in entity argument in opsgenie alert payload
	// defaults: gatus-
	EntityPrefix string `yaml:"entity-prefix"`

	//AliasPrefix is a prefix to be used in alias argument in opsgenie alert payload
	// defaults: gatus-healthcheck-
	AliasPrefix string `yaml:"alias-prefix"`

	//tags define tags to be used in opsgenie alert payload
	// defaults: []
	Tags []string `yaml:"tags"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

func (provider *AlertProvider) IsValid() bool {
	return len(provider.APIKey) > 0
}

// Send an alert using the provider
//
// Relevant: https://docs.opsgenie.com/docs/alert-api
func (provider *AlertProvider) Send(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	err := provider.createAlert(endpoint, alert, result, resolved)

	if err != nil {
		return err
	}

	if resolved {
		err = provider.closeAlert(endpoint, alert)

		if err != nil {
			return err
		}
	}

	if alert.IsSendingOnResolved() {
		if resolved {
			// The alert has been resolved and there's no error, so we can clear the alert's ResolveKey
			alert.ResolveKey = ""
		} else {
			alert.ResolveKey = provider.alias(buildKey(endpoint))
		}
	}

	return nil
}

func (provider *AlertProvider) createAlert(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) error {
	buffer := bytes.NewBuffer(provider.buildCreateRequestBody(endpoint, alert, result, resolved))

	_, err := provider.sendRequest(restAPI, http.MethodPost,  buffer)

	return err
}

func (provider *AlertProvider) closeAlert(endpoint *core.Endpoint, alert *alert.Alert) error {
	buffer := bytes.NewBuffer(provider.buildCloseRequestBody(endpoint, alert))
	url := restAPI + "/" + provider.alias(buildKey(endpoint)) +  "/close?identifierType=alias"

	_, err := provider.sendRequest(url, http.MethodPost,  buffer)


	return err
}

func (provider *AlertProvider) sendRequest(url, method string, buffer *bytes.Buffer) (*http.Response, error) {
	request, err := http.NewRequest(method, url, buffer)

	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "GenieKey "+provider.APIKey)

	res, err := client.GetHTTPClient(nil).Do(request)

	if err != nil {
		return nil, err
	}

	if res.StatusCode > 399 {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("call to provider alert returned status code %d: %s", res.StatusCode, string(body))
	}

	return res, nil
}

func (provider *AlertProvider) buildCreateRequestBody(endpoint *core.Endpoint, alert *alert.Alert, result *core.Result, resolved bool) []byte {
	var message, description, results string

	key := buildKey(endpoint)

	if resolved {
		message = fmt.Sprintf("RESOLVED: %s - %s", endpoint.Name, alert.GetDescription())
		description = fmt.Sprintf("An alert for *%s* has been resolved after passing successfully %d time(s) in a row", endpoint.Name, alert.SuccessThreshold)
	} else {
		message = fmt.Sprintf("%s - %s", endpoint.Name, alert.GetDescription())
		description = fmt.Sprintf("An alert for *%s* has been triggered due to having failed %d time(s) in a row", endpoint.Name, alert.FailureThreshold)
	}

	for _, conditionResult := range result.ConditionResults {
		var prefix string
		if conditionResult.Success {
			prefix = "▣"
		} else {
			prefix = "▢"
		}
		results += fmt.Sprintf("%s - `%s`\n", prefix, conditionResult.Condition)
	}

	description = description + "\n" + results

	payload := opsgenieAlertRequest{
		Message:     "[TEST] " + message,
		Source:      key,
		Description: description,
		Priority:    provider.priority(),
		Alias:       provider.alias(key),
		Entity:      provider.alias(key),
		Tags:        provider.Tags,
	}

	b, _ := json.Marshal(payload)

	return b
}

func (provider *AlertProvider) buildCloseRequestBody(endpoint *core.Endpoint, alert *alert.Alert) []byte {
	payload := map[string]string{
		"source": buildKey(endpoint),
		"note": fmt.Sprintf("RESOLVED: %s - %s", endpoint.Name, alert.GetDescription()),
	}

	b, _ := json.Marshal(payload)

	return b
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

func (provider AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

func buildKey(endpoint *core.Endpoint) string {
	return strings.ToLower(strings.ReplaceAll(endpoint.Name, " ", "-"))
}

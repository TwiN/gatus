package jira

import (
	"bytes"
	"encoding/base64"
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
	ErrBaseURLNotSet    = errors.New("base-url not set")
	ErrUsernameNotSet   = errors.New("username not set")
	ErrTokenNotSet      = errors.New("token not set")
	ErrProjectKeyNotSet = errors.New("project-key not set")
)

const (
	defaultIssueType         = "Task"
	defaultResolveTransition = "Done"
)

type Config struct {
	BaseURL           string `yaml:"base-url"`    // e.g. https://your-domain.atlassian.net
	Username          string `yaml:"username"`    // Account email used for basic authentication
	Token             string `yaml:"token"`       // API token paired with the username
	ProjectKey        string `yaml:"project-key"` // Key of the project to create issues in, e.g. OPS
	IssueType         string `yaml:"issue-type,omitempty"`
	ResolveTransition string `yaml:"resolve-transition,omitempty"` // Transition used to close the issue when resolved
}

func (cfg *Config) Validate() error {
	if len(cfg.BaseURL) == 0 {
		return ErrBaseURLNotSet
	}
	if len(cfg.Username) == 0 {
		return ErrUsernameNotSet
	}
	if len(cfg.Token) == 0 {
		return ErrTokenNotSet
	}
	if len(cfg.ProjectKey) == 0 {
		return ErrProjectKeyNotSet
	}
	if len(cfg.IssueType) == 0 {
		cfg.IssueType = defaultIssueType
	}
	if len(cfg.ResolveTransition) == 0 {
		cfg.ResolveTransition = defaultResolveTransition
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.BaseURL) > 0 {
		cfg.BaseURL = override.BaseURL
	}
	if len(override.Username) > 0 {
		cfg.Username = override.Username
	}
	if len(override.Token) > 0 {
		cfg.Token = override.Token
	}
	if len(override.ProjectKey) > 0 {
		cfg.ProjectKey = override.ProjectKey
	}
	if len(override.IssueType) > 0 {
		cfg.IssueType = override.IssueType
	}
	if len(override.ResolveTransition) > 0 {
		cfg.ResolveTransition = override.ResolveTransition
	}
}

// AlertProvider is the configuration necessary for sending an alert using Jira
type AlertProvider struct {
	DefaultConfig Config `yaml:",inline"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// Validate the provider's configuration
func (provider *AlertProvider) Validate() error {
	return provider.DefaultConfig.Validate()
}

// Send creates an issue in the configured project if the resolved parameter passed is false,
// or transitions the relevant issue(s) to the resolve transition if the resolved parameter passed is true.
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	summary := "alert(gatus): " + ep.DisplayName()
	if !resolved {
		return provider.createIssue(cfg, summary, provider.buildIssueDescription(ep, alert, result))
	}
	return provider.resolveIssues(cfg, summary)
}

func (provider *AlertProvider) createIssue(cfg *Config, summary, description string) error {
	body, err := json.Marshal(map[string]any{
		"fields": map[string]any{
			"project":     map[string]string{"key": cfg.ProjectKey},
			"summary":     summary,
			"description": description,
			"issuetype":   map[string]string{"name": cfg.IssueType},
		},
	})
	if err != nil {
		return err
	}
	response, err := provider.sendRequest(cfg, http.MethodPost, "/rest/api/2/issue", body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("failed to create issue, status: %d, body: %s", response.StatusCode, readErrorBody(response))
	}
	return nil
}

// resolveIssues looks for open issues in the project that match the summary and transitions each of them.
// It pages through the search results so a matching issue is not missed when the project has more open
// issues than fit in a single response.
func (provider *AlertProvider) resolveIssues(cfg *Config, summary string) error {
	jql := fmt.Sprintf("project = %q AND statusCategory != Done ORDER BY created DESC", cfg.ProjectKey)
	startAt := 0
	for {
		path := fmt.Sprintf("/rest/api/2/search?jql=%s&startAt=%d&maxResults=100", url.QueryEscape(jql), startAt)
		response, err := provider.sendRequest(cfg, http.MethodGet, path, nil)
		if err != nil {
			return err
		}
		if response.StatusCode >= 400 {
			err := fmt.Errorf("failed to search issues, status: %d, body: %s", response.StatusCode, readErrorBody(response))
			response.Body.Close()
			return err
		}
		var searchResponse struct {
			Total  int `json:"total"`
			Issues []struct {
				Key    string `json:"key"`
				Fields struct {
					Summary string `json:"summary"`
				} `json:"fields"`
			} `json:"issues"`
		}
		if err := json.NewDecoder(response.Body).Decode(&searchResponse); err != nil {
			response.Body.Close()
			return err
		}
		response.Body.Close()
		for _, issue := range searchResponse.Issues {
			if issue.Fields.Summary == summary {
				if err := provider.transitionIssue(cfg, issue.Key); err != nil {
					return err
				}
			}
		}
		startAt += len(searchResponse.Issues)
		if len(searchResponse.Issues) == 0 || startAt >= searchResponse.Total {
			break
		}
	}
	return nil
}

// transitionIssue moves the given issue through the transition named by cfg.ResolveTransition
func (provider *AlertProvider) transitionIssue(cfg *Config, issueKey string) error {
	transitionsResponse, err := provider.sendRequest(cfg, http.MethodGet, "/rest/api/2/issue/"+issueKey+"/transitions", nil)
	if err != nil {
		return err
	}
	defer transitionsResponse.Body.Close()
	if transitionsResponse.StatusCode >= 400 {
		return fmt.Errorf("failed to fetch transitions for %s, status: %d, body: %s", issueKey, transitionsResponse.StatusCode, readErrorBody(transitionsResponse))
	}
	var transitions struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"transitions"`
	}
	if err := json.NewDecoder(transitionsResponse.Body).Decode(&transitions); err != nil {
		return err
	}
	var transitionID string
	for _, transition := range transitions.Transitions {
		if strings.EqualFold(transition.Name, cfg.ResolveTransition) {
			transitionID = transition.ID
			break
		}
	}
	if len(transitionID) == 0 {
		return fmt.Errorf("no transition named %q available for issue %s", cfg.ResolveTransition, issueKey)
	}
	body, err := json.Marshal(map[string]any{"transition": map[string]string{"id": transitionID}})
	if err != nil {
		return err
	}
	response, err := provider.sendRequest(cfg, http.MethodPost, "/rest/api/2/issue/"+issueKey+"/transitions", body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return fmt.Errorf("failed to transition issue %s, status: %d, body: %s", issueKey, response.StatusCode, readErrorBody(response))
	}
	return nil
}

// readErrorBody returns a trimmed, size-limited copy of the response body so Jira's error details
// (JQL parse errors, auth/permission failures, invalid transitions) surface in the returned error.
func readErrorBody(response *http.Response) string {
	body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
	return strings.TrimSpace(string(body))
}

func (provider *AlertProvider) sendRequest(cfg *Config, method, path string, body []byte) (*http.Response, error) {
	var payload io.Reader
	if body != nil {
		payload = bytes.NewBuffer(body)
	}
	request, err := http.NewRequest(method, strings.TrimSuffix(cfg.BaseURL, "/")+path, payload)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Token)))
	return client.GetHTTPClient(nil).Do(request)
}

// buildIssueDescription builds the description of the issue
func (provider *AlertProvider) buildIssueDescription(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result) string {
	var formattedConditionResults string
	if len(result.ConditionResults) > 0 {
		formattedConditionResults = "\n\nCondition results:\n"
		for _, conditionResult := range result.ConditionResults {
			status := "not satisfied"
			if conditionResult.Success {
				status = "satisfied"
			}
			formattedConditionResults += fmt.Sprintf("- %s (%s)\n", conditionResult.Condition, status)
		}
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = "\n" + alertDescription
	}
	message := fmt.Sprintf("An alert for %s has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
	return message + description + formattedConditionResults
}

// GetDefaultAlert returns the provider's default alert configuration
func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

// GetConfig returns the configuration for the provider with the overrides applied
func (provider *AlertProvider) GetConfig(group string, alert *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig
	// Handle alert overrides
	if len(alert.ProviderOverride) != 0 {
		overrideConfig := Config{}
		if err := yaml.Unmarshal(alert.ProviderOverrideAsBytes(), &overrideConfig); err != nil {
			return nil, err
		}
		cfg.Merge(&overrideConfig)
	}
	// Validate the configuration (we're returning the cfg here even if there's an error mostly for testing purposes)
	err := cfg.Validate()
	return &cfg, err
}

// ValidateOverrides validates the alert's provider override and, if present, the group override
func (provider *AlertProvider) ValidateOverrides(group string, alert *alert.Alert) error {
	_, err := provider.GetConfig(group, alert)
	return err
}

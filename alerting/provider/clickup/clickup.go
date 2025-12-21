package clickup

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrListIDNotSet = errors.New("list-id not set")
	ErrTokenNotSet  = errors.New("token not set")
)

type Config struct {
	APIURL          string   `yaml:"api-url"`
	ListID          string   `yaml:"list-id"`
	Token           string   `yaml:"token"`
	Assignees       []string `yaml:"assignees"`
	Status          string   `yaml:"status"`
	Priority        int      `yaml:"priority"`
	Name            string   `yaml:"name,omitempty"`
	MarkdownContent string   `yaml:"content,omitempty"`
}

func (cfg *Config) Validate() error {
	if cfg.ListID == "" {
		return ErrListIDNotSet
	}
	if cfg.Token == "" {
		return ErrTokenNotSet
	}
	if cfg.APIURL == "" {
		cfg.APIURL = "https://api.clickup.com/api/v2/list/" + cfg.ListID + "/task"
	}
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if override.ListID != "" {
		cfg.ListID = override.ListID
	}
	if override.Token != "" {
		cfg.Token = override.Token
	}
	if override.Status != "" {
		cfg.Status = override.Status
	}
	if override.Priority != 0 {
		cfg.Priority = override.Priority
	}
	if len(override.Assignees) > 0 {
		cfg.Assignees = override.Assignees
	}
	if override.Name != "" {
		cfg.Name = override.Name
	}
	if override.MarkdownContent != "" {
		cfg.MarkdownContent = override.MarkdownContent
	}
}

type AlertProvider struct {
	DefaultConfig Config       `yaml:",inline"`
	DefaultAlert  *alert.Alert `yaml:"default-alert,omitempty"`
}

func (provider *AlertProvider) Validate() error {
	return provider.DefaultConfig.Validate()
}

func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}

	if resolved {
		return provider.CloseTask(cfg, ep)
	}

	name := cfg.Name
	if name == "" {
		name = "Health Check: [ENDPOINT_GROUP]:[ENDPOINT_NAME]"
	}

	markdownContent := cfg.MarkdownContent
	if markdownContent == "" {
		markdownContent = "Triggered: [ENDPOINT_GROUP] - [ENDPOINT_NAME] - [ALERT_DESCRIPTION] - [RESULT_ERRORS]"
	}

	// Replace placeholders
	name = strings.ReplaceAll(name, "[ENDPOINT_GROUP]", ep.Group)
	name = strings.ReplaceAll(name, "[ENDPOINT_NAME]", ep.Name)

	markdownContent = strings.ReplaceAll(markdownContent, "[ENDPOINT_GROUP]", ep.Group)
	markdownContent = strings.ReplaceAll(markdownContent, "[ENDPOINT_NAME]", ep.Name)
	markdownContent = strings.ReplaceAll(markdownContent, "[ALERT_DESCRIPTION]", alert.GetDescription())
	markdownContent = strings.ReplaceAll(markdownContent, "[RESULT_ERRORS]", strings.Join(result.Errors, ", "))

	body := map[string]interface{}{
		"name":             name,
		"markdown_content": markdownContent,
		"assignees":        cfg.Assignees,
		"status":           cfg.Status,
		"priority":         cfg.Priority,
		"notify_all":       true,
	}

	return provider.CreateTask(cfg, body)
}

func (provider *AlertProvider) CreateTask(cfg *Config, body map[string]interface{}) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", cfg.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cfg.Token)

	httpClient := client.GetHTTPClient(nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to create task, status: %d", resp.StatusCode)
	}

	return nil
}

func (provider *AlertProvider) CloseTask(cfg *Config, ep *endpoint.Endpoint) error {
	fetchURL := fmt.Sprintf("https://api.clickup.com/api/v2/list/%s/task?include_closed=false", cfg.ListID)

	req, err := http.NewRequest("GET", fetchURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", cfg.Token)

	httpClient := client.GetHTTPClient(nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to fetch tasks, status: %d", resp.StatusCode)
	}

	var fetchResponse struct {
		Tasks []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fetchResponse); err != nil {
		return err
	}

	var matchingTaskIDs []string
	for _, task := range fetchResponse.Tasks {
		if strings.Contains(task.Name, ep.Group) && strings.Contains(task.Name, ep.Name) {
			matchingTaskIDs = append(matchingTaskIDs, task.ID)
		}
	}

	if len(matchingTaskIDs) == 0 {
		return fmt.Errorf("no matching tasks found for %s:%s", ep.Group, ep.Name)
	}

	for _, taskID := range matchingTaskIDs {
		if err := provider.UpdateTaskStatus(cfg, taskID, "closed"); err != nil {
			return fmt.Errorf("failed to close task %s: %v", taskID, err)
		}
	}

	return nil
}

func (provider *AlertProvider) UpdateTaskStatus(cfg *Config, taskID, status string) error {
	updateURL := fmt.Sprintf("https://api.clickup.com/api/v2/task/%s", taskID)
	body := map[string]interface{}{"status": status}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", cfg.Token)

	httpClient := client.GetHTTPClient(nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("failed to update task %s, status: %d", taskID, resp.StatusCode)
	}

	return nil
}

func (provider *AlertProvider) GetDefaultAlert() *alert.Alert {
	return provider.DefaultAlert
}

func (provider *AlertProvider) GetConfig(group string, alert *alert.Alert) (*Config, error) {
	cfg := provider.DefaultConfig
	if len(alert.ProviderOverride) != 0 {
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

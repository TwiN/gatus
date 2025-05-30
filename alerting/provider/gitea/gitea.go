package gitea

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"gopkg.in/yaml.v3"
)

var (
	ErrRepositoryURLNotSet  = errors.New("repository-url not set")
	ErrInvalidRepositoryURL = errors.New("invalid repository-url")
	ErrTokenNotSet          = errors.New("token not set")
)

type Config struct {
	RepositoryURL string   `yaml:"repository-url"`      // The URL of the Gitea repository to create issues in
	Token         string   `yaml:"token"`               // Token requires at least RW on issues and RO on metadata
	Assignees     []string `yaml:"assignees,omitempty"` // Assignees is a list of users to assign the issue to

	username        string
	repositoryOwner string
	repositoryName  string
	giteaClient     *gitea.Client

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client,omitempty"`
}

func (cfg *Config) Validate() error {
	if len(cfg.RepositoryURL) == 0 {
		return ErrRepositoryURLNotSet
	}
	if len(cfg.Token) == 0 {
		return ErrTokenNotSet
	}
	// Validate format of the repository URL
	repositoryURL, err := url.Parse(cfg.RepositoryURL)
	if err != nil {
		return err
	}
	baseURL := repositoryURL.Scheme + "://" + repositoryURL.Host
	pathParts := strings.Split(repositoryURL.Path, "/")
	if len(pathParts) != 3 {
		return ErrInvalidRepositoryURL
	}
	if cfg.repositoryOwner == pathParts[1] && cfg.repositoryName == pathParts[2] && cfg.giteaClient != nil {
		// Already validated, let's skip the rest of the validation to avoid unnecessary API calls
		return nil
	}
	cfg.repositoryOwner = pathParts[1]
	cfg.repositoryName = pathParts[2]
	opts := []gitea.ClientOption{
		gitea.SetToken(cfg.Token),
	}
	if cfg.ClientConfig != nil && cfg.ClientConfig.Insecure {
		// add new http client for skip verify
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		opts = append(opts, gitea.SetHTTPClient(httpClient))
	}
	cfg.giteaClient, err = gitea.NewClient(baseURL, opts...)
	if err != nil {
		return err
	}
	user, _, err := cfg.giteaClient.GetMyUserInfo()
	if err != nil {
		return err
	}
	cfg.username = user.UserName
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if override.ClientConfig != nil {
		cfg.ClientConfig = override.ClientConfig
	}
	if len(override.RepositoryURL) > 0 {
		cfg.RepositoryURL = override.RepositoryURL
	}
	if len(override.Token) > 0 {
		cfg.Token = override.Token
	}
	if len(override.Assignees) > 0 {
		cfg.Assignees = override.Assignees
	}
}

// AlertProvider is the configuration necessary for sending an alert using Discord
type AlertProvider struct {
	DefaultConfig Config `yaml:",inline"`

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`
}

// Validate the provider's configuration
func (provider *AlertProvider) Validate() error {
	return provider.DefaultConfig.Validate()
}

// Send creates an issue in the designed RepositoryURL if the resolved parameter passed is false,
// or closes the relevant issue(s) if the resolved parameter passed is true.
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	cfg, err := provider.GetConfig(ep.Group, alert)
	if err != nil {
		return err
	}
	title := "alert(gatus): " + ep.DisplayName()
	if !resolved {
		_, _, err = cfg.giteaClient.CreateIssue(
			cfg.repositoryOwner,
			cfg.repositoryName,
			gitea.CreateIssueOption{
				Title:     title,
				Body:      provider.buildIssueBody(ep, alert, result),
				Assignees: cfg.Assignees,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to create issue: %w", err)
		}
		return nil
	}
	issues, _, err := cfg.giteaClient.ListRepoIssues(
		cfg.repositoryOwner,
		cfg.repositoryName,
		gitea.ListIssueOption{
			State:     gitea.StateOpen,
			CreatedBy: cfg.username,
			ListOptions: gitea.ListOptions{
				PageSize: 100,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}
	for _, issue := range issues {
		if issue.Title == title {
			stateClosed := gitea.StateClosed
			_, _, err = cfg.giteaClient.EditIssue(
				cfg.repositoryOwner,
				cfg.repositoryName,
				issue.Index,
				gitea.EditIssueOption{
					State: &stateClosed,
				},
			)
			if err != nil {
				return fmt.Errorf("failed to close issue: %w", err)
			}
		}
	}
	return nil
}

// buildIssueBody builds the body of the issue
func (provider *AlertProvider) buildIssueBody(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result) string {
	var formattedConditionResults string
	if len(result.ConditionResults) > 0 {
		formattedConditionResults = "\n\n## Condition results\n"
		for _, conditionResult := range result.ConditionResults {
			var prefix string
			if conditionResult.Success {
				prefix = ":white_check_mark:"
			} else {
				prefix = ":x:"
			}
			formattedConditionResults += fmt.Sprintf("- %s - `%s`\n", prefix, conditionResult.Condition)
		}
	}
	var description string
	if alertDescription := alert.GetDescription(); len(alertDescription) > 0 {
		description = ":\n> " + alertDescription
	}
	message := fmt.Sprintf("An alert for **%s** has been triggered due to having failed %d time(s) in a row", ep.DisplayName(), alert.FailureThreshold)
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

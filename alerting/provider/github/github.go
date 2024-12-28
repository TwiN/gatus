package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

var (
	ErrRepositoryURLNotSet  = errors.New("repository-url not set")
	ErrInvalidRepositoryURL = errors.New("invalid repository-url")
	ErrTokenNotSet          = errors.New("token not set")
)

type Config struct {
	RepositoryURL string `yaml:"repository-url"` // The URL of the GitHub repository to create issues in
	Token         string `yaml:"token"`          // Token requires at least RW on issues and RO on metadata

	username        string
	repositoryOwner string
	repositoryName  string
	githubClient    *github.Client
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
	if cfg.repositoryOwner == pathParts[1] && cfg.repositoryName == pathParts[2] && cfg.githubClient != nil {
		// Already validated, let's skip the rest of the validation to avoid unnecessary API calls
		return nil
	}
	cfg.repositoryOwner = pathParts[1]
	cfg.repositoryName = pathParts[2]
	// Create oauth2 HTTP client with GitHub token
	httpClientWithStaticTokenSource := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: cfg.Token,
	}))
	// Create GitHub client
	if baseURL == "https://github.com" {
		cfg.githubClient = github.NewClient(httpClientWithStaticTokenSource)
	} else {
		cfg.githubClient, err = github.NewEnterpriseClient(baseURL, baseURL, httpClientWithStaticTokenSource)
		if err != nil {
			return fmt.Errorf("failed to create enterprise GitHub client: %w", err)
		}
	}
	// Retrieve the username once to validate that the token is valid
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	user, _, err := cfg.githubClient.Users.Get(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to retrieve GitHub user: %w", err)
	}
	cfg.username = *user.Login
	return nil
}

func (cfg *Config) Merge(override *Config) {
	if len(override.RepositoryURL) > 0 {
		cfg.RepositoryURL = override.RepositoryURL
	}
	if len(override.Token) > 0 {
		cfg.Token = override.Token
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
		_, _, err := cfg.githubClient.Issues.Create(context.Background(), cfg.repositoryOwner, cfg.repositoryName, &github.IssueRequest{
			Title: github.String(title),
			Body:  github.String(provider.buildIssueBody(ep, alert, result)),
		})
		if err != nil {
			return fmt.Errorf("failed to create issue: %w", err)
		}
	} else {
		issues, _, err := cfg.githubClient.Issues.ListByRepo(context.Background(), cfg.repositoryOwner, cfg.repositoryName, &github.IssueListByRepoOptions{
			State:       "open",
			Creator:     cfg.username,
			ListOptions: github.ListOptions{PerPage: 100},
		})
		if err != nil {
			return fmt.Errorf("failed to list issues: %w", err)
		}
		for _, issue := range issues {
			if *issue.Title == title {
				_, _, err = cfg.githubClient.Issues.Edit(context.Background(), cfg.repositoryOwner, cfg.repositoryName, *issue.Number, &github.IssueRequest{
					State: github.String("closed"),
				})
				if err != nil {
					return fmt.Errorf("failed to close issue: %w", err)
				}
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

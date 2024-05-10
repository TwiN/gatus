package github

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"
)

// AlertProvider is the configuration necessary for sending an alert using Discord
type AlertProvider struct {
	RepositoryURL string `yaml:"repository-url"` // The URL of the GitHub repository to create issues in
	Token         string `yaml:"token"`          // Token requires at least RW on issues and RO on metadata

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	username        string
	repositoryOwner string
	repositoryName  string
	githubClient    *github.Client
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if len(provider.Token) == 0 || len(provider.RepositoryURL) == 0 {
		return false
	}
	// Validate format of the repository URL
	repositoryURL, err := url.Parse(provider.RepositoryURL)
	if err != nil {
		return false
	}
	baseURL := repositoryURL.Scheme + "://" + repositoryURL.Host
	pathParts := strings.Split(repositoryURL.Path, "/")
	if len(pathParts) != 3 {
		return false
	}
	provider.repositoryOwner = pathParts[1]
	provider.repositoryName = pathParts[2]
	// Create oauth2 HTTP client with GitHub token
	httpClientWithStaticTokenSource := oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: provider.Token,
	}))
	// Create GitHub client
	if baseURL == "https://github.com" {
		provider.githubClient = github.NewClient(httpClientWithStaticTokenSource)
	} else {
		provider.githubClient, err = github.NewEnterpriseClient(baseURL, baseURL, httpClientWithStaticTokenSource)
		if err != nil {
			return false
		}
	}
	// Retrieve the username once to validate that the token is valid
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	user, _, err := provider.githubClient.Users.Get(ctx, "")
	if err != nil {
		return false
	}
	provider.username = *user.Login
	return true
}

// Send creates an issue in the designed RepositoryURL if the resolved parameter passed is false,
// or closes the relevant issue(s) if the resolved parameter passed is true.
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	title := "alert(gatus): " + ep.DisplayName()
	if !resolved {
		_, _, err := provider.githubClient.Issues.Create(context.Background(), provider.repositoryOwner, provider.repositoryName, &github.IssueRequest{
			Title: github.String(title),
			Body:  github.String(provider.buildIssueBody(ep, alert, result)),
		})
		if err != nil {
			return fmt.Errorf("failed to create issue: %w", err)
		}
	} else {
		issues, _, err := provider.githubClient.Issues.ListByRepo(context.Background(), provider.repositoryOwner, provider.repositoryName, &github.IssueListByRepoOptions{
			State:       "open",
			Creator:     provider.username,
			ListOptions: github.ListOptions{PerPage: 100},
		})
		if err != nil {
			return fmt.Errorf("failed to list issues: %w", err)
		}
		for _, issue := range issues {
			if *issue.Title == title {
				_, _, err = provider.githubClient.Issues.Edit(context.Background(), provider.repositoryOwner, provider.repositoryName, *issue.Number, &github.IssueRequest{
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

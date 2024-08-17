package gitea

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

// AlertProvider is the configuration necessary for sending an alert using Discord
type AlertProvider struct {
	RepositoryURL string `yaml:"repository-url"` // The URL of the Gitea repository to create issues in
	Token         string `yaml:"token"`          // Token requires at least RW on issues and RO on metadata

	// DefaultAlert is the default alert configuration to use for endpoints with an alert of the appropriate type
	DefaultAlert *alert.Alert `yaml:"default-alert,omitempty"`

	// ClientConfig is the configuration of the client used to communicate with the provider's target
	ClientConfig *client.Config `yaml:"client,omitempty"`

	// Assignees is a list of users to assign the issue to
	Assignees []string `yaml:"assignees,omitempty"`

	username        string
	repositoryOwner string
	repositoryName  string
	giteaClient     *gitea.Client
}

// IsValid returns whether the provider's configuration is valid
func (provider *AlertProvider) IsValid() bool {
	if provider.ClientConfig == nil {
		provider.ClientConfig = client.GetDefaultConfig()
	}

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

	opts := []gitea.ClientOption{
		gitea.SetToken(provider.Token),
	}

	if provider.ClientConfig != nil && provider.ClientConfig.Insecure {
		// add new http client for skip verify
		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		opts = append(opts, gitea.SetHTTPClient(httpClient))
	}

	provider.giteaClient, err = gitea.NewClient(baseURL, opts...)
	if err != nil {
		return false
	}

	user, _, err := provider.giteaClient.GetMyUserInfo()
	if err != nil {
		return false
	}

	provider.username = user.UserName

	return true
}

// Send creates an issue in the designed RepositoryURL if the resolved parameter passed is false,
// or closes the relevant issue(s) if the resolved parameter passed is true.
func (provider *AlertProvider) Send(ep *endpoint.Endpoint, alert *alert.Alert, result *endpoint.Result, resolved bool) error {
	title := "alert(gatus): " + ep.DisplayName()
	if !resolved {
		_, _, err := provider.giteaClient.CreateIssue(
			provider.repositoryOwner,
			provider.repositoryName,
			gitea.CreateIssueOption{
				Title:     title,
				Body:      provider.buildIssueBody(ep, alert, result),
				Assignees: provider.Assignees,
			},
		)
		if err != nil {
			return fmt.Errorf("failed to create issue: %w", err)
		}
		return nil
	}

	issues, _, err := provider.giteaClient.ListRepoIssues(
		provider.repositoryOwner,
		provider.repositoryName,
		gitea.ListIssueOption{
			State:     gitea.StateOpen,
			CreatedBy: provider.username,
			ListOptions: gitea.ListOptions{
				Page: 100,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	for _, issue := range issues {
		if issue.Title == title {
			stateClosed := gitea.StateClosed
			_, _, err = provider.giteaClient.EditIssue(
				provider.repositoryOwner,
				provider.repositoryName,
				issue.ID,
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

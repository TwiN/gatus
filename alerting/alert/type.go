package alert

// Type is the type of the alert.
// The value will generally be the name of the alert provider
type Type string

const (
	// TypeAWSSES is the Type for the awsses alerting provider
	TypeAWSSES Type = "aws-ses"

	// TypeCustom is the Type for the custom alerting provider
	TypeCustom Type = "custom"

	// TypeDiscord is the Type for the discord alerting provider
	TypeDiscord Type = "discord"

	// TypeEmail is the Type for the email alerting provider
	TypeEmail Type = "email"

	// TypeGitHub is the Type for the github alerting provider
	TypeGitHub Type = "github"

	// TypeGitLab is the Type for the gitlab alerting provider
	TypeGitLab Type = "gitlab"

	// TypeGitea is the Type for the gitea alerting provider
	TypeGitea Type = "gitea"

	// TypeGoogleChat is the Type for the googlechat alerting provider
	TypeGoogleChat Type = "googlechat"

	// TypeGotify is the Type for the gotify alerting provider
	TypeGotify Type = "gotify"

  // TypeHomeAssistant is the Type for the homeassistant alerting provider
	TypeHomeAssistant Type = "homeassistant"
  
	// TypeIlert is the Type for the ilert alerting provider
	TypeIlert Type = "ilert"

	// TypeIncidentIO is the Type for the incident-io alerting provider
	TypeIncidentIO Type = "incident-io"

	// TypeJetBrainsSpace is the Type for the jetbrains alerting provider
	TypeJetBrainsSpace Type = "jetbrainsspace"

	// TypeMatrix is the Type for the matrix alerting provider
	TypeMatrix Type = "matrix"

	// TypeMattermost is the Type for the mattermost alerting provider
	TypeMattermost Type = "mattermost"

	// TypeMessagebird is the Type for the messagebird alerting provider
	TypeMessagebird Type = "messagebird"

	// TypeNtfy is the Type for the ntfy alerting provider
	TypeNtfy Type = "ntfy"

	// TypeOpsgenie is the Type for the opsgenie alerting provider
	TypeOpsgenie Type = "opsgenie"

	// TypePagerDuty is the Type for the pagerduty alerting provider
	TypePagerDuty Type = "pagerduty"

	// TypePushover is the Type for the pushover alerting provider
	TypePushover Type = "pushover"

	// TypeSlack is the Type for the slack alerting provider
	TypeSlack Type = "slack"

	// TypeTeams is the Type for the teams alerting provider
	TypeTeams Type = "teams"

	// TypeTeamsWorkflows is the Type for the teams-workflows alerting provider
	TypeTeamsWorkflows Type = "teams-workflows"

	// TypeTelegram is the Type for the telegram alerting provider
	TypeTelegram Type = "telegram"

	// TypeTwilio is the Type for the twilio alerting provider
	TypeTwilio Type = "twilio"

	// TypeZulip is the Type for the Zulip alerting provider
	TypeZulip Type = "zulip"
)

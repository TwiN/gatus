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

	// TypeGoogleChat is the Type for the googlechat alerting provider
	TypeGoogleChat Type = "googlechat"

	// TypeGotify is the Type for the gotify alerting provider
	TypeGotify Type = "gotify"

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

	// TypeTelegram is the Type for the telegram alerting provider
	TypeTelegram Type = "telegram"

	// TypeTwilio is the Type for the twilio alerting provider
	TypeTwilio Type = "twilio"
)

package alert

// Type is the type of the alert.
// The value will generally be the name of the alert provider
type Type string

const (
	// TypeAWSSES is the Type for the awsses alerting provider
	TypeAWSSES Type = "aws-ses"

	// TypeClickUp is the Type for the clickup alerting provider
	TypeClickUp Type = "clickup"

	// TypeCustom is the Type for the custom alerting provider
	TypeCustom Type = "custom"

	// TypeDatadog is the Type for the datadog alerting provider
	TypeDatadog Type = "datadog"

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

	// TypeIFTTT is the Type for the ifttt alerting provider
	TypeIFTTT Type = "ifttt"

	// TypeIlert is the Type for the ilert alerting provider
	TypeIlert Type = "ilert"

	// TypeIncidentIO is the Type for the incident-io alerting provider
	TypeIncidentIO Type = "incident-io"

	// TypeLine is the Type for the line alerting provider
	TypeLine Type = "line"

	// TypeMatrix is the Type for the matrix alerting provider
	TypeMatrix Type = "matrix"

	// TypeMattermost is the Type for the mattermost alerting provider
	TypeMattermost Type = "mattermost"

	// TypeMessagebird is the Type for the messagebird alerting provider
	TypeMessagebird Type = "messagebird"

	// TypeNewRelic is the Type for the newrelic alerting provider
	TypeNewRelic Type = "newrelic"

	// TypeN8N is the Type for the n8n alerting provider
	TypeN8N Type = "n8n"

	// TypeNtfy is the Type for the ntfy alerting provider
	TypeNtfy Type = "ntfy"

	// TypeOpsgenie is the Type for the opsgenie alerting provider
	TypeOpsgenie Type = "opsgenie"

	// TypePagerDuty is the Type for the pagerduty alerting provider
	TypePagerDuty Type = "pagerduty"

	// TypePlivo is the Type for the plivo alerting provider
	TypePlivo Type = "plivo"

	// TypePushover is the Type for the pushover alerting provider
	TypePushover Type = "pushover"

	// TypeRocketChat is the Type for the rocketchat alerting provider
	TypeRocketChat Type = "rocketchat"

	// TypeSendGrid is the Type for the sendgrid alerting provider
	TypeSendGrid Type = "sendgrid"

	// TypeSignal is the Type for the signal alerting provider
	TypeSignal Type = "signal"

	// TypeSIGNL4 is the Type for the signl4 alerting provider
	TypeSIGNL4 Type = "signl4"

	// TypeSlack is the Type for the slack alerting provider
	TypeSlack Type = "slack"

	// TypeSplunk is the Type for the splunk alerting provider
	TypeSplunk Type = "splunk"

	// TypeSquadcast is the Type for the squadcast alerting provider
	TypeSquadcast Type = "squadcast"

	// TypeTeams is the Type for the teams alerting provider
	TypeTeams Type = "teams"

	// TypeTeamsWorkflows is the Type for the teams-workflows alerting provider
	TypeTeamsWorkflows Type = "teams-workflows"

	// TypeTelegram is the Type for the telegram alerting provider
	TypeTelegram Type = "telegram"

	// TypeTwilio is the Type for the twilio alerting provider
	TypeTwilio Type = "twilio"

	// TypeVonage is the Type for the vonage alerting provider
	TypeVonage Type = "vonage"

	// TypeWebex is the Type for the webex alerting provider
	TypeWebex Type = "webex"

	// TypeZapier is the Type for the zapier alerting provider
	TypeZapier Type = "zapier"

	// TypeZulip is the Type for the Zulip alerting provider
	TypeZulip Type = "zulip"
)

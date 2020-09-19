package alerting

type Config struct {
	Slack     *SlackAlertProvider     `yaml:"slack"`
	PagerDuty *PagerDutyAlertProvider `yaml:"pagerduty"`
	Twilio    *TwilioAlertProvider    `yaml:"twilio"`
	Custom    *CustomAlertProvider    `yaml:"custom"`
}

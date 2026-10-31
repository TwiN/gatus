package placeholder

import (
	"strings"

	"github.com/TwiN/gatus/v5/alerting/alert"
	"github.com/TwiN/gatus/v5/config/endpoint"
)

// Replace resolves the placeholders supported by alerting provider templates.
// Substituted values are not scanned again, so a value containing a placeholder is left as is.
func Replace(template string, ep *endpoint.Endpoint, alert *alert.Alert, resolved bool) string {
	alertState := "TRIGGERED"
	if resolved {
		alertState = "RESOLVED"
	}
	return strings.NewReplacer(
		"[ENDPOINT_NAME]", ep.Name,
		"[ENDPOINT_GROUP]", ep.Group,
		"[ALERT_DESCRIPTION]", alert.GetDescription(),
		"[ALERT_TRIGGERED_OR_RESOLVED]", alertState,
	).Replace(template)
}

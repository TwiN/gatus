package endpoint

import "net/url"

// NotificationURL returns a browser link suitable for built-in alert notifications.
// URLs with credentials, query parameters or fragments are omitted rather than
// partially redacted, since removing parts could change their destination.
func (endpoint *Endpoint) NotificationURL() string {
	if endpoint.UIConfig != nil && endpoint.UIConfig.HideURL {
		return ""
	}
	parsed, err := url.Parse(endpoint.URL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return ""
	}
	return parsed.String()
}

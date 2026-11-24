package watchdog

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// urlPattern finds a URL in a message, for the errors that carry one without being a *url.Error that can be found in the chain
var urlPattern = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://(?:<redacted>|[^\s"'<>])+`)

// redactError returns the message of err with the URL of any *url.Error in it reduced to its scheme and host.
//
// Alert providers put credentials in the request URL (a webhook URL, a bot token in the path, an access token in the query),
// and a transport failure comes back as a *url.Error whose message contains the whole URL. Redacting here, where the error
// of every provider is logged, means that no provider has to remember to do it.
func redactError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	for current := err; current != nil; current = errors.Unwrap(current) {
		var urlError *url.Error
		if errors.As(current, &urlError) && len(urlError.URL) > 0 {
			message = strings.ReplaceAll(message, urlError.URL, redactURL(urlError.URL))
			current = urlError
		}
	}
	// An error that holds a *url.Error without wrapping it (fmt.Errorf with %v) cannot be found above, but its message
	// still contains the URL
	return urlPattern.ReplaceAllStringFunc(message, func(match string) string {
		if strings.Contains(match, "/<redacted>") {
			return match
		}
		trimmed := strings.TrimRight(match, ".,;:)]}")
		return redactURL(trimmed) + match[len(trimmed):]
	})
}

// redactURL keeps the scheme and the host of a URL, which is enough to tell which service failed, and drops the rest:
// the user info, the path, the query and the fragment.
func redactURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil || len(parsed.Host) == 0 {
		return "<redacted>"
	}
	return parsed.Scheme + "://" + parsed.Host + "/<redacted>"
}

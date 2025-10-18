package security

import "github.com/TwiN/gocache/v2"

var sessions = gocache.NewCache().WithEvictionPolicy(gocache.LeastRecentlyUsed) // TODO: Move this to storage

// GetSessionSubject returns the user subject for a given session token
func GetSessionSubject(token string) (string, bool) {
	subject, exists := sessions.Get(token)
	if !exists {
		return "", false
	}
	if subjectStr, ok := subject.(string); ok {
		return subjectStr, true
	}
	return "", false
}

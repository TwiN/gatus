package security

import (
	"net/http"
	"strings"
)

// Handler takes care of security for a given handler with the given security configuration
func Handler(handler http.HandlerFunc, security *Config) http.HandlerFunc {
	if security == nil {
		return handler
	} else if security.Basic != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			usernameEntered, passwordEntered, ok := r.BasicAuth()
			if !ok || usernameEntered != security.Basic.Username || Sha512(passwordEntered) != strings.ToLower(security.Basic.PasswordSha512Hash) {
				w.Header().Set("WWW-Authenticate", "Basic")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte("Unauthorized"))
				return
			}
			handler(w, r)
		}
	} else if security.OIDC != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			// TODO: Check if the user is authenticated, and redirect to /login if they're not?
			handler(w, r)
		}
	}
	return handler
}

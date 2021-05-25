package security

import (
	"net/http"
	"strings"
)

// Handler takes care of security for a given handler with the given security configuration
func Handler(handler http.HandlerFunc, security *Config) http.HandlerFunc {
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
}

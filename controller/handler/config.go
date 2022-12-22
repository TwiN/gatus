package handler

import (
	"fmt"
	"net/http"

	"github.com/TwiN/gatus/v5/security"
)

// ConfigHandler is a handler that returns information for the front end of the application.
type ConfigHandler struct {
	securityConfig *security.Config
}

func (handler ConfigHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hasOIDC := false
	isAuthenticated := true // Default to true if no security config is set
	if handler.securityConfig != nil {
		hasOIDC = handler.securityConfig.OIDC != nil
		isAuthenticated = handler.securityConfig.IsAuthenticated(r)
	}
	// Return the config
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(fmt.Sprintf(`{"oidc":%v,"authenticated":%v}`, hasOIDC, isAuthenticated)))
}

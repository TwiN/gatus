package security

import (
	"net/http"
	"strings"

	"github.com/TwiN/g8"
	"github.com/gorilla/mux"
)

const (
	cookieNameState   = "gatus_state"
	cookieNameNonce   = "gatus_nonce"
	cookieNameSession = "gatus_session"
)

// Config is the security configuration for Gatus
type Config struct {
	Basic *BasicConfig `yaml:"basic,omitempty"`
	OIDC  *OIDCConfig  `yaml:"oidc,omitempty"`
}

// IsValid returns whether the security configuration is valid or not
func (c *Config) IsValid() bool {
	return (c.Basic != nil && c.Basic.isValid()) || (c.OIDC != nil && c.OIDC.isValid())
}

// RegisterHandlers registers all handlers required based on the security configuration
func (c *Config) RegisterHandlers(router *mux.Router) error {
	if c.OIDC != nil {
		if err := c.OIDC.initialize(); err != nil {
			return err
		}
		router.HandleFunc("/oidc/login", c.OIDC.loginHandler)
		router.HandleFunc("/authorization-code/callback", c.OIDC.callbackHandler)
	}
	return nil
}

func (c *Config) ApplySecurityMiddleware(api *mux.Router) {
	if c.OIDC != nil {
		// We're going to use g8 for session handling
		clientProvider := g8.NewClientProvider(func(token string) *g8.Client {
			if _, exists := sessions.Get(token); exists {
				return g8.NewClient(token)
			}
			return nil
		})
		customTokenExtractorFunc := func(request *http.Request) string {
			sessionCookie, err := request.Cookie(cookieNameSession)
			if err != nil {
				return ""
			}
			return sessionCookie.Value
		}
		// TODO: g8: Add a way to update cookie after? would need the writer
		authorizationService := g8.NewAuthorizationService().WithClientProvider(clientProvider)
		gate := g8.New().WithAuthorizationService(authorizationService).WithCustomTokenExtractor(customTokenExtractorFunc)
		api.Use(gate.Protect)
	} else if c.Basic != nil {
		api.Use(func(handler http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				usernameEntered, passwordEntered, ok := r.BasicAuth()
				if !ok || usernameEntered != c.Basic.Username || Sha512(passwordEntered) != strings.ToLower(c.Basic.PasswordSha512Hash) {
					w.Header().Set("WWW-Authenticate", "Basic")
					w.WriteHeader(http.StatusUnauthorized)
					_, _ = w.Write([]byte("Unauthorized"))
					return
				}
				handler.ServeHTTP(w, r)
			})
		})
	}
}

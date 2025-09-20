package security

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/TwiN/logr"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

const (
	DefaultOIDCSessionTTL = 8 * time.Hour
)

// OIDCConfig is the configuration for OIDC authentication
type OIDCConfig struct {
	IssuerURL       string        `yaml:"issuer-url"`   // e.g. https://dev-12345678.okta.com
	RedirectURL     string        `yaml:"redirect-url"` // e.g. http://localhost:8080/authorization-code/callback
	ClientID        string        `yaml:"client-id"`
	ClientSecret    string        `yaml:"client-secret"`
	Scopes          []string      `yaml:"scopes"`           // e.g. ["openid"]
	AllowedSubjects []string      `yaml:"allowed-subjects"` // e.g. ["user1@example.com"]. If empty, all subjects are allowed
	SessionTTL      time.Duration `yaml:"session-ttl"`      // e.g. 8h. Defaults to 8 hours

	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

// ValidateAndSetDefaults returns whether the OIDC configuration is valid and sets default values.
func (c *OIDCConfig) ValidateAndSetDefaults() bool {
	if c.SessionTTL <= 0 {
		c.SessionTTL = DefaultOIDCSessionTTL
	}
	return len(c.IssuerURL) > 0 && len(c.RedirectURL) > 0 && strings.HasSuffix(c.RedirectURL, "/authorization-code/callback") && len(c.ClientID) > 0 && len(c.ClientSecret) > 0 && len(c.Scopes) > 0
}

func (c *OIDCConfig) initialize() error {
	provider, err := oidc.NewProvider(context.Background(), c.IssuerURL)
	if err != nil {
		return err
	}
	c.verifier = provider.Verifier(&oidc.Config{ClientID: c.ClientID})
	// Configure an OpenID Connect aware OAuth2 client.
	c.oauth2Config = oauth2.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Scopes:       c.Scopes,
		RedirectURL:  c.RedirectURL,
		Endpoint:     provider.Endpoint(),
	}
	return nil
}

func (c *OIDCConfig) loginHandler(ctx *fiber.Ctx) error {
	state, nonce := uuid.NewString(), uuid.NewString()
	ctx.Cookie(&fiber.Cookie{
		Name:     cookieNameState,
		Value:    state,
		Path:     "/",
		MaxAge:   int(time.Hour.Seconds()),
		SameSite: "lax",
		HTTPOnly: true,
	})
	ctx.Cookie(&fiber.Cookie{
		Name:     cookieNameNonce,
		Value:    nonce,
		Path:     "/",
		MaxAge:   int(time.Hour.Seconds()),
		SameSite: "lax",
		HTTPOnly: true,
	})
	return ctx.Redirect(c.oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
}

func (c *OIDCConfig) callbackHandler(w http.ResponseWriter, r *http.Request) { // TODO: Migrate to a native fiber handler
	// Check if there's an error
	if len(r.URL.Query().Get("error")) > 0 {
		http.Error(w, r.URL.Query().Get("error")+": "+r.URL.Query().Get("error_description"), http.StatusBadRequest)
		return
	}
	// Ensure that the state has the expected value
	state, err := r.Cookie(cookieNameState)
	if err != nil {
		http.Error(w, "state not found", http.StatusBadRequest)
		return
	}
	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, "state did not match", http.StatusBadRequest)
		return
	}
	// Validate token
	oauth2Token, err := c.oauth2Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "Error exchanging token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		http.Error(w, "Missing 'id_token' in oauth2 token", http.StatusInternalServerError)
		return
	}
	idToken, err := c.verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		http.Error(w, "Failed to verify id_token: "+err.Error(), http.StatusInternalServerError)
		return
	}
	// Validate nonce
	nonce, err := r.Cookie(cookieNameNonce)
	if err != nil {
		http.Error(w, "nonce not found", http.StatusBadRequest)
		return
	}
	if idToken.Nonce != nonce.Value {
		http.Error(w, "nonce did not match", http.StatusBadRequest)
		return
	}
	if len(c.AllowedSubjects) == 0 {
		// If there's no allowed subjects, all subjects are allowed.
		c.setSessionCookie(w, idToken)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	for _, subject := range c.AllowedSubjects {
		if strings.ToLower(subject) == strings.ToLower(idToken.Subject) {
			c.setSessionCookie(w, idToken)
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}
	logr.Debugf("[security.callbackHandler] Subject %s is not in the list of allowed subjects", idToken.Subject)
	http.Redirect(w, r, "/?error=access_denied", http.StatusFound)
}

func (c *OIDCConfig) setSessionCookie(w http.ResponseWriter, idToken *oidc.IDToken) {
	// At this point, the user has been confirmed. All that's left to do is create a session.
	sessionID := uuid.NewString()
	sessions.SetWithTTL(sessionID, idToken.Subject, c.SessionTTL)
	http.SetCookie(w, &http.Cookie{
		Name:     cookieNameSession,
		Value:    sessionID,
		Path:     "/",
		MaxAge:   int(c.SessionTTL.Seconds()),
		SameSite: http.SameSiteStrictMode,
	})
}

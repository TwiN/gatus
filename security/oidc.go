package security

import (
	"context"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// OIDCConfig is the configuration for OIDC authentication
type OIDCConfig struct {
	IssuerURL    string   `yaml:"issuer-url"`   // e.g. https://dev-12345678.okta.com
	RedirectURL  string   `yaml:"redirect-url"` // e.g. http://localhost:8080/authorization-code/callback
	ClientID     string   `yaml:"client-id"`
	ClientSecret string   `yaml:"client-secret"`
	Scopes       []string `yaml:"scopes"` // e.g. [openid]

	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
}

// isValid returns whether the basic security configuration is valid or not
func (c *OIDCConfig) isValid() bool {
	return len(c.IssuerURL) > 0 && len(c.RedirectURL) > 0 && len(c.ClientID) > 0 && len(c.ClientSecret) > 0 && len(c.Scopes) > 0
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

func (c *OIDCConfig) loginHandler(w http.ResponseWriter, r *http.Request) {
	state, nonce := uuid.NewString(), uuid.NewString()
	http.SetCookie(w, &http.Cookie{
		Name:     "state",
		Value:    state,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "nonce",
		Value:    nonce,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
	})
	http.Redirect(w, r, c.oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
}

func (c *OIDCConfig) callbackHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure that the state has the expected value
	state, err := r.Cookie("state")
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
	nonce, err := r.Cookie("nonce")
	if err != nil {
		http.Error(w, "nonce not found", http.StatusBadRequest)
		return
	}
	if idToken.Nonce != nonce.Value {
		http.Error(w, "nonce did not match", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TwiN/gatus/v5/security"
	"github.com/gorilla/mux"
)

func TestConfigHandler_ServeHTTP(t *testing.T) {
	securityConfig := &security.Config{
		OIDC: &security.OIDCConfig{
			IssuerURL:       "https://sso.gatus.io/",
			RedirectURL:     "http://localhost:80/authorization-code/callback",
			Scopes:          []string{"openid"},
			AllowedSubjects: []string{"user1@example.com"},
		},
	}
	handler := ConfigHandler{securityConfig: securityConfig}
	// Create a fake router. We're doing this because I need the gate to be initialized.
	securityConfig.ApplySecurityMiddleware(mux.NewRouter())
	// Test the config handler
	request, _ := http.NewRequest("GET", "/api/v1/config", http.NoBody)
	responseRecorder := httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, request)
	if responseRecorder.Code != http.StatusOK {
		t.Error("expected code to be 200, but was", responseRecorder.Code)
	}
	if responseRecorder.Body.String() != `{"oidc":true,"authenticated":false}` {
		t.Error("expected body to be `{\"oidc\":true,\"authenticated\":false}`, but was", responseRecorder.Body.String())
	}
}

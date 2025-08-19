package api

import (
	"io"
	"net/http"
	"testing"

	"github.com/TwiN/gatus/v5/security"
	"github.com/gofiber/fiber/v2"
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
	app := fiber.New()
	app.Get("/api/v1/config", handler.GetConfig)
	err := securityConfig.ApplySecurityMiddleware(app)
	if err != nil {
		t.Error("expected err to be nil, but was", err)
	}
	// Test the config handler
	request, _ := http.NewRequest("GET", "/api/v1/config", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Error("expected err to be nil, but was", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Error("expected code to be 200, but was", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Error("expected err to be nil, but was", err)
	}
	if string(body) != `{"announcements":[],"authenticated":false,"oidc":true}` {
		t.Error("expected body to be `{\"announcements\":[],\"authenticated\":false,\"oidc\":true}`, but was", string(body))
	}
}

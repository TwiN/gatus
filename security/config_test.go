package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
)

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	c := &Config{
		Basic: nil,
		OIDC:  nil,
	}
	if c.ValidateAndSetDefaults() {
		t.Error("expected empty config to be valid")
	}
}

func TestConfig_ApplySecurityMiddleware(t *testing.T) {
	///////////
	// BASIC //
	///////////
	t.Run("basic", func(t *testing.T) {
		// Bcrypt
		c := &Config{Basic: &BasicConfig{
			Username:                        "john.doe",
			PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
		}}
		app := fiber.New()
		if err := c.ApplySecurityMiddleware(app); err != nil {
			t.Error("expected no error, got", err)
		}
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendStatus(200)
		})
		// Try to access the route without basic auth
		request := httptest.NewRequest("GET", "/test", http.NoBody)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if response.StatusCode != 401 {
			t.Error("expected code to be 401, but was", response.StatusCode)
		}
		// Try again, but with basic auth
		request = httptest.NewRequest("GET", "/test", http.NoBody)
		request.SetBasicAuth("john.doe", "hunter2")
		response, err = app.Test(request)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if response.StatusCode != 200 {
			t.Error("expected code to be 200, but was", response.StatusCode)
		}
	})
	//////////
	// OIDC //
	//////////
	t.Run("oidc", func(t *testing.T) {
		c := &Config{OIDC: &OIDCConfig{
			IssuerURL:       "https://sso.gatus.io/",
			RedirectURL:     "http://localhost:80/authorization-code/callback",
			Scopes:          []string{"openid"},
			AllowedSubjects: []string{"user1@example.com"},
			SessionTTL:      DefaultOIDCSessionTTL,
			oauth2Config:    oauth2.Config{},
			verifier:        nil,
		}}
		app := fiber.New()
		if err := c.ApplySecurityMiddleware(app); err != nil {
			t.Error("expected no error, got", err)
		}
		app.Get("/test", func(c *fiber.Ctx) error {
			return c.SendStatus(200)
		})
		// Try without any session cookie
		request := httptest.NewRequest("GET", "/test", http.NoBody)
		response, err := app.Test(request)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if response.StatusCode != 401 {
			t.Error("expected code to be 401, but was", response.StatusCode)
		}
		// Try with a session cookie
		request = httptest.NewRequest("GET", "/test", http.NoBody)
		request.AddCookie(&http.Cookie{Name: "session", Value: "123"})
		response, err = app.Test(request)
		if err != nil {
			t.Fatal("expected no error, got", err)
		}
		if response.StatusCode != 401 {
			t.Error("expected code to be 401, but was", response.StatusCode)
		}
	})
}

func TestConfig_RegisterHandlers(t *testing.T) {
	c := &Config{}
	app := fiber.New()
	c.RegisterHandlers(app)
	// Try to access the OIDC handler. This should fail, because the security config doesn't have OIDC
	request := httptest.NewRequest("GET", "/oidc/login", http.NoBody)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if response.StatusCode != 404 {
		t.Error("expected code to be 404, but was", response.StatusCode)
	}
	// Set an empty OIDC config. This should fail, because the IssuerURL is required.
	c.OIDC = &OIDCConfig{}
	if err := c.RegisterHandlers(app); err == nil {
		t.Fatal("expected an error, but got none")
	}
	// Set the OIDC config and try again
	c.OIDC = &OIDCConfig{
		IssuerURL:       "https://sso.gatus.io/",
		RedirectURL:     "http://localhost:80/authorization-code/callback",
		Scopes:          []string{"openid"},
		AllowedSubjects: []string{"user1@example.com"},
	}
	if err := c.RegisterHandlers(app); err != nil {
		t.Fatal("expected no error, but got", err)
	}
	request = httptest.NewRequest("GET", "/oidc/login", http.NoBody)
	response, err = app.Test(request)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	if response.StatusCode != 302 {
		t.Error("expected code to be 302, but was", response.StatusCode)
	}
}

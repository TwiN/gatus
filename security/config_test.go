package security

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
)

func TestConfig_ValidateAndSetDefaults(t *testing.T) {
	validBasicConfig := &BasicConfig{
		Username:                        "test",
		PasswordBcryptHashBase64Encoded: "somevalue",
	}
	validOIDCConfig := &OIDCConfig{
		IssuerURL:    "testurl",
		RedirectURL:  "testredirecturl/authorization-code/callback",
		ClientID:     "testid",
		ClientSecret: "testsecret",
		Scopes:       []string{"testscope"},
	}

	type Scenario struct {
		Name        string
		Config      *Config
		ExpectValid bool
	}
	scenarios := []Scenario{
		{
			Name: "empty",
			Config: &Config{
				Basic: nil,
				OIDC:  nil,
			},
			ExpectValid: true,
		},
		{
			Name: "empty-basic",
			Config: &Config{
				Basic: &BasicConfig{},
				OIDC:  nil,
			},
			ExpectValid: false,
		},
		{
			Name: "empty-oidc",
			Config: &Config{
				Basic: nil,
				OIDC:  &OIDCConfig{},
			},
			ExpectValid: false,
		},
		{
			Name: "valid-basic-only",
			Config: &Config{
				Basic: validBasicConfig,
				OIDC:  nil,
			},
			ExpectValid: true,
		},
		{
			Name: "valid-oidc-only",
			Config: &Config{
				Basic: nil,
				OIDC:  validOIDCConfig,
			},
			ExpectValid: true,
		},
		{
			Name: "valid-basic-and-oidc",
			Config: &Config{
				Basic: validBasicConfig,
				OIDC:  validOIDCConfig,
			},
			ExpectValid: true,
		},
		{
			Name: "valid-basic-with-api-tokens",
			Config: &Config{
				Basic: &BasicConfig{
					Username:                        "test",
					PasswordBcryptHashBase64Encoded: "somevalue",
					APITokens:                       []string{"token"},
				},
			},
			ExpectValid: true,
		},
		{
			Name: "basic-with-empty-api-token",
			Config: &Config{
				Basic: &BasicConfig{
					Username:                        "test",
					PasswordBcryptHashBase64Encoded: "somevalue",
					APITokens:                       []string{"token", ""},
				},
			},
			ExpectValid: false,
		},
		{
			Name: "valid-oidc-with-api-tokens",
			Config: &Config{
				OIDC: &OIDCConfig{
					IssuerURL:    "testurl",
					RedirectURL:  "testredirecturl/authorization-code/callback",
					ClientID:     "testid",
					ClientSecret: "testsecret",
					Scopes:       []string{"testscope"},
					APITokens:    []string{"token"},
				},
			},
			ExpectValid: true,
		},
		{
			Name: "oidc-with-empty-api-token",
			Config: &Config{
				OIDC: &OIDCConfig{
					IssuerURL:    "testurl",
					RedirectURL:  "testredirecturl/authorization-code/callback",
					ClientID:     "testid",
					ClientSecret: "testsecret",
					Scopes:       []string{"testscope"},
					APITokens:    []string{""},
				},
			},
			ExpectValid: false,
		},
		{
			Name: "valid-metrics-protected-with-basic",
			Config: &Config{
				Basic:   validBasicConfig,
				Metrics: &MetricsConfig{Protected: true},
			},
			ExpectValid: true,
		},
		{
			Name: "valid-metrics-tokens-only",
			Config: &Config{
				Metrics: &MetricsConfig{Tokens: []string{"token"}},
			},
			ExpectValid: true,
		},
		{
			Name: "metrics-protected-without-provider-or-tokens",
			Config: &Config{
				Metrics: &MetricsConfig{Protected: true},
			},
			ExpectValid: false,
		},
		{
			Name: "metrics-with-empty-token",
			Config: &Config{
				Basic:   validBasicConfig,
				Metrics: &MetricsConfig{Tokens: []string{""}},
			},
			ExpectValid: false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			isValid := scenario.Config.ValidateAndSetDefaults()
			if isValid && !scenario.ExpectValid {
				t.Errorf("scenario %s: expected config to be invalid", scenario.Name)
			} else if !isValid && scenario.ExpectValid {
				t.Errorf("scenario %s: expected config to be valid", scenario.Name)
			}
		})
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

func TestConfig_ApplySecurityMiddlewareWithAPITokens(t *testing.T) {
	basicConfig := &BasicConfig{
		Username:                        "john.doe",
		PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
		APITokens:                       []string{"basic-api-token"},
	}
	oidcConfig := &OIDCConfig{
		IssuerURL:   "https://sso.gatus.io/",
		RedirectURL: "http://localhost:80/authorization-code/callback",
		Scopes:      []string{"openid"},
		APITokens:   []string{"oidc-api-token"},
	}
	type Scenario struct {
		Name          string
		Config        *Config
		Authorization string
		WithBasicAuth bool
		ExpectedCode  int
	}
	scenarios := []Scenario{
		{
			Name:         "basic-without-auth",
			Config:       &Config{Basic: basicConfig},
			ExpectedCode: 401,
		},
		{
			Name:          "basic-with-basic-auth",
			Config:        &Config{Basic: basicConfig},
			WithBasicAuth: true,
			ExpectedCode:  200,
		},
		{
			Name:          "basic-with-api-token",
			Config:        &Config{Basic: basicConfig},
			Authorization: "Bearer basic-api-token",
			ExpectedCode:  200,
		},
		{
			Name:          "basic-with-invalid-api-token",
			Config:        &Config{Basic: basicConfig},
			Authorization: "Bearer invalid-token",
			ExpectedCode:  401,
		},
		{
			Name:          "basic-with-metrics-token",
			Config:        &Config{Basic: basicConfig, Metrics: &MetricsConfig{Tokens: []string{"metrics-token"}}},
			Authorization: "Bearer metrics-token",
			ExpectedCode:  401,
		},
		{
			Name:         "oidc-without-auth",
			Config:       &Config{OIDC: oidcConfig},
			ExpectedCode: 401,
		},
		{
			Name:          "oidc-with-api-token",
			Config:        &Config{OIDC: oidcConfig},
			Authorization: "Bearer oidc-api-token",
			ExpectedCode:  200,
		},
		{
			Name:          "basic-and-oidc-with-basic-api-token",
			Config:        &Config{Basic: basicConfig, OIDC: oidcConfig},
			Authorization: "Bearer basic-api-token",
			ExpectedCode:  200,
		},
		{
			Name:          "basic-and-oidc-with-oidc-api-token",
			Config:        &Config{Basic: basicConfig, OIDC: oidcConfig},
			Authorization: "Bearer oidc-api-token",
			ExpectedCode:  200,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			app := fiber.New()
			if err := scenario.Config.ApplySecurityMiddleware(app); err != nil {
				t.Fatal("expected no error, got", err)
			}
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendStatus(200)
			})
			if code := testRequest(t, app, scenario.Authorization, scenario.WithBasicAuth); code != scenario.ExpectedCode {
				t.Errorf("expected code to be %d, but was %d", scenario.ExpectedCode, code)
			}
		})
	}
}

func TestConfig_ApplyMetricsSecurityMiddleware(t *testing.T) {
	basicConfig := &BasicConfig{
		Username:                        "john.doe",
		PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
		APITokens:                       []string{"api-token"},
	}
	oidcConfig := &OIDCConfig{
		IssuerURL:   "https://sso.gatus.io/",
		RedirectURL: "http://localhost:80/authorization-code/callback",
		Scopes:      []string{"openid"},
	}
	type Scenario struct {
		Name          string
		Config        *Config
		Authorization string
		WithBasicAuth bool
		ExpectedCode  int
	}
	scenarios := []Scenario{
		{
			Name:         "unprotected",
			Config:       &Config{Basic: basicConfig},
			ExpectedCode: 200,
		},
		{
			Name:         "unprotected-with-empty-metrics-config",
			Config:       &Config{Basic: basicConfig, Metrics: &MetricsConfig{}},
			ExpectedCode: 200,
		},
		{
			Name:         "tokens-only-without-auth",
			Config:       &Config{Metrics: &MetricsConfig{Tokens: []string{"metrics-token"}}},
			ExpectedCode: 401,
		},
		{
			Name:          "tokens-only-with-token",
			Config:        &Config{Metrics: &MetricsConfig{Tokens: []string{"metrics-token"}}},
			Authorization: "Bearer metrics-token",
			ExpectedCode:  200,
		},
		{
			Name:          "tokens-only-with-invalid-token",
			Config:        &Config{Metrics: &MetricsConfig{Tokens: []string{"metrics-token"}}},
			Authorization: "Bearer invalid-token",
			ExpectedCode:  401,
		},
		{
			Name:         "protected-with-basic-without-auth",
			Config:       &Config{Basic: basicConfig, Metrics: &MetricsConfig{Protected: true}},
			ExpectedCode: 401,
		},
		{
			Name:          "protected-with-basic-with-basic-auth",
			Config:        &Config{Basic: basicConfig, Metrics: &MetricsConfig{Protected: true}},
			WithBasicAuth: true,
			ExpectedCode:  200,
		},
		{
			Name:          "protected-with-basic-with-api-token",
			Config:        &Config{Basic: basicConfig, Metrics: &MetricsConfig{Protected: true}},
			Authorization: "Bearer api-token",
			ExpectedCode:  401,
		},
		{
			Name:          "tokens-with-basic-with-basic-auth",
			Config:        &Config{Basic: basicConfig, Metrics: &MetricsConfig{Tokens: []string{"metrics-token"}}},
			WithBasicAuth: true,
			ExpectedCode:  200,
		},
		{
			Name:          "tokens-with-basic-with-token",
			Config:        &Config{Basic: basicConfig, Metrics: &MetricsConfig{Tokens: []string{"metrics-token"}}},
			Authorization: "Bearer metrics-token",
			ExpectedCode:  200,
		},
		{
			Name:         "protected-with-oidc-without-session",
			Config:       &Config{OIDC: oidcConfig, Metrics: &MetricsConfig{Protected: true}},
			ExpectedCode: 401,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			app := fiber.New()
			if err := scenario.Config.ApplyMetricsSecurityMiddleware(app); err != nil {
				t.Fatal("expected no error, got", err)
			}
			app.Get("/test", func(c *fiber.Ctx) error {
				return c.SendStatus(200)
			})
			if code := testRequest(t, app, scenario.Authorization, scenario.WithBasicAuth); code != scenario.ExpectedCode {
				t.Errorf("expected code to be %d, but was %d", scenario.ExpectedCode, code)
			}
		})
	}
}

func TestConfig_IsAuthenticated(t *testing.T) {
	c := &Config{
		Basic: &BasicConfig{
			Username:                        "john.doe",
			PasswordBcryptHashBase64Encoded: "JDJhJDA4JDFoRnpPY1hnaFl1OC9ISlFsa21VS09wOGlPU1ZOTDlHZG1qeTFvb3dIckRBUnlHUmNIRWlT",
			APITokens:                       []string{"basic-api-token"},
		},
		OIDC: &OIDCConfig{
			IssuerURL:   "https://sso.gatus.io/",
			RedirectURL: "http://localhost:80/authorization-code/callback",
			Scopes:      []string{"openid"},
			APITokens:   []string{"oidc-api-token"},
		},
		Metrics: &MetricsConfig{Tokens: []string{"metrics-token"}},
	}
	app := fiber.New()
	app.Get("/test", func(ctx *fiber.Ctx) error {
		if c.IsAuthenticated(ctx) {
			return ctx.SendStatus(200)
		}
		return ctx.SendStatus(401)
	})
	type Scenario struct {
		Name          string
		Authorization string
		Expected      bool
	}
	scenarios := []Scenario{
		{
			Name:          "without-auth",
			Authorization: "",
			Expected:      false,
		},
		{
			Name:          "with-basic-api-token",
			Authorization: "Bearer basic-api-token",
			Expected:      true,
		},
		{
			Name:          "with-oidc-api-token",
			Authorization: "Bearer oidc-api-token",
			Expected:      true,
		},
		{
			Name:          "with-invalid-api-token",
			Authorization: "Bearer invalid-token",
			Expected:      false,
		},
		{
			Name:          "with-metrics-token",
			Authorization: "Bearer metrics-token",
			Expected:      false,
		},
	}
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			if isAuthenticated := testRequest(t, app, scenario.Authorization, false) == 200; isAuthenticated != scenario.Expected {
				t.Errorf("expected %v, got %v", scenario.Expected, isAuthenticated)
			}
		})
	}
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

func testRequest(t *testing.T, app *fiber.App, authorization string, withBasicAuth bool) int {
	t.Helper()
	request := httptest.NewRequest("GET", "/test", http.NoBody)
	if withBasicAuth {
		request.SetBasicAuth("john.doe", "hunter2")
	} else if len(authorization) > 0 {
		request.Header.Set("Authorization", authorization)
	}
	response, err := app.Test(request)
	if err != nil {
		t.Fatal("expected no error, got", err)
	}
	return response.StatusCode
}

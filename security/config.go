package security

import (
	"encoding/base64"
	"net/http"

	g8 "github.com/TwiN/g8/v2"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"golang.org/x/crypto/bcrypt"
)

const (
	cookieNameState   = "gatus_state"
	cookieNameNonce   = "gatus_nonce"
	cookieNameSession = "gatus_session"
)

// Config is the security configuration for Gatus
type Config struct {
	Basic   *BasicConfig   `yaml:"basic,omitempty"`
	OIDC    *OIDCConfig    `yaml:"oidc,omitempty"`
	Metrics *MetricsConfig `yaml:"metrics,omitempty"`

	gate *g8.Gate
}

// ValidateAndSetDefaults returns whether the security configuration is valid or not and sets default values.
func (c *Config) ValidateAndSetDefaults() bool {
	return (c.Basic == nil || c.Basic.isValid()) && (c.OIDC == nil || c.OIDC.ValidateAndSetDefaults()) && (c.Metrics == nil || c.Metrics.isValid(c.Basic != nil || c.OIDC != nil))
}

// RegisterHandlers registers all handlers required based on the security configuration
func (c *Config) RegisterHandlers(router fiber.Router) error {
	if c.OIDC != nil {
		if err := c.OIDC.initialize(); err != nil {
			return err
		}
		router.All("/oidc/login", c.OIDC.loginHandler)
		router.All("/authorization-code/callback", adaptor.HTTPHandlerFunc(c.OIDC.callbackHandler))
	}
	return nil
}

// ApplySecurityMiddleware applies an authentication middleware to the router passed.
// The router passed should be a sub-router in charge of handlers that require authentication.
func (c *Config) ApplySecurityMiddleware(router fiber.Router) error {
	return c.applyMiddleware(router, c.apiTokens())
}

// ApplyMetricsSecurityMiddleware applies an authentication middleware to the router passed if metrics are protected.
// The router passed should be a sub-router in charge of the metrics handler.
func (c *Config) ApplyMetricsSecurityMiddleware(router fiber.Router) error {
	if c.Metrics == nil || !c.Metrics.isProtected() {
		return nil
	}
	return c.applyMiddleware(router, c.Metrics.Tokens)
}

// applyMiddleware applies a middleware accepting any of the bearer tokens passed, falling back to Basic or OIDC auth
func (c *Config) applyMiddleware(router fiber.Router, tokens []string) error {
	providerMiddleware, err := c.newProviderMiddleware()
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		if providerMiddleware != nil {
			router.Use(providerMiddleware)
		}
		return nil
	}
	router.Use(func(ctx *fiber.Ctx) error {
		if hasValidBearerToken(ctx, tokens) {
			return ctx.Next()
		}
		if providerMiddleware == nil {
			return ctx.Status(401).SendString("Unauthorized")
		}
		return providerMiddleware(ctx)
	})
	return nil
}

// newProviderMiddleware returns the OIDC or Basic authentication middleware, or nil if neither is configured
func (c *Config) newProviderMiddleware() (fiber.Handler, error) {
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
		c.gate = g8.New().WithAuthorizationService(authorizationService).WithCustomTokenExtractor(customTokenExtractorFunc)
		return adaptor.HTTPMiddleware(c.gate.Protect), nil
	} else if c.Basic != nil {
		var decodedBcryptHash []byte
		if len(c.Basic.PasswordBcryptHashBase64Encoded) > 0 {
			var err error
			decodedBcryptHash, err = base64.URLEncoding.DecodeString(c.Basic.PasswordBcryptHashBase64Encoded)
			if err != nil {
				return nil, err
			}
		}
		return basicauth.New(basicauth.Config{
			Authorizer: func(username, password string) bool {
				if len(c.Basic.PasswordBcryptHashBase64Encoded) > 0 {
					if username != c.Basic.Username || bcrypt.CompareHashAndPassword(decodedBcryptHash, []byte(password)) != nil {
						return false
					}
				}
				return true
			},
			Unauthorized: func(ctx *fiber.Ctx) error {
				ctx.Set("WWW-Authenticate", "Basic")
				return ctx.Status(401).SendString("Unauthorized")
			},
		}), nil
	}
	return nil, nil
}

// apiTokens returns the API tokens of all configured authentication methods
func (c *Config) apiTokens() []string {
	var tokens []string
	if c.Basic != nil {
		tokens = append(tokens, c.Basic.APITokens...)
	}
	if c.OIDC != nil {
		tokens = append(tokens, c.OIDC.APITokens...)
	}
	return tokens
}

// IsAuthenticated checks whether the user is authenticated
// If the Config does not warrant authentication, it will always return true.
func (c *Config) IsAuthenticated(ctx *fiber.Ctx) bool {
	if hasValidBearerToken(ctx, c.apiTokens()) {
		return true
	}
	if c.gate != nil {
		// TODO: Update g8 to support fasthttp natively? (see g8's fasthttp branch)
		request, err := adaptor.ConvertRequest(ctx, false)
		if err != nil {
			logr.Errorf("[security.IsAuthenticated] Unexpected error converting request: %v", err)
			return false
		}
		token := c.gate.ExtractTokenFromRequest(request)
		_, hasSession := sessions.Get(token)
		return hasSession
	}
	return false
}

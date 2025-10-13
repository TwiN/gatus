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

const (
	authLevelGlobal = "global"
	authLevelEndpoint = "endpoint"
)

// Config is the security configuration for Gatus
type Config struct {
	Basic *BasicConfig `yaml:"basic,omitempty"`
	OIDC  *OIDCConfig  `yaml:"oidc,omitempty"`      
	Level string  `yaml:"level,omitempty"`
	
	gate *g8.Gate
}

// ValidateAndSetDefaults returns whether the security configuration is valid or not and sets default values.
func (c *Config) ValidateAndSetDefaults() bool {
	basicValid := c.Basic != nil && c.Basic.isValid()
	oauthValid := c.OIDC != nil && c.OIDC.ValidateAndSetDefaults()
	
	// Set default level
	if c.Level == "" {
		c.Level = authLevelGlobal
	}
	levelValid := c.Level == authLevelGlobal || c.Level == authLevelEndpoint

	return levelValid && (basicValid || oauthValid)
}

// IsGlobal tells wether the auth is global or at endpoint level
func (c *Config) IsGlobal() bool {
	return c.Level == authLevelGlobal
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
		router.Use(adaptor.HTTPMiddleware(c.gate.Protect))
	} else if c.Basic != nil {
		var decodedBcryptHash []byte
		if len(c.Basic.PasswordBcryptHashBase64Encoded) > 0 {
			var err error
			decodedBcryptHash, err = base64.URLEncoding.DecodeString(c.Basic.PasswordBcryptHashBase64Encoded)
			if err != nil {
				return err
			}
		}
		router.Use(basicauth.New(basicauth.Config{
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
		}))
	}
	return nil
}

// IsAuthenticated checks whether the user is authenticated
// If the Config does not warrant authentication, it will always return true.
func (c *Config) IsAuthenticated(ctx *fiber.Ctx) bool {
	if c.gate != nil {
		// OIDC authentication check
		request, err := adaptor.ConvertRequest(ctx, false)
		if err != nil {
			logr.Errorf("[security.IsAuthenticated] Unexpected error converting request: %v", err)
			return false
		}
		token := c.gate.ExtractTokenFromRequest(request)
		_, hasSession := sessions.Get(token)
		return hasSession
	} else if c.Basic != nil {
		// Basic Auth authentication check
		authHeader := ctx.Get("Authorization")
		if authHeader == "" {
			return false
		}
		// Parse the Basic Auth header manually (fasthttp doesn't have BasicAuth() helper)
		// Format: "Basic base64(username:password)"
		const prefix = "Basic "
		if len(authHeader) < len(prefix) {
			return false
		}
		if authHeader[:len(prefix)] != prefix {
			return false
		}
		// Decode the base64 credentials
		decoded, err := base64.StdEncoding.DecodeString(authHeader[len(prefix):])
		if err != nil {
			return false
		}
		// Split username:password
		credentials := string(decoded)
		colonIndex := -1
		for i := 0; i < len(credentials); i++ {
			if credentials[i] == ':' {
				colonIndex = i
				break
			}
		}
		if colonIndex == -1 {
			return false
		}
		username := credentials[:colonIndex]
		password := credentials[colonIndex+1:]

		// Validate credentials
		var decodedBcryptHash []byte
		if len(c.Basic.PasswordBcryptHashBase64Encoded) > 0 {
			decodedBcryptHash, err = base64.URLEncoding.DecodeString(c.Basic.PasswordBcryptHashBase64Encoded)
			if err != nil {
				return false
			}
			if username != c.Basic.Username || bcrypt.CompareHashAndPassword(decodedBcryptHash, []byte(password)) != nil {
				return false
			}
			return true
		}
	}
	return false
}

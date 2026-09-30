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
	authLevelGlobal   = "global"
	authLevelEndpoint = "endpoint"
)

// Config is the security configuration for Gatus
type Config struct {
	Basic *BasicConfig `yaml:"basic,omitempty"`
	OIDC  *OIDCConfig  `yaml:"oidc,omitempty"`
	Level string       `yaml:"level,omitempty"`

	gate *g8.Gate
}

// ValidateAndSetDefaults returns whether the security configuration is valid or not and sets default values.
func (c *Config) ValidateAndSetDefaults() bool {
	basicValid := c.Basic == nil || c.Basic.isValid()
	oauthValid := c.OIDC == nil || c.OIDC.ValidateAndSetDefaults()

	// Set default level
	if c.Level == "" {
		c.Level = authLevelGlobal
	}
	levelValid := c.Level == authLevelGlobal || c.Level == authLevelEndpoint
	if c.Level == authLevelEndpoint && c.Basic == nil && c.OIDC == nil {
		return false
	}

	logr.Infof("[security.config.ValidateAndSetDefaults] Set auth level to %s", c.Level)
	return levelValid && basicValid && oauthValid
}

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

// validateBasicCredentials checks username/password against the configured Basic Auth credentials.
func (c *Config) validateBasicCredentials(username, password string) bool {
	if len(c.Basic.PasswordBcryptHashBase64Encoded) == 0 {
		return true
	}
	decodedBcryptHash, err := base64.URLEncoding.DecodeString(c.Basic.PasswordBcryptHashBase64Encoded)
	if err != nil {
		return false
	}
	return username == c.Basic.Username && bcrypt.CompareHashAndPassword(decodedBcryptHash, []byte(password)) == nil
}

func (c *Config) InitializeGate() {
	if c.OIDC != nil && c.gate == nil {
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
	}
}

// ApplySecurityMiddleware applies an authentication middleware to the router passed.
func (c *Config) ApplySecurityMiddleware(router fiber.Router) error {
	c.InitializeGate()
	if c.OIDC != nil {
		router.Use(adaptor.HTTPMiddleware(c.gate.Protect))
	} else if c.Basic != nil {
		router.Use(basicauth.New(basicauth.Config{
			Authorizer: func(username, password string) bool {
				return c.validateBasicCredentials(username, password)
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
		authHeader := ctx.Get("Authorization")
		if authHeader == "" {
			return false
		}
		const prefix = "Basic "
		if len(authHeader) < len(prefix) || authHeader[:len(prefix)] != prefix {
			return false
		}
		decoded, err := base64.StdEncoding.DecodeString(authHeader[len(prefix):])
		if err != nil {
			return false
		}
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
		return c.validateBasicCredentials(credentials[:colonIndex], credentials[colonIndex+1:])
	}
	return false
}

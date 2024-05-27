// please do verify before pull request


package security

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	g8 "github.com/TwiN/g8/v2"
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


type Config struct {
	Basic *BasicConfig `yaml:"basic,omitempty"`
	OIDC  *OIDCConfig  `yaml:"oidc,omitempty"`

	gate *g8.Gate
}


func (c *Config) IsValid() bool {
	return (c.Basic != nil && c.Basic.isValid()) || (c.OIDC != nil && c.OIDC.isValid())
}
// adding description for better understanding ::
//  ENGLISH IS NOT MY MOTHER TOUNGUE SO READ THE DESCRIPTION TWICE AND UNDERSTAND THE CODE
// THANK YOU



// RegisterHandlers registers all handlers required based on the security configuration
func (c *Config) RegisterHandlers(router fiber.Router) error {
	if c.OIDC != nil {
		if err := c.OIDC.initialize(); err != nil {
			return err
	}
		router.Use(adaptor.HTTPHandler(c.OIDC.handleCallback))
		router.Use(adaptor.HTTPHandler(c.OIDC.handleLogout))
	}

	if c.Basic != nil {
		router.Use(basicauth.New(c.Basic.username, c.Basic.password))
	}

	return nil
}


func (c *Config) ApplySecurityMiddleware(router fiber.Router) {
	if c.OIDC != nil {
		router.Use(c.OIDC.middleware)
	} else if c.Basic != nil {
		router.Use(basicauth.New(c.Basic.username, c.Basic.password))
	}
}

// IsAuthenticated returns whether the user is authenticated or not   
func (c *Config) IsAuthenticated(ctx *fiber.Ctx) bool {
	if c.OIDC != nil {
		return c.OIDC.isAuthenticated(ctx)
	}

	return true
}

// BasicConfig is the configuration for Basic authentication ..
type BasicConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

    
func (c *BasicConfig) isValid() bool {
	return c.Username != "" && c.Password != ""
}

// OIDCConfig is the configuration for OIDC authentication 
type OIDCConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	IssuerURL    string `yaml:"issuer_url"`
	RedirectURL  string `yaml:"redirect_url"`
	Scopes       string `yaml:"scopes"`
}

// isValid returns whether the OIDC authentication configuration is valid or not valid
func (c *OIDCConfig) isValid() bool {
	return c.ClientID != "" && c.ClientSecret != "" && c.IssuerURL != "" && c.RedirectURL != "" && c.Scopes != ""
}

// initialize the OIDC authentication configuration
func (c *OIDCConfig) initialize() error {
	if c.gate == nil {
		c.gate = g8.New()
	}

	if err := c.gate.SetClientID(c.ClientID); err != nil {
		return err
	}

	if err := c.gate.SetClientSecret(c.ClientSecret); err != nil {
		return err
	}

	if err := c.gate.SetIssuerURL(c.IssuerURL); err != nil {
		return err
	}

	if err := c.gate.SetRedirectURL(c.RedirectURL); err != nil {
		return err
	}

	if err := c.gate.SetScopes(c.Scopes); err != nil {
		return err
	}

	return nil
}

// handleCallback handles the OIDC authentication callback
func (c *OIDCConfig) handleCallback(c *g8.Context) error {
	if c.IsAuthenticated() {
		return c.Redirect(c.Query("redirect_url"))
	}

	if err := c.Authenticate(); err != nil {
		return err
	}

	if err := c.SaveSession(); err != nil {
		return err
	}

	return c.Redirect(c.Query("redirect_url"))
}

// handleLogout handles the OIDC authentication logout
func (c *OIDCConfig) handleLogout(c *g8.Context) error {
	if err := c.InvalidateSession(); err != nil {
		return err
	}

	return c.Redirect(c.Query("redirect_url"))
}

// middleware is the OIDC authentication middleware
func (c *OIDCConfig) middleware(ctx *fiber.Ctx) error {
	if c.gate == nil {
		return fmt.Errorf("OIDC gate not initialized")
	}

	session, err := c.gate.GetSession(ctx)
	if err != nil {
return err
	}

	if !session.IsAuthenticated() {
		return c.gate.Authenticate(ctx)
	}

	return ctx.Next()
}

// isAuthenticated returns whether the user is authenticated or not
func (c *OIDCConfig) isAuthenticated(ctx *fiber.Ctx) bool {
	if c.gate == nil {
		return true
	}

	session, err := c.gate.GetSession(ctx)
	if err != nil {
		log.Println("Error getting session:", err)
		return false
	}

	return session.IsAuthenticated()
}

// HashPassword hashes the password using bcrypt
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash checks whether the password matches the hash
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomString generates a random string of the specified length
func GenerateRandomString(length int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = letters[rand.Intn(len(letters))]
	}
	return string(result)
}

// Base64Encode encodes the string using base64
func Base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// Base64Decode decodes the base64-encoded string
func Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

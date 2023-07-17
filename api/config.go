package api

import (
	"fmt"

	"github.com/TwiN/gatus/v5/security"
	"github.com/gofiber/fiber/v2"
)

type ConfigHandler struct {
	securityConfig *security.Config
}

func (handler ConfigHandler) GetConfig(c *fiber.Ctx) error {
	hasOIDC := false
	isAuthenticated := true // Default to true if no security config is set
	if handler.securityConfig != nil {
		hasOIDC = handler.securityConfig.OIDC != nil
		isAuthenticated = handler.securityConfig.IsAuthenticated(c)
	}
	// Return the config
	c.Set("Content-Type", "application/json")
	return c.Status(200).
		SendString(fmt.Sprintf(`{"oidc":%v,"authenticated":%v}`, hasOIDC, isAuthenticated))
}

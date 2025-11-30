package api

import (
	"encoding/json"
	"fmt"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/security"
	"github.com/gofiber/fiber/v2"
)

type ConfigHandler struct {
	securityConfig *security.Config
	config         *config.Config
}

func (handler ConfigHandler) GetConfig(c *fiber.Ctx) error {
	hasOIDC := false
	isAuthenticated := true // Default to true if no security config is set
	if handler.securityConfig != nil {
		hasOIDC = handler.securityConfig.OIDC != nil
		isAuthenticated = handler.securityConfig.IsAuthenticated(c)
	}

	// Prepare response with announcements
	response := map[string]interface{}{
		"oidc":          hasOIDC,
		"authenticated": isAuthenticated,
	}
	// Add announcements if available, otherwise use empty slice
	if handler.config != nil && handler.config.Announcements != nil && len(handler.config.Announcements) > 0 {
		response["announcements"] = handler.config.Announcements
	} else {
		response["announcements"] = []interface{}{}
	}

	// Return the config as JSON
	c.Set("Content-Type", "application/json")
	responseBytes, err := json.Marshal(response)
	if err != nil {
		return c.Status(500).SendString(fmt.Sprintf(`{"error":"Failed to marshal response: %s"}`, err.Error()))
	}
	return c.Status(200).Send(responseBytes)
}

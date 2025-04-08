package api

import (
	"fmt"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

type ConfigHandler struct {
	config *config.Config
}

func (handler ConfigHandler) GetConfig(c *fiber.Ctx) error {
	hasOIDC := false
	isAuthenticated := true // Default to true if no security config is set
	if handler.config.Security != nil {
		hasOIDC = handler.config.Security.OIDC != nil
		isAuthenticated = handler.config.Security.IsAuthenticated(c)
	}
	// Return the config
	c.Set("Content-Type", "application/json")
	return c.Status(200).
		SendString(fmt.Sprintf(`{"oidc":%v,"authenticated":%v}`, hasOIDC, isAuthenticated))
}

func (handler ConfigHandler) ReloadConfig(c *fiber.Ctx) error {
	// Force last modified time update which will cause config reload
	handler.config.RequestConfigForceReload()

	logr.Info("[api.config] Configuration Reload has been requested")

	// Return OK
	c.Set("Content-Type", "application/json")
	return c.SendStatus(204)
}

package api

import (
	"github.com/TwiN/gatus/v5/alerting/health"
	"github.com/gofiber/fiber/v2"
)

// AlertingHealth exposes alerting configuration and delivery health for authenticated clients.
func AlertingHealth(c *fiber.Ctx) error {
	return c.JSON(health.GetSnapshot())
}
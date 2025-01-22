package api

import (
	"fmt"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/watchdog"
	"github.com/gofiber/fiber/v2"
)

func GetAllEndpoints(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {

		var s []endpoint.EndpointWithKey
		for _, e := range cfg.Endpoints {
			s = append(s, endpoint.EndpointWithKey{Endpoint: *e, Key: e.Key()})

		}
		return c.Status(200).JSON(s)

	}
}

func OnDemandTrigger(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := c.Params("key")
		e := cfg.GetEndpointByKey(key)
		if e == nil {
			return c.Status(404).JSON(fiber.Map{"error": "endpoint not found"})
		}
		watchdog.OnDemandExecute(e, cfg.Alerting, cfg.Maintenance, cfg.Connectivity, cfg.DisableMonitoringLock, cfg.Metrics)
		c.Set("Content-Type", "application/json")
		return c.Status(200).
			JSON(fiber.Map{"message": fmt.Sprintf("endpoint %s triggered", key)})

	}
}

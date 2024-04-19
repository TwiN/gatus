package api

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/watchdog"
	"github.com/gofiber/fiber/v2"
)

func CreateExternalEndpointResult(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Check if the success query parameter is present
		success, exists := c.Queries()["success"]
		if !exists || (success != "true" && success != "false") {
			return c.Status(400).SendString("missing or invalid success query parameter")
		}
		// Check if the authorization bearer token header is correct
		authorizationHeader := string(c.Request().Header.Peek("Authorization"))
		if !strings.HasPrefix(authorizationHeader, "Bearer ") {
			return c.Status(401).SendString("invalid Authorization header")
		}
		token := strings.TrimSpace(strings.TrimPrefix(authorizationHeader, "Bearer "))
		if len(token) == 0 {
			return c.Status(401).SendString("bearer token must not be empty")
		}
		key := c.Params("key")
		externalEndpoint := cfg.GetExternalEndpointByKey(key)
		if externalEndpoint == nil {
			log.Printf("[api.CreateExternalEndpointResult] External endpoint with key=%s not found", key)
			return c.Status(404).SendString("not found")
		}
		if externalEndpoint.Token != token {
			log.Printf("[api.CreateExternalEndpointResult] Invalid token for external endpoint with key=%s", key)
			return c.Status(401).SendString("invalid token")
		}
		// Persist the result in the storage
		result := &core.Result{
			Timestamp: time.Now(),
			Success:   c.QueryBool("success"),
			Errors:    []string{},
		}
		convertedEndpoint := externalEndpoint.ToEndpoint()
		if err := store.Get().Insert(convertedEndpoint, result); err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return c.Status(404).SendString(err.Error())
			}
			log.Printf("[api.CreateExternalEndpointResult] Failed to insert result in storage: %s", err.Error())
			return c.Status(500).SendString(err.Error())
		}
		log.Printf("[api.CreateExternalEndpointResult] Successfully inserted result for external endpoint with key=%s and success=%s", c.Params("key"), success)
		// Check if an alert should be triggered or resolved
		if !cfg.Maintenance.IsUnderMaintenance() {
			watchdog.HandleAlerting(convertedEndpoint, result, cfg.Alerting, cfg.Debug)
			externalEndpoint.NumberOfSuccessesInARow = convertedEndpoint.NumberOfSuccessesInARow
			externalEndpoint.NumberOfFailuresInARow = convertedEndpoint.NumberOfFailuresInARow
		}
		// Return the result
		return c.Status(200).SendString("")
	}
}

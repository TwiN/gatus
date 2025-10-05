package api

import (
	"fmt"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/suite"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/gofiber/fiber/v2"
)

// SuiteStatuses handles requests to retrieve all suite statuses
func SuiteStatuses(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, 100)
		params := paging.NewSuiteStatusParams().WithPagination(page, pageSize)
		suiteStatuses, err := store.Get().GetAllSuiteStatuses(params)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("Failed to retrieve suite statuses: %v", err),
			})
		}
		// If no statuses exist yet, create empty ones from config
		if len(suiteStatuses) == 0 {
			for _, s := range cfg.Suites {
				if s.IsEnabled() {
					suiteStatuses = append(suiteStatuses, suite.NewStatus(s))
				}
			}
		}
		return c.Status(fiber.StatusOK).JSON(suiteStatuses)
	}
}

// SuiteStatus handles requests to retrieve a single suite's status
func SuiteStatus(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, 100)
		key := c.Params("key")
		params := paging.NewSuiteStatusParams().WithPagination(page, pageSize)
		status, err := store.Get().GetSuiteStatusByKey(key, params)
		if err != nil || status == nil {
			// Try to find the suite in config
			for _, s := range cfg.Suites {
				if s.Key() == key {
					status = suite.NewStatus(s)
					break
				}
			}
			if status == nil {
				return c.Status(404).JSON(fiber.Map{
					"error": fmt.Sprintf("Suite with key '%s' not found", key),
				})
			}
		}
		return c.Status(fiber.StatusOK).JSON(status)
	}
}

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
		// Annotate suite statuses with their current suspended state from the configuration, and create
		// placeholder statuses for any enabled suite that doesn't have one yet (either because it never ran, or
		// because it's suspended and was therefore never run), so that suspended suites are still visible (and
		// filterable) in the UI.
		suspendedKeys := make(map[string]bool)
		for _, s := range cfg.Suites {
			if s.IsSuspended() {
				suspendedKeys[s.Key()] = true
			}
		}
		seenKeys := make(map[string]bool, len(suiteStatuses))
		for _, status := range suiteStatuses {
			seenKeys[status.Key] = true
			status.Suspended = suspendedKeys[status.Key]
		}
		for _, s := range cfg.Suites {
			if s.IsEnabled() && !seenKeys[s.Key()] {
				suiteStatuses = append(suiteStatuses, suite.NewStatus(s))
				seenKeys[s.Key()] = true
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
		} else {
			for _, s := range cfg.Suites {
				if s.Key() == key {
					status.Suspended = s.IsSuspended()
					break
				}
			}
		}
		return c.Status(fiber.StatusOK).JSON(status)
	}
}

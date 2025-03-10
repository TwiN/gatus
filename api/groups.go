package api

import (
	"encoding/json"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

func Groups(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		groups, err := store.Get().GetGroups()
		if err != nil {
			logr.Errorf("[api.Groups] Failed to retrieve groups: %s", err.Error())
			return c.Status(500).SendString(err.Error())
		}
		output, err := json.Marshal(groups)
		if err != nil {
			logr.Errorf("[api.Groups] Unable to marshal object to JSON: %s", err.Error())
			return c.Status(500).SendString("unable to marshal object to JSON")
		}
		c.Set("Content-Type", "application/json")
		return c.Status(200).Send(output)
	}
}

// GroupStatuses retrieves all endpoint.Status for a given group
func GroupStatuses(c *fiber.Ctx) error {
	page, pageSize := extractPageAndPageSizeFromRequest(c)
	endpointStatuses, err := store.Get().GetEndpointStatusesByGroup(c.Params("group"), paging.NewEndpointStatusParams().WithResults(page, pageSize))
	if err != nil {
		logr.Errorf("[api.GroupStatuses] Failed to retrieve group statuses: %s", err.Error())
		return c.Status(500).SendString(err.Error())
	}
	output, err := json.Marshal(endpointStatuses)
	if err != nil {
		logr.Errorf("[api.GroupStatuses] Unable to marshal object to JSON: %s", err.Error())
		return c.Status(500).SendString("unable to marshal object to JSON")
	}
	c.Set("Content-Type", "application/json")
	return c.Status(200).Send(output)
}
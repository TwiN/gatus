package api

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/gofiber/fiber/v2"
)

func UptimeRaw(c *fiber.Ctx) error {
	duration := c.Params("duration")
	parsedDuration, err := ParseCustomDuration(duration)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	from := time.Now().Add(-parsedDuration)
	// Because uptime metrics are stored by hour, we have to ensure at least 2 hours for 1h queries
	if parsedDuration < 2*time.Hour {
		from = time.Now().Add(-2 * time.Hour)
	}
	key, err := url.QueryUnescape(c.Params("key"))
	if err != nil {
		return c.Status(400).SendString("invalid key encoding")
	}
	uptime, err := store.Get().GetUptimeByKey(key, from, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return c.Status(404).SendString(err.Error())
		} else if errors.Is(err, common.ErrInvalidTimeRange) {
			return c.Status(400).SendString(err.Error())
		}
		return c.Status(500).SendString(err.Error())
	}

	c.Set("Content-Type", "text/plain")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Expires", "0")
	return c.Status(200).Send([]byte(fmt.Sprintf("%f", uptime)))
}

func ResponseTimeRaw(c *fiber.Ctx) error {
	duration := c.Params("duration")
	parsedDuration, err := ParseCustomDuration(duration)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	from := time.Now().Add(-parsedDuration)
	// Because response time metrics are stored by hour, we have to ensure at least 2 hours for 1h queries
	if parsedDuration < 2*time.Hour {
		from = time.Now().Add(-2 * time.Hour)
	}
	key, err := url.QueryUnescape(c.Params("key"))
	if err != nil {
		return c.Status(400).SendString("invalid key encoding")
	}
	responseTime, err := store.Get().GetAverageResponseTimeByKey(key, from, time.Now())
	if err != nil {
		if errors.Is(err, common.ErrEndpointNotFound) {
			return c.Status(404).SendString(err.Error())
		} else if errors.Is(err, common.ErrInvalidTimeRange) {
			return c.Status(400).SendString(err.Error())
		}
		return c.Status(500).SendString(err.Error())
	}

	c.Set("Content-Type", "text/plain")
	c.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Set("Expires", "0")
	return c.Status(200).Send([]byte(fmt.Sprintf("%d", responseTime)))
}

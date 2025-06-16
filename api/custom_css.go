package api

import (
	"github.com/gofiber/fiber/v2"
)

type CustomCSSHandler struct {
	customCSS string
}

func (handler CustomCSSHandler) GetCustomCSS(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/css")
	return c.Status(200).SendString(handler.customCSS)
}

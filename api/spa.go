package api

import (
	_ "embed"
	"html/template"
	"log"

	"github.com/TwiN/gatus/v5/config/ui"
	static "github.com/TwiN/gatus/v5/web"
	"github.com/gofiber/fiber/v2"
)

func SinglePageApplication(ui *ui.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		t, err := template.ParseFS(static.FileSystem, static.IndexPath)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			log.Println("[api][SinglePageApplication] Failed to parse template. This should never happen, because the template is validated on start. Error:", err.Error())
			return c.Status(500).SendString("Failed to parse template. This should never happen, because the template is validated on start.")
		}
		c.Set("Content-Type", "text/html")
		err = t.Execute(c, ui)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			log.Println("[api][SinglePageApplication] Failed to execute template. This should never happen, because the template is validated on start. Error:", err.Error())
			return c.Status(500).SendString("Failed to parse template. This should never happen, because the template is validated on start.")
		}
		return c.SendStatus(200)
	}
}

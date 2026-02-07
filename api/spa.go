package api

import (
	_ "embed"
	"html/template"
	"log/slog"

	"github.com/TwiN/gatus/v5/config/ui"
	static "github.com/TwiN/gatus/v5/web"
	"github.com/gofiber/fiber/v2"
)

func SinglePageApplication(uiConfig *ui.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		vd := ui.ViewData{UI: uiConfig}
		{
			themeFromCookie := string(c.Request().Header.Cookie("theme"))
			if len(themeFromCookie) > 0 {
				if themeFromCookie == "dark" {
					vd.Theme = "dark"
				}
			} else if uiConfig.IsDarkMode() { // Since there's no theme cookie, we'll rely on ui.DarkMode
				vd.Theme = "dark"
			}
		}
		t, err := template.ParseFS(static.FileSystem, static.IndexPath)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			slog.Error("Failed to parse template. This should never happen, because the template is validated on start.", "error", err)
			return c.Status(500).SendString("Failed to parse template. This should never happen, because the template is validated on start.")
		}
		c.Set("Content-Type", "text/html")
		err = t.Execute(c, vd)
		if err != nil {
			// This should never happen, because ui.ValidateAndSetDefaults validates that the template works.
			slog.Error("Failed to parse template. This should never happen, because the template is validated on start.", "error", err)
			return c.Status(500).SendString("Failed to parse template. This should never happen, because the template is validated on start.")
		}
		return c.SendStatus(200)
	}
}

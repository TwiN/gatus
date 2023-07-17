package controller

import (
	"log"
	"os"
	"time"

	"github.com/TwiN/gatus/v5/api"
	"github.com/TwiN/gatus/v5/config"
	"github.com/gofiber/fiber/v2"
)

var (
	app *fiber.App
)

// Handle creates the router and starts the server
func Handle(cfg *config.Config) {
	api := api.New(cfg)
	app = api.Router()
	server := app.Server()
	server.ReadTimeout = 15 * time.Second
	server.WriteTimeout = 15 * time.Second
	server.IdleTimeout = 15 * time.Second
	server.TLSConfig = cfg.Web.TLSConfig()
	if os.Getenv("ROUTER_TEST") == "true" {
		return
	}
	log.Println("[controller][Handle] Listening on " + cfg.Web.SocketAddress())
	if server.TLSConfig != nil {
		log.Println("[controller][Handle]", app.ListenTLS(cfg.Web.SocketAddress(), "", ""))
	} else {
		log.Println("[controller][Handle]", app.Listen(cfg.Web.SocketAddress()))
	}
}

// Shutdown stops the server
func Shutdown() {
	if app != nil {
		_ = app.Shutdown()
		app = nil
	}
}

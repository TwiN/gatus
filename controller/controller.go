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
	if os.Getenv("ROUTER_TEST") == "true" {
		return
	}
	log.Println("[controller][Handle] Listening on " + cfg.Web.SocketAddress())
	if cfg.Web.HasTLS() {
		err := app.ListenTLS(cfg.Web.SocketAddress(), cfg.Web.TLS.CertificateFile, cfg.Web.TLS.PrivateKeyFile)
		if err != nil {
			log.Fatal("[controller][Handle]", err)
		}
	} else {
		err := app.Listen(cfg.Web.SocketAddress())
		if err != nil {
			log.Fatal("[controller][Handle]", err)
		}
	}
	log.Println("[controller][Handle] Server has shut down successfully")
}

// Shutdown stops the server
func Shutdown() {
	if app != nil {
		_ = app.Shutdown()
		app = nil
	}
}

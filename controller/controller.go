package controller

import (
	"net"
	"os"
	"time"

	"github.com/TwiN/gatus/v5/api"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/logr"
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

	if cfg.Web.Socket != "" {
		if cfg.Web.TLS != nil {
			logr.Warn("[controller.Handle] Using socket mode: defined TLS settings will be ignored.")
		}

		logr.Info("[controller.Handle] Listening on " + cfg.Web.Socket)
		uds_listener, err := net.Listen("unix", cfg.Web.Socket)
		if err != nil {
			logr.Fatalf("[controller.Handle] Socket creation failed: %s", err.Error())
		}

		err = os.Chmod(cfg.Web.Socket, 0660)
		if err != nil {
			logr.Fatalf("[controller.Handle] Failed to set 660 permissions on socket: %s", err.Error())
		}

		err = app.Listener(uds_listener)
		if err != nil {
			logr.Fatalf("[controller.Handle] Listening on socket failed: %s", err.Error())
		}
	} else {
		logr.Info("[controller.Handle] Listening on " + cfg.Web.SocketAddress())
		if cfg.Web.HasTLS() {
			err := app.ListenTLS(cfg.Web.SocketAddress(), cfg.Web.TLS.CertificateFile, cfg.Web.TLS.PrivateKeyFile)
			if err != nil {
				logr.Fatalf("[controller.Handle] %s", err.Error())
			}
		} else {
			err := app.Listen(cfg.Web.SocketAddress())
			if err != nil {
				logr.Fatalf("[controller.Handle] %s", err.Error())
			}
		}
	}

	logr.Info("[controller.Handle] Server has shut down successfully")
}

// Shutdown stops the server
func Shutdown() {
	if app != nil {
		_ = app.Shutdown()
		app = nil
	}
}

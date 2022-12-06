package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/controller/handler"
)

var (
	// server is the http.Server created by Handle.
	// The only reason it exists is for testing purposes.
	server *http.Server
)

// Handle creates the router and starts the server
func Handle(cfg *config.Config) {
	var router http.Handler = handler.CreateRouter(cfg)
	if os.Getenv("ENVIRONMENT") == "dev" {
		router = handler.DevelopmentCORS(router)
	}
	server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Web.Address, cfg.Web.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	log.Println("[controller][Handle] Listening on " + cfg.Web.SocketAddress())
	if os.Getenv("ROUTER_TEST") == "true" {
		return
	}
	log.Println("[controller][Handle]", server.ListenAndServe())
}

// Shutdown stops the server
func Shutdown() {
	if server != nil {
		_ = server.Shutdown(context.TODO())
		server = nil
	}
}

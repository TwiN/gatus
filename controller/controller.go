package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/TwiN/gatus/v3/config"
	"github.com/TwiN/gatus/v3/config/ui"
	"github.com/TwiN/gatus/v3/config/web"
	"github.com/TwiN/gatus/v3/controller/handler"
	"github.com/TwiN/gatus/v3/security"
)

var (
	// server is the http.Server created by Handle.
	// The only reason it exists is for testing purposes.
	server *http.Server
)

// Handle creates the router and starts the server
func Handle(securityConfig *security.Config, webConfig *web.Config, uiConfig *ui.Config, config *config.Config, enableMetrics bool) {
	var router http.Handler = handler.CreateRouter(ui.StaticFolder, securityConfig, uiConfig, config, enableMetrics)
	if os.Getenv("ENVIRONMENT") == "dev" {
		router = handler.DevelopmentCORS(router)
	}
	server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", webConfig.Address, webConfig.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	log.Println("[controller][Handle] Listening on " + webConfig.SocketAddress())
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

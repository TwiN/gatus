package handler

import (
	"net/http"

	"github.com/TwiN/gatus/v3/config/ui"
	"github.com/TwiN/gatus/v3/security"
	"github.com/TwiN/gatus/v3/web"
	"github.com/TwiN/health"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CreateRouter(securityConfig *security.Config, uiConfig *ui.Config, enabledMetrics bool) *mux.Router {
	router := mux.NewRouter()
	if enabledMetrics {
		router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	}
	router.Handle("/health", health.Handler().WithJSON(true)).Methods("GET")
	// Endpoints
	router.HandleFunc("/api/v1/services/statuses", secureIfNecessary(securityConfig, ServiceStatuses)).Methods("GET") // No GzipHandler for this one, because we cache the content as Gzipped already
	router.HandleFunc("/api/v1/services/{key}/statuses", secureIfNecessary(securityConfig, GzipHandlerFunc(ServiceStatus))).Methods("GET")
	// TODO: router.HandleFunc("/api/v1/services/{key}/uptimes", secureIfNecessary(securityConfig, GzipHandlerFunc(serviceUptimesHandler))).Methods("GET")
	// TODO: router.HandleFunc("/api/v1/services/{key}/events", secureIfNecessary(securityConfig, GzipHandlerFunc(serviceEventsHandler))).Methods("GET")
	router.HandleFunc("/api/v1/services/{key}/uptimes/{duration}/badge.svg", UptimeBadge).Methods("GET")
	router.HandleFunc("/api/v1/services/{key}/response-times/{duration}/badge.svg", ResponseTimeBadge).Methods("GET")
	router.HandleFunc("/api/v1/services/{key}/response-times/{duration}/chart.svg", ResponseTimeChart).Methods("GET")
	// SPA
	router.HandleFunc("/services/{service}", SinglePageApplication(uiConfig)).Methods("GET")
	router.HandleFunc("/", SinglePageApplication(uiConfig)).Methods("GET")
	// Everything else falls back on static content
	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.FS(web.StaticFolder))))
	return router
}

func secureIfNecessary(securityConfig *security.Config, handler http.HandlerFunc) http.HandlerFunc {
	if securityConfig != nil && securityConfig.IsValid() {
		return security.Handler(handler, securityConfig)
	}
	return handler
}

package handler

import (
	"net/http"

	"github.com/TwiN/gatus/v3/config/ui"
	"github.com/TwiN/gatus/v3/security"
	"github.com/TwiN/health"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CreateRouter(staticFolder string, securityConfig *security.Config, uiConfig *ui.Config, enabledMetrics bool) *mux.Router {
	router := mux.NewRouter()
	if enabledMetrics {
		router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	}
	router.Handle("/health", health.Handler().WithJSON(true)).Methods("GET")
	router.HandleFunc("/favicon.ico", FavIcon(staticFolder)).Methods("GET")
	// Endpoints
	router.HandleFunc("/api/v1/endpoints/statuses", secureIfNecessary(securityConfig, EndpointStatuses)).Methods("GET") // No GzipHandler for this one, because we cache the content as Gzipped already
	router.HandleFunc("/api/v1/endpoints/{key}/statuses", secureIfNecessary(securityConfig, GzipHandlerFunc(EndpointStatus))).Methods("GET")
	router.HandleFunc("/api/v1/endpoints/{key}/uptimes/{duration}/badge.svg", UptimeBadge).Methods("GET")
	router.HandleFunc("/api/v1/endpoints/{key}/response-times/{duration}/badge.svg", ResponseTimeBadge).Methods("GET")
	router.HandleFunc("/api/v1/endpoints/{key}/response-times/{duration}/chart.svg", ResponseTimeChart).Methods("GET")
	// XXX: Remove the lines between this and the next XXX comment in v4.0.0
	router.HandleFunc("/api/v1/services/statuses", secureIfNecessary(securityConfig, EndpointStatuses)).Methods("GET") // No GzipHandler for this one, because we cache the content as Gzipped already
	router.HandleFunc("/api/v1/services/{key}/statuses", secureIfNecessary(securityConfig, GzipHandlerFunc(EndpointStatus))).Methods("GET")
	router.HandleFunc("/api/v1/services/{key}/uptimes/{duration}/badge.svg", UptimeBadge).Methods("GET")
	router.HandleFunc("/api/v1/services/{key}/response-times/{duration}/badge.svg", ResponseTimeBadge).Methods("GET")
	router.HandleFunc("/api/v1/services/{key}/response-times/{duration}/chart.svg", ResponseTimeChart).Methods("GET")
	// XXX: Remove the lines between this and the previous XXX comment in v4.0.0
	// SPA
	router.HandleFunc("/services/{name}", SinglePageApplication(staticFolder, uiConfig)).Methods("GET") // XXX: Remove this in v4.0.0
	router.HandleFunc("/endpoints/{name}", SinglePageApplication(staticFolder, uiConfig)).Methods("GET")
	router.HandleFunc("/", SinglePageApplication(staticFolder, uiConfig)).Methods("GET")
	// Everything else falls back on static content
	router.PathPrefix("/").Handler(GzipHandler(http.FileServer(http.Dir(staticFolder))))
	return router
}

func secureIfNecessary(securityConfig *security.Config, handler http.HandlerFunc) http.HandlerFunc {
	if securityConfig != nil && securityConfig.IsValid() {
		return security.Handler(handler, securityConfig)
	}
	return handler
}

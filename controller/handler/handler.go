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
	api := router.PathPrefix("/api").Subrouter()
	protected := api.PathPrefix("/").Subrouter()
	unprotected := api.PathPrefix("/").Subrouter()
	if securityConfig != nil {
		if err := securityConfig.RegisterHandlers(router); err != nil {
			panic(err)
		}
		if err := securityConfig.ApplySecurityMiddleware(protected); err != nil {
			panic(err)
		}
	}
	// Endpoints
	unprotected.Handle("/v1/config", ConfigHandler{securityConfig: securityConfig}).Methods("GET")
	protected.HandleFunc("/v1/endpoints/statuses", EndpointStatuses).Methods("GET") // No GzipHandler for this one, because we cache the content as Gzipped already
	protected.HandleFunc("/v1/endpoints/{key}/statuses", GzipHandlerFunc(EndpointStatus)).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/health/badge.svg", HealthBadge).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/uptimes/{duration}/badge.svg", UptimeBadge).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/response-times/{duration}/badge.svg", ResponseTimeBadge).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/response-times/{duration}/chart.svg", ResponseTimeChart).Methods("GET")
	// XXX: Remove the lines between this and the next XXX comment in v4.0.0
	protected.HandleFunc("/v1/services/statuses", EndpointStatuses).Methods("GET") // No GzipHandler for this one, because we cache the content as Gzipped already
	protected.HandleFunc("/v1/services/{key}/statuses", GzipHandlerFunc(EndpointStatus)).Methods("GET")
	unprotected.HandleFunc("/v1/services/{key}/uptimes/{duration}/badge.svg", UptimeBadge).Methods("GET")
	unprotected.HandleFunc("/v1/services/{key}/response-times/{duration}/badge.svg", ResponseTimeBadge).Methods("GET")
	unprotected.HandleFunc("/v1/services/{key}/response-times/{duration}/chart.svg", ResponseTimeChart).Methods("GET")
	// XXX: Remove the lines between this and the previous XXX comment in v4.0.0
	// Misc
	router.Handle("/health", health.Handler().WithJSON(true)).Methods("GET")
	router.HandleFunc("/favicon.ico", FavIcon(staticFolder)).Methods("GET")
	// SPA
	router.HandleFunc("/services/{name}", SinglePageApplication(staticFolder, uiConfig)).Methods("GET") // XXX: Remove this in v4.0.0
	router.HandleFunc("/endpoints/{name}", SinglePageApplication(staticFolder, uiConfig)).Methods("GET")
	router.HandleFunc("/", SinglePageApplication(staticFolder, uiConfig)).Methods("GET")
	// Everything else falls back on static content
	router.PathPrefix("/").Handler(GzipHandler(http.FileServer(http.Dir(staticFolder))))
	return router
}

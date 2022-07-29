package handler

import (
	"net/http"

    "github.com/TwiN/gatus/v4/config"
	"github.com/TwiN/health"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CreateRouter(staticFolder string, config *config.Config) *mux.Router {
	router := mux.NewRouter()
	api := router.PathPrefix("/api").Subrouter()
	protected := api.PathPrefix("/").Subrouter()
	unprotected := api.PathPrefix("/").Subrouter()
	if config != nil {
		if config.Metrics {
			router.Handle("/metrics", promhttp.Handler()).Methods("GET")
		}
		if config.Security != nil {
			if err := config.Security.RegisterHandlers(router); err != nil {
				panic(err)
			}
			if err := config.Security.ApplySecurityMiddleware(protected); err != nil {
				panic(err)
			}
		}

		// Endpoints
		unprotected.Handle("/v1/config", ConfigHandler{securityConfig: config.Security}).Methods("GET")
		unprotected.HandleFunc("/v1/endpoints/{key}/response-times/{duration}/badge.svg", ResponseTimeBadge(config)).Methods("GET")
		unprotected.HandleFunc("/v1/services/{key}/response-times/{duration}/badge.svg", ResponseTimeBadge(config)).Methods("GET")
		// SPA
		router.HandleFunc("/services/{name}", SinglePageApplication(staticFolder, config.UI)).Methods("GET") // XXX: Remove this in v4.0.0
		router.HandleFunc("/endpoints/{name}", SinglePageApplication(staticFolder, config.UI)).Methods("GET")
		router.HandleFunc("/", SinglePageApplication(staticFolder, config.UI)).Methods("GET")
	}

	protected.HandleFunc("/v1/endpoints/statuses", EndpointStatuses).Methods("GET") // No GzipHandler for this one, because we cache the content as Gzipped already
	protected.HandleFunc("/v1/endpoints/{key}/statuses", GzipHandlerFunc(EndpointStatus)).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/health/badge.svg", HealthBadge).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/uptimes/{duration}/badge.svg", UptimeBadge).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/response-times/{duration}/chart.svg", ResponseTimeChart).Methods("GET")
	// Misc
	router.Handle("/health", health.Handler().WithJSON(true)).Methods("GET")
	router.HandleFunc("/favicon.ico", FavIcon(staticFolder)).Methods("GET")
	// Everything else falls back on static content
	router.PathPrefix("/").Handler(GzipHandler(http.FileServer(http.Dir(staticFolder))))
	return router
}

package handler

import (
	"io/fs"
	"net/http"

	"github.com/TwiN/gatus/v5/config"
	static "github.com/TwiN/gatus/v5/web"
	"github.com/TwiN/health"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func CreateRouter(cfg *config.Config) *mux.Router {
	router := mux.NewRouter()
	if cfg.Metrics {
		router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	}
	api := router.PathPrefix("/api").Subrouter()
	protected := api.PathPrefix("/").Subrouter()
	unprotected := api.PathPrefix("/").Subrouter()
	if cfg.Security != nil {
		if err := cfg.Security.RegisterHandlers(router); err != nil {
			panic(err)
		}
		if err := cfg.Security.ApplySecurityMiddleware(protected); err != nil {
			panic(err)
		}
	}
	// Endpoints
	unprotected.Handle("/v1/config", ConfigHandler{securityConfig: cfg.Security}).Methods("GET")
	protected.HandleFunc("/v1/endpoints/statuses", EndpointStatuses(cfg)).Methods("GET") // No GzipHandler for this one, because we cache the content as Gzipped already
	protected.HandleFunc("/v1/endpoints/{key}/statuses", GzipHandlerFunc(EndpointStatus)).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/health/badge.svg", HealthBadge).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/uptimes/{duration}/badge.svg", UptimeBadge).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/response-times/{duration}/badge.svg", ResponseTimeBadge(cfg)).Methods("GET")
	unprotected.HandleFunc("/v1/endpoints/{key}/response-times/{duration}/chart.svg", ResponseTimeChart).Methods("GET")
	// Misc
	router.Handle("/health", health.Handler().WithJSON(true)).Methods("GET")
	// SPA
	router.HandleFunc("/endpoints/{name}", SinglePageApplication(cfg.UI)).Methods("GET")
	router.HandleFunc("/", SinglePageApplication(cfg.UI)).Methods("GET")
	// Everything else falls back on static content
	staticFileSystem, err := fs.Sub(static.FileSystem, static.RootPath)
	if err != nil {
		panic(err)
	}
	router.PathPrefix("/").Handler(GzipHandler(http.FileServer(http.FS(staticFileSystem))))
	return router
}

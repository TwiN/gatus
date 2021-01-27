package controller

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/security"
	"github.com/TwinProduction/gatus/watchdog"
	"github.com/TwinProduction/gocache"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	cacheTTL = 10 * time.Second
)

var (
	cache = gocache.NewCache().WithMaxSize(100).WithEvictionPolicy(gocache.LeastRecentlyUsed)
)

func init() {
	if err := cache.StartJanitor(); err != nil {
		log.Fatal("[controller][init] Failed to start cache janitor:", err.Error())
	}
}

// Handle creates the router and starts the server
func Handle() {
	cfg := config.Get()
	var router http.Handler = CreateRouter(cfg)
	if os.Getenv("ENVIRONMENT") == "dev" {
		router = developmentCorsHandler(router)
	}
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Web.Address, cfg.Web.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	log.Printf("[controller][Handle] Listening on %s%s\n", cfg.Web.SocketAddress(), cfg.Web.ContextRoot)
	log.Fatal(server.ListenAndServe())
}

// CreateRouter creates the router for the http server
func CreateRouter(cfg *config.Config) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/favicon.ico", favIconHandler).Methods("GET") // favicon needs to be always served from the root
	router.HandleFunc(cfg.Web.PrependWithContextRoot("/api/v1/statuses"), secureIfNecessary(cfg, serviceStatusesHandler)).Methods("GET")
	router.HandleFunc(cfg.Web.PrependWithContextRoot("/api/v1/statuses/{key}"), secureIfNecessary(cfg, GzipHandlerFunc(serviceStatusHandler))).Methods("GET")
	router.HandleFunc(cfg.Web.PrependWithContextRoot("/api/v1/badges/uptime/{duration}/{identifier}"), badgeHandler).Methods("GET")
	router.HandleFunc(cfg.Web.PrependWithContextRoot("/health"), healthHandler).Methods("GET")
	router.PathPrefix(cfg.Web.ContextRoot).Handler(GzipHandler(http.StripPrefix(cfg.Web.ContextRoot, http.FileServer(http.Dir("./static")))))
	if cfg.Metrics {
		router.Handle(cfg.Web.PrependWithContextRoot("/metrics"), promhttp.Handler()).Methods("GET")
	}
	return router
}

func secureIfNecessary(cfg *config.Config, handler http.HandlerFunc) http.HandlerFunc {
	if cfg.Security != nil && cfg.Security.IsValid() {
		return security.Handler(serviceStatusesHandler, cfg.Security)
	}
	return handler
}

// serviceStatusesHandler handles requests to retrieve all service statuses
// Due to the size of the response, this function leverages a cache.
// Must not be wrapped by GzipHandler
func serviceStatusesHandler(writer http.ResponseWriter, r *http.Request) {
	gzipped := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
	var exists bool
	var value interface{}
	if gzipped {
		writer.Header().Set("Content-Encoding", "gzip")
		value, exists = cache.Get("service-status-gzipped")
	} else {
		value, exists = cache.Get("service-status")
	}
	var data []byte
	if !exists {
		var err error
		buffer := &bytes.Buffer{}
		gzipWriter := gzip.NewWriter(buffer)
		data, err = watchdog.GetServiceStatusesAsJSON()
		if err != nil {
			log.Printf("[controller][serviceStatusesHandler] Unable to marshal object to JSON: %s", err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("Unable to marshal object to JSON"))
			return
		}
		_, _ = gzipWriter.Write(data)
		_ = gzipWriter.Close()
		gzippedData := buffer.Bytes()
		cache.SetWithTTL("service-status", data, cacheTTL)
		cache.SetWithTTL("service-status-gzipped", gzippedData, cacheTTL)
		if gzipped {
			data = gzippedData
		}
	} else {
		data = value.([]byte)
	}
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(data)
}

// serviceStatusHandler retrieves a single ServiceStatus by group name and service name
func serviceStatusHandler(writer http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceStatus := watchdog.GetServiceStatusByKey(vars["key"])
	if serviceStatus == nil {
		log.Printf("[controller][serviceStatusHandler] Service with key=%s not found", vars["key"])
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte("not found"))
		return
	}
	data, err := json.Marshal(serviceStatus)
	if err != nil {
		log.Printf("[controller][serviceStatusHandler] Unable to marshal object to JSON: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte("Unable to marshal object to JSON"))
		return
	}
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(data)
}

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("{\"status\":\"UP\"}"))
}

// favIconHandler handles requests for /favicon.ico
func favIconHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "./static/favicon.ico")
}

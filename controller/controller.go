package controller

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/security"
	"github.com/TwinProduction/gatus/storage"
	"github.com/TwinProduction/gatus/storage/store/common"
	"github.com/TwinProduction/gatus/storage/store/common/paging"
	"github.com/TwinProduction/gocache"
	"github.com/TwinProduction/health"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	cacheTTL = 10 * time.Second
)

var (
	cache = gocache.NewCache().WithMaxSize(100).WithEvictionPolicy(gocache.FirstInFirstOut)

	// staticFolder is the path to the location of the static folder from the root path of the project
	// The only reason this is exposed is to allow running tests from a different path than the root path of the project
	staticFolder = "./web/static"

	// server is the http.Server created by Handle.
	// The only reason it exists is for testing purposes.
	server *http.Server
)

// Handle creates the router and starts the server
func Handle(securityConfig *security.Config, webConfig *config.WebConfig, enableMetrics bool) {
	var router http.Handler = CreateRouter(securityConfig, enableMetrics)
	if os.Getenv("ENVIRONMENT") == "dev" {
		router = developmentCorsHandler(router)
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

// CreateRouter creates the router for the http server
func CreateRouter(securityConfig *security.Config, enabledMetrics bool) *mux.Router {
	router := mux.NewRouter()
	if enabledMetrics {
		router.Handle("/metrics", promhttp.Handler()).Methods("GET")
	}
	router.Handle("/health", health.Handler().WithJSON(true)).Methods("GET")
	router.HandleFunc("/favicon.ico", favIconHandler).Methods("GET")
	// Deprecated endpoints
	router.HandleFunc("/api/v1/statuses", secureIfNecessary(securityConfig, serviceStatusesHandler)).Methods("GET") // No GzipHandler for this one, because we cache the content as Gzipped already
	router.HandleFunc("/api/v1/statuses/{key}", secureIfNecessary(securityConfig, GzipHandlerFunc(serviceStatusHandler))).Methods("GET")
	router.HandleFunc("/api/v1/badges/uptime/{duration}/{identifier}", uptimeBadgeHandler).Methods("GET")
	// New endpoints
	router.HandleFunc("/api/v1/services/statuses", secureIfNecessary(securityConfig, serviceStatusesHandler)).Methods("GET") // No GzipHandler for this one, because we cache the content as Gzipped already
	router.HandleFunc("/api/v1/services/{key}/statuses", secureIfNecessary(securityConfig, GzipHandlerFunc(serviceStatusHandler))).Methods("GET")
	// TODO: router.HandleFunc("/api/v1/services/{key}/uptimes", secureIfNecessary(securityConfig, GzipHandlerFunc(serviceUptimesHandler))).Methods("GET")
	// TODO: router.HandleFunc("/api/v1/services/{key}/events", secureIfNecessary(securityConfig, GzipHandlerFunc(serviceEventsHandler))).Methods("GET")
	router.HandleFunc("/api/v1/services/{key}/uptimes/{duration}/badge.svg", uptimeBadgeHandler).Methods("GET")
	router.HandleFunc("/api/v1/services/{key}/response-times/{duration}/badge.svg", responseTimeBadgeHandler).Methods("GET")
	router.HandleFunc("/api/v1/services/{key}/response-times/{duration}/chart.svg", responseTimeChartHandler).Methods("GET")
	// SPA
	router.HandleFunc("/services/{service}", spaHandler).Methods("GET")
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

// serviceStatusesHandler handles requests to retrieve all service statuses
// Due to the size of the response, this function leverages a cache.
// Must not be wrapped by GzipHandler
func serviceStatusesHandler(writer http.ResponseWriter, r *http.Request) {
	page, pageSize := extractPageAndPageSizeFromRequest(r)
	gzipped := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
	var exists bool
	var value interface{}
	if gzipped {
		writer.Header().Set("Content-Encoding", "gzip")
		value, exists = cache.Get(fmt.Sprintf("service-status-%d-%d-gzipped", page, pageSize))
	} else {
		value, exists = cache.Get(fmt.Sprintf("service-status-%d-%d", page, pageSize))
	}
	var data []byte
	if !exists {
		var err error
		buffer := &bytes.Buffer{}
		gzipWriter := gzip.NewWriter(buffer)
		data, err = json.Marshal(storage.Get().GetAllServiceStatuses(paging.NewServiceStatusParams().WithResults(page, pageSize)))
		if err != nil {
			log.Printf("[controller][serviceStatusesHandler] Unable to marshal object to JSON: %s", err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("Unable to marshal object to JSON"))
			return
		}
		_, _ = gzipWriter.Write(data)
		_ = gzipWriter.Close()
		gzippedData := buffer.Bytes()
		cache.SetWithTTL(fmt.Sprintf("service-status-%d-%d", page, pageSize), data, cacheTTL)
		cache.SetWithTTL(fmt.Sprintf("service-status-%d-%d-gzipped", page, pageSize), gzippedData, cacheTTL)
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
	page, pageSize := extractPageAndPageSizeFromRequest(r)
	vars := mux.Vars(r)
	serviceStatus := storage.Get().GetServiceStatusByKey(vars["key"], paging.NewServiceStatusParams().WithResults(page, pageSize).WithEvents(1, common.MaximumNumberOfEvents))
	if serviceStatus == nil {
		log.Printf("[controller][serviceStatusHandler] Service with key=%s not found", vars["key"])
		writer.WriteHeader(http.StatusNotFound)
		_, _ = writer.Write([]byte("not found"))
		return
	}
	uptime7Days, _ := storage.Get().GetUptimeByKey(vars["key"], time.Now().Add(-24*7*time.Hour), time.Now())
	uptime24Hours, _ := storage.Get().GetUptimeByKey(vars["key"], time.Now().Add(-24*time.Hour), time.Now())
	uptime1Hour, _ := storage.Get().GetUptimeByKey(vars["key"], time.Now().Add(-time.Hour), time.Now())
	data := map[string]interface{}{
		"serviceStatus": serviceStatus,
		// The following fields, while present on core.ServiceStatus, are annotated to remain hidden so that we can
		// expose only the necessary data on /api/v1/statuses.
		// Since the /api/v1/statuses/{key} endpoint does need this data, however, we explicitly expose it here
		"events": serviceStatus.Events,
		// TODO: remove this in v3.0.0. Not used by front-end, only used for API. Left here for v2.x.x backward compatibility
		"uptime": map[string]float64{
			"7d":  uptime7Days,
			"24h": uptime24Hours,
			"1h":  uptime1Hour,
		},
	}
	output, err := json.Marshal(data)
	if err != nil {
		log.Printf("[controller][serviceStatusHandler] Unable to marshal object to JSON: %s", err.Error())
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte("unable to marshal object to JSON"))
		return
	}
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(output)
}

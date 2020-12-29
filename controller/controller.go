package controller

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/security"
	"github.com/TwinProduction/gatus/storage"
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
func Handle(resultGetter storage.ResultGetter) {
	cfg := config.Get()
	router := CreateRouter(cfg, resultGetter)
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
func CreateRouter(cfg *config.Config, resultGetter storage.ResultGetter) *mux.Router {
	router := mux.NewRouter()
	statusesHandler := func(writer http.ResponseWriter, r *http.Request) {
		serviceStatusesHandler(writer, r, resultGetter)
	}
	if cfg.Security != nil && cfg.Security.IsValid() {
		statusesHandler = security.Handler(statusesHandler, cfg.Security)
	}

	badgeHandler := func(writer http.ResponseWriter, r *http.Request) {
		badgeHandler(writer, r, resultGetter)
	}

	router.HandleFunc("/favicon.ico", favIconHandler).Methods("GET") // favicon needs to be always served from the root
	router.HandleFunc(cfg.Web.PrependWithContextRoot("/api/v1/statuses"), statusesHandler).Methods("GET")
	router.HandleFunc(cfg.Web.PrependWithContextRoot("/api/v1/badges/uptime/{duration}/{identifier}"), badgeHandler).Methods("GET")
	router.HandleFunc(cfg.Web.PrependWithContextRoot("/health"), healthHandler).Methods("GET")
	router.PathPrefix(cfg.Web.ContextRoot).Handler(GzipHandler(http.StripPrefix(cfg.Web.ContextRoot, http.FileServer(http.Dir("./static")))))
	if cfg.Metrics {
		router.Handle(cfg.Web.PrependWithContextRoot("/metrics"), promhttp.Handler()).Methods("GET")
	}
	return router
}

func serviceStatusesHandler(writer http.ResponseWriter, r *http.Request, resultGetter storage.ResultGetter) {
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

		results, err := resultGetter.GetAll()
		if err != nil {
			log.Printf("[main][serviceStatusesHandler] Unable to get results: %s", err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("Unable to get results"))
			return
		}

		data, err := json.Marshal(results)
		if err != nil {
			log.Printf("[main][serviceStatusesHandler] Unable to marshal object to JSON: %s", err.Error())
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

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("{\"status\":\"UP\"}"))
}

// favIconHandler handles requests for /favicon.ico
func favIconHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "./static/favicon.ico")
}

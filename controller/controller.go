package controller

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/security"
	"github.com/TwinProduction/gatus/watchdog"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	cacheTTL = 10 * time.Second
)

var (
	cachedServiceStatuses          []byte
	cachedServiceStatusesGzipped   []byte
	cachedServiceStatusesTimestamp time.Time
)

// Handle creates the router and starts the server
func Handle() {
	cfg := config.Get()
	router := CreateRouter(cfg)
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
	statusesHandler := serviceStatusesHandler
	if cfg.Security != nil && cfg.Security.IsValid() {
		statusesHandler = security.Handler(serviceStatusesHandler, cfg.Security)
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

func serviceStatusesHandler(writer http.ResponseWriter, r *http.Request) {
	if isExpired := cachedServiceStatusesTimestamp.IsZero() || time.Now().Sub(cachedServiceStatusesTimestamp) > cacheTTL; isExpired {
		buffer := &bytes.Buffer{}
		gzipWriter := gzip.NewWriter(buffer)
		data, err := watchdog.GetJSONEncodedServiceStatuses()
		if err != nil {
			log.Printf("[main][serviceStatusesHandler] Unable to marshal object to JSON: %s", err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("Unable to marshal object to JSON"))
			return
		}
		gzipWriter.Write(data)
		gzipWriter.Close()
		cachedServiceStatuses = data
		cachedServiceStatusesGzipped = buffer.Bytes()
		cachedServiceStatusesTimestamp = time.Now()
	}
	var data []byte
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		writer.Header().Set("Content-Encoding", "gzip")
		data = cachedServiceStatusesGzipped
	} else {
		data = cachedServiceStatuses
	}
	writer.Header().Add("Content-type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(data)
}

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Add("Content-type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("{\"status\":\"UP\"}"))
}

// favIconHandler handles requests for /favicon.ico
func favIconHandler(writer http.ResponseWriter, request *http.Request) {
	http.ServeFile(writer, request, "./static/favicon.ico")
}

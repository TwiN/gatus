package main

import (
	"bytes"
	"compress/gzip"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/security"
	"github.com/TwinProduction/gatus/watchdog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const cacheTTL = 10 * time.Second

var (
	cachedServiceStatuses          []byte
	cachedServiceStatusesGzipped   []byte
	cachedServiceStatusesTimestamp time.Time
)

func main() {
	cfg := loadConfiguration()
	statusesHandler := serviceStatusesHandler
	if cfg.Security != nil && cfg.Security.IsValid() {
		statusesHandler = security.Handler(serviceStatusesHandler, cfg.Security)
	}
	http.HandleFunc("/favicon.ico", favIconHandler) // favicon needs to be always served from the root
	http.HandleFunc(cfg.Web.PrependWithContextRoot("/api/v1/statuses"), statusesHandler)
	http.HandleFunc(cfg.Web.PrependWithContextRoot("/health"), healthHandler)
	http.Handle(cfg.Web.ContextRoot, GzipHandler(http.StripPrefix(cfg.Web.ContextRoot, http.FileServer(http.Dir("./static")))))

	if cfg.Metrics {
		http.Handle(cfg.Web.PrependWithContextRoot("/metrics"), promhttp.Handler())
	}
	log.Printf("[main][main] Listening on %s%s\n", cfg.Web.SocketAddress(), cfg.Web.ContextRoot)
	go watchdog.Monitor(cfg)
	log.Fatal(http.ListenAndServe(cfg.Web.SocketAddress(), nil))
}

func loadConfiguration() *config.Config {
	var err error
	customConfigFile := os.Getenv("GATUS_CONFIG_FILE")
	if len(customConfigFile) > 0 {
		err = config.Load(customConfigFile)
	} else {
		err = config.LoadDefaultConfiguration()
	}
	if err != nil {
		panic(err)
	}
	return config.Get()
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

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
	"github.com/TwinProduction/gatus/discovery"
	"github.com/TwinProduction/gatus/security"
	"github.com/TwinProduction/gatus/watchdog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const cacheTTL = 10 * time.Second

var (
	cachedServiceResults          []byte
	cachedServiceResultsGzipped   []byte
	cachedServiceResultsTimestamp time.Time
)

func main() {
	cfg := loadConfiguration()
	if cfg.AutoDiscoverK8SServices {
		discoveredServices := discovery.GetServices(cfg)
		cfg.Services = append(cfg.Services, discoveredServices...)
	}
	resultsHandler := serviceResultsHandler
	if cfg.Security != nil && cfg.Security.IsValid() {
		resultsHandler = security.Handler(serviceResultsHandler, cfg.Security)
	}
	http.HandleFunc("/api/v1/results", resultsHandler)
	http.HandleFunc("/health", healthHandler)
	http.Handle("/", GzipHandler(http.FileServer(http.Dir("./static"))))
	if cfg.Metrics {
		http.Handle("/metrics", promhttp.Handler())
	}
	log.Println("[main][main] Listening on port 8080")
	go watchdog.Monitor(cfg)
	log.Fatal(http.ListenAndServe(":8080", nil))
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

func serviceResultsHandler(writer http.ResponseWriter, r *http.Request) {
	if isExpired := cachedServiceResultsTimestamp.IsZero() || time.Now().Sub(cachedServiceResultsTimestamp) > cacheTTL; isExpired {
		buffer := &bytes.Buffer{}
		gzipWriter := gzip.NewWriter(buffer)
		data, err := watchdog.GetJSONEncodedServiceResults()
		if err != nil {
			log.Printf("[main][serviceResultsHandler] Unable to marshal object to JSON: %s", err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("Unable to marshal object to JSON"))
			return
		}
		gzipWriter.Write(data)
		gzipWriter.Close()
		cachedServiceResults = data
		cachedServiceResultsGzipped = buffer.Bytes()
		cachedServiceResultsTimestamp = time.Now()
	}
	var data []byte
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		writer.Header().Set("Content-Encoding", "gzip")
		data = cachedServiceResultsGzipped
	} else {
		data = cachedServiceResults
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

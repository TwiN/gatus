package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/watchdog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const CacheTTL = 10 * time.Second

var (
	cachedServiceResults          []byte
	cachedServiceResultsGzipped   []byte
	cachedServiceResultsTimestamp time.Time
)

func main() {
	cfg := loadConfiguration()
	http.HandleFunc("/api/v1/results", serviceResultsHandler)
	http.HandleFunc("/health", healthHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))
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
	if isExpired := cachedServiceResultsTimestamp.IsZero() || time.Now().Sub(cachedServiceResultsTimestamp) > CacheTTL; isExpired {
		buffer := &bytes.Buffer{}
		gzipWriter := gzip.NewWriter(buffer)
		serviceResults := watchdog.GetServiceResults()
		data, err := json.Marshal(serviceResults)
		if err != nil {
			log.Printf("[main][serviceResultsHandler] Unable to marshall object to JSON: %s", err.Error())
			writer.WriteHeader(http.StatusInternalServerError)
			_, _ = writer.Write([]byte("Unable to marshall object to JSON"))
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

package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/security"
	"github.com/TwinProduction/gatus/watchdog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const cacheTTL = 10 * time.Second

var (
	cachedServiceResults          []byte
	cachedServiceResultsGzipped   []byte
	cachedServiceResultsTimestamp time.Time
	port                          int
	host                          string
)

func init() {
	// Customizing priority:
	// (1) command line parameters will be preferred over
	// (2) environment variables will be preferred over
	// (3) application defaults

	// set defaults for the case that neither an environment variable nor a
	// command line parameter is passed
	var defaultHost = ""
	var defaultPort = 8080

	// assume set if the is a valid port number
	if p, err := strconv.Atoi(os.Getenv("GATUS_CONFIG_PORT")); err == nil && p > 0 {
		defaultPort = p
	}

	// explicitly asked if the user has set a the environment variable to
	// blank / empty in order to allow listening on all interfaces
	if h, set := os.LookupEnv("GATUS_CONFIG_HOST"); set == true {
		defaultHost = h
	}

	flag.IntVar(&port, "port", defaultPort, "port to listen (default: 8080)")
	flag.IntVar(&port, "p", defaultPort, "port to listen (default: 8080 ; shorthand)")
	flag.StringVar(&host, "host", defaultHost, "host to listen on (default all interfaces on host)")
	flag.StringVar(&host, "h", defaultHost, "host to listen on (default all interfaces on host; shorthand)")
}

func main() {
	flag.Parse()

	cfg := loadConfiguration()
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

	log.Printf("[main][main] Listening on %s:%d", host, port)
	go watchdog.Monitor(cfg)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("%s:%d", host, port), nil))
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

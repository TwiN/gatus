package main

import (
	"encoding/json"
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/watchdog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os"
)

func main() {
	cfg := loadConfiguration()
	go watchdog.Monitor(cfg)
	http.HandleFunc("/api/v1/results", serviceResultsHandler)
	http.HandleFunc("/health", healthHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	if cfg.Metrics {
		http.Handle("/metrics", promhttp.Handler())
	}
	log.Println("[main][main] Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func loadConfiguration() *config.Config {
	args := os.Args
	var err error
	if len(args) == 2 {
		err = config.Load(args[1])
	} else {
		err = config.LoadDefaultConfiguration()
	}
	if err != nil {
		panic(err)
	}
	return config.Get()
}

func serviceResultsHandler(writer http.ResponseWriter, _ *http.Request) {
	serviceResults := watchdog.GetServiceResults()
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(structToJsonBytes(serviceResults))
}

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(structToJsonBytes(&core.HealthStatus{Status: "UP"}))
}

func structToJsonBytes(obj interface{}) []byte {
	bytes, err := json.Marshal(obj)
	if err != nil {
		log.Printf("[main][structToJsonBytes] Unable to marshall object to JSON: %s", err.Error())
	}
	return bytes
}

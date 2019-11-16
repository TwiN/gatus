package main

import (
	"encoding/json"
	"github.com/TwinProduction/gatus/config"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/watchdog"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
)

func main() {
	go watchdog.Monitor()
	http.HandleFunc("/api/v1/results", serviceResultsHandler)
	http.HandleFunc("/health", healthHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	if config.Get().Metrics {
		http.Handle("/metrics", promhttp.Handler())
	}
	log.Println("[main][main] Listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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

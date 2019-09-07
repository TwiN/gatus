package main

import (
	"encoding/json"
	"github.com/TwinProduction/gatus/core"
	"github.com/TwinProduction/gatus/watchdog"
	"log"
	"net/http"
)

func main() {
	go watchdog.Monitor()
	http.HandleFunc("/api/v1/results", serviceResultsHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", indexHandler)
	log.Println("[main][main] Listening on port 80")
	log.Fatal(http.ListenAndServe(":80", nil))
}

func serviceResultsHandler(writer http.ResponseWriter, request *http.Request) {
	serviceResults := watchdog.GetServiceResults()
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(structToJsonBytes(serviceResults))
}

func indexHandler(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusNotImplemented)
	_, _ = writer.Write(structToJsonBytes(&core.ServerMessage{Error: true, Message: "Not implemented yet"}))
}

func healthHandler(writer http.ResponseWriter, request *http.Request) {
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

package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/TwinProduction/gatus/v3/storage"
	"github.com/TwinProduction/gatus/v3/storage/store/common"
	"github.com/TwinProduction/gatus/v3/storage/store/common/paging"
	"github.com/TwinProduction/gocache"
	"github.com/gorilla/mux"
)

const (
	cacheTTL = 10 * time.Second
)

var (
	cache = gocache.NewCache().WithMaxSize(100).WithEvictionPolicy(gocache.FirstInFirstOut)
)

// ServiceStatuses handles requests to retrieve all service statuses
// Due to the size of the response, this function leverages a cache.
// Must not be wrapped by GzipHandler
func ServiceStatuses(writer http.ResponseWriter, r *http.Request) {
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
		serviceStatuses, err := storage.Get().GetAllServiceStatuses(paging.NewServiceStatusParams().WithResults(page, pageSize))
		if err != nil {
			log.Printf("[handler][ServiceStatuses] Failed to retrieve service statuses: %s", err.Error())
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err = json.Marshal(serviceStatuses)
		if err != nil {
			log.Printf("[handler][ServiceStatuses] Unable to marshal object to JSON: %s", err.Error())
			http.Error(writer, "unable to marshal object to JSON", http.StatusInternalServerError)
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

// ServiceStatus retrieves a single ServiceStatus by group name and service name
func ServiceStatus(writer http.ResponseWriter, r *http.Request) {
	page, pageSize := extractPageAndPageSizeFromRequest(r)
	vars := mux.Vars(r)
	serviceStatus, err := storage.Get().GetServiceStatusByKey(vars["key"], paging.NewServiceStatusParams().WithResults(page, pageSize).WithEvents(1, common.MaximumNumberOfEvents))
	if err != nil {
		if err == common.ErrServiceNotFound {
			http.Error(writer, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("[handler][ServiceStatus] Failed to retrieve service status: %s", err.Error())
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	if serviceStatus == nil {
		log.Printf("[handler][ServiceStatus] Service with key=%s not found", vars["key"])
		http.Error(writer, "not found", http.StatusNotFound)
		return
	}
	output, err := json.Marshal(serviceStatus)
	if err != nil {
		log.Printf("[handler][ServiceStatus] Unable to marshal object to JSON: %s", err.Error())
		http.Error(writer, "unable to marshal object to JSON", http.StatusInternalServerError)
		return
	}
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(output)
}

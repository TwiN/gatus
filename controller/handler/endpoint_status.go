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

	"github.com/TwiN/gatus/v3/storage/store"
	"github.com/TwiN/gatus/v3/storage/store/common"
	"github.com/TwiN/gatus/v3/storage/store/common/paging"
	"github.com/TwiN/gocache/v2"
	"github.com/gorilla/mux"
)

const (
	cacheTTL = 10 * time.Second
)

var (
	cache = gocache.NewCache().WithMaxSize(100).WithEvictionPolicy(gocache.FirstInFirstOut)
)

// EndpointStatuses handles requests to retrieve all EndpointStatus
// Due to the size of the response, this function leverages a cache.
// Must not be wrapped by GzipHandler
func EndpointStatuses(writer http.ResponseWriter, r *http.Request) {
	page, pageSize := extractPageAndPageSizeFromRequest(r)
	gzipped := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
	var exists bool
	var value interface{}
	if gzipped {
		writer.Header().Set("Content-Encoding", "gzip")
		value, exists = cache.Get(fmt.Sprintf("endpoint-status-%d-%d-gzipped", page, pageSize))
	} else {
		value, exists = cache.Get(fmt.Sprintf("endpoint-status-%d-%d", page, pageSize))
	}
	var data []byte
	if !exists {
		var err error
		buffer := &bytes.Buffer{}
		gzipWriter := gzip.NewWriter(buffer)
		endpointStatuses, err := store.Get().GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(page, pageSize))
		if err != nil {
			log.Printf("[handler][EndpointStatuses] Failed to retrieve endpoint statuses: %s", err.Error())
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err = json.Marshal(endpointStatuses)
		if err != nil {
			log.Printf("[handler][EndpointStatuses] Unable to marshal object to JSON: %s", err.Error())
			http.Error(writer, "unable to marshal object to JSON", http.StatusInternalServerError)
			return
		}
		_, _ = gzipWriter.Write(data)
		_ = gzipWriter.Close()
		gzippedData := buffer.Bytes()
		cache.SetWithTTL(fmt.Sprintf("endpoint-status-%d-%d", page, pageSize), data, cacheTTL)
		cache.SetWithTTL(fmt.Sprintf("endpoint-status-%d-%d-gzipped", page, pageSize), gzippedData, cacheTTL)
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

// EndpointStatus retrieves a single core.EndpointStatus by group and endpoint name
func EndpointStatus(writer http.ResponseWriter, r *http.Request) {
	page, pageSize := extractPageAndPageSizeFromRequest(r)
	vars := mux.Vars(r)
	endpointStatus, err := store.Get().GetEndpointStatusByKey(vars["key"], paging.NewEndpointStatusParams().WithResults(page, pageSize).WithEvents(1, common.MaximumNumberOfEvents))
	if err != nil {
		if err == common.ErrEndpointNotFound {
			http.Error(writer, err.Error(), http.StatusNotFound)
			return
		}
		log.Printf("[handler][EndpointStatus] Failed to retrieve endpoint status: %s", err.Error())
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	if endpointStatus == nil {
		log.Printf("[handler][EndpointStatus] Endpoint with key=%s not found", vars["key"])
		http.Error(writer, "not found", http.StatusNotFound)
		return
	}
	output, err := json.Marshal(endpointStatus)
	if err != nil {
		log.Printf("[handler][EndpointStatus] Unable to marshal object to JSON: %s", err.Error())
		http.Error(writer, "unable to marshal object to JSON", http.StatusInternalServerError)
		return
	}
	writer.Header().Add("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(output)
}

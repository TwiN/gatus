package handler

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
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
func EndpointStatuses(cfg *config.Config) http.HandlerFunc {
	return func(writer http.ResponseWriter, r *http.Request) {
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
			// ALPHA: Retrieve endpoint statuses from remote instances
			if endpointStatusesFromRemote, err := getEndpointStatusesFromRemoteInstances(cfg.Remote); err != nil {
				log.Printf("[handler][EndpointStatuses] Silently failed to retrieve endpoint statuses from remote: %s", err.Error())
			} else if endpointStatusesFromRemote != nil {
				endpointStatuses = append(endpointStatuses, endpointStatusesFromRemote...)
			}
			// Marshal endpoint statuses to JSON
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
}

func getEndpointStatusesFromRemoteInstances(remoteConfig *remote.Config) ([]*core.EndpointStatus, error) {
	if remoteConfig == nil || len(remoteConfig.Instances) == 0 {
		return nil, nil
	}
	var endpointStatusesFromAllRemotes []*core.EndpointStatus
	httpClient := client.GetHTTPClient(remoteConfig.ClientConfig)
	for _, instance := range remoteConfig.Instances {
		response, err := httpClient.Get(instance.URL)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(response.Body)
		if err != nil {
			_ = response.Body.Close()
			log.Printf("[handler][getEndpointStatusesFromRemoteInstances] Silently failed to retrieve endpoint statuses from %s: %s", instance.URL, err.Error())
			continue
		}
		var endpointStatuses []*core.EndpointStatus
		if err = json.Unmarshal(body, &endpointStatuses); err != nil {
			_ = response.Body.Close()
			log.Printf("[handler][getEndpointStatusesFromRemoteInstances] Silently failed to retrieve endpoint statuses from %s: %s", instance.URL, err.Error())
			continue
		}
		_ = response.Body.Close()
		for _, endpointStatus := range endpointStatuses {
			endpointStatus.Name = instance.EndpointPrefix + endpointStatus.Name
		}
		endpointStatusesFromAllRemotes = append(endpointStatusesFromAllRemotes, endpointStatuses...)
	}
	return endpointStatusesFromAllRemotes, nil
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

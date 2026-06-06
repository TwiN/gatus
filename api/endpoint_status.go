package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/sync/errgroup"
)

// EndpointStatuses handles requests to retrieve all EndpointStatus
// Due to how intensive this operation can be on the storage, this function leverages a cache.
func EndpointStatuses(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, cfg.Storage.MaximumNumberOfResults)
		value, exists := cache.Get(fmt.Sprintf("endpoint-status-%d-%d", page, pageSize))

		var data []byte
		var err error

		if !exists {
			endpointStatuses := make([]*endpoint.Status, 0)
			wg := errgroup.Group{}
			sliceMutex := sync.Mutex{}

			wg.Go(func() error {
				es, err := store.Get().GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(page, pageSize))
				if err != nil {
					return fmt.Errorf("[api.EndpointStatuses] failed to retrieve endpoint statuses: %w", err)
				}

				sliceMutex.Lock()
				endpointStatuses = append(endpointStatuses, es...)
				sliceMutex.Unlock()

				return nil
			})

			if cfg.Remote != nil && len(cfg.Remote.Instances) > 0 {
				httpClient := client.GetHTTPClient(cfg.Remote.ClientConfig)

				// ALPHA: Retrieve endpoint statuses from remote instances
				for _, instance := range cfg.Remote.Instances {
					wg.Go(func() error {
						start := time.Now()

						response, err := httpClient.Get(instance.URL)
						responseCode := 0

						// just in case response is nil from an error
						if response != nil {
							responseCode = response.StatusCode
							defer response.Body.Close()
						}

						logr.Infof("[api.EndpointStatuses] remote=%s; status=%d; duration=%s", instance.URL, responseCode, time.Since(start).Truncate(100 * time.Nanosecond))

						if err != nil {
							// log but dont return to the errgroup to avoid 500 errors for remote instance failures
							logr.Errorf("[api.EndpointStatuses] Failed to retrieve endpoint statuses from %s: %s", instance.URL, err.Error())
							return nil
						}

						var es []*endpoint.Status
						if err = json.NewDecoder(response.Body).Decode(&es); err != nil {
							// log but dont return to the errgroup to avoid 500 errors for remote instance failures
							logr.Errorf("[api.EndpointStatuses] Failed to decode endpoint statuses from %s: %s", instance.URL, err.Error())
							return nil
						}

						for _, endpointStatus := range es {
							endpointStatus.Name = instance.EndpointPrefix + endpointStatus.Name
						}

						sliceMutex.Lock()
						endpointStatuses = append(endpointStatuses, es...)
						sliceMutex.Unlock()

						return nil
					})
				}
			}

			if err := wg.Wait(); err != nil {
				return c.Status(500).SendString(fmt.Sprintf("unable to retrieve endpoint statuses: %s", err.Error()))
			}

			// Marshal endpoint statuses to JSON
			data, err = json.Marshal(endpointStatuses)
			if err != nil {
				logr.Errorf("[api.EndpointStatuses] Unable to marshal object to JSON: %s", err.Error())
				return c.Status(500).SendString("unable to marshal object to JSON")
			}

			cache.SetWithTTL(fmt.Sprintf("endpoint-status-%d-%d", page, pageSize), data, cacheTTL)
		} else {
			data = value.([]byte)
		}

		c.Set("Content-Type", "application/json")
		return c.Status(200).Send(data)
	}
}

// EndpointStatus retrieves a single endpoint.Status by group and endpoint name
func EndpointStatus(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, cfg.Storage.MaximumNumberOfResults)
		key, err := url.QueryUnescape(c.Params("key"))
		if err != nil {
			logr.Errorf("[api.EndpointStatus] Failed to decode key: %s", err.Error())
			return c.Status(400).SendString("invalid key encoding")
		}
		endpointStatus, err := store.Get().GetEndpointStatusByKey(key, paging.NewEndpointStatusParams().WithResults(page, pageSize).WithEvents(1, cfg.Storage.MaximumNumberOfEvents))
		if err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return c.Status(404).SendString(err.Error())
			}
			logr.Errorf("[api.EndpointStatus] Failed to retrieve endpoint status: %s", err.Error())
			return c.Status(500).SendString(err.Error())
		}
		if endpointStatus == nil { // XXX: is this check necessary?
			logr.Errorf("[api.EndpointStatus] Endpoint with key=%s not found", key)
			return c.Status(404).SendString("not found")
		}
		output, err := json.Marshal(endpointStatus)
		if err != nil {
			logr.Errorf("[api.EndpointStatus] Unable to marshal object to JSON: %s", err.Error())
			return c.Status(500).SendString("unable to marshal object to JSON")
		}
		c.Set("Content-Type", "application/json")
		return c.Status(200).Send(output)
	}
}

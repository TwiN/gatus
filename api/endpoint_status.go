package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/gatus/v5/core"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/gofiber/fiber/v2"
)

// EndpointStatuses handles requests to retrieve all EndpointStatus
// Due to how intensive this operation can be on the storage, this function leverages a cache.
func EndpointStatuses(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c)
		value, exists := cache.Get(fmt.Sprintf("endpoint-status-%d-%d", page, pageSize))
		var data []byte
		if !exists {
			endpointStatuses, err := store.Get().GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(page, pageSize))
			if err != nil {
				log.Printf("[api][EndpointStatuses] Failed to retrieve endpoint statuses: %s", err.Error())
				return c.Status(500).SendString(err.Error())
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
				log.Printf("[api][EndpointStatuses] Unable to marshal object to JSON: %s", err.Error())
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
func EndpointStatus(c *fiber.Ctx) error {
	page, pageSize := extractPageAndPageSizeFromRequest(c)
	endpointStatus, err := store.Get().GetEndpointStatusByKey(c.Params("key"), paging.NewEndpointStatusParams().WithResults(page, pageSize).WithEvents(1, common.MaximumNumberOfEvents))
	if err != nil {
		if err == common.ErrEndpointNotFound {
			return c.Status(404).SendString(err.Error())
		}
		log.Printf("[api][EndpointStatus] Failed to retrieve endpoint status: %s", err.Error())
		return c.Status(500).SendString(err.Error())
	}
	if endpointStatus == nil { // XXX: is this check necessary?
		log.Printf("[api][EndpointStatus] Endpoint with key=%s not found", c.Params("key"))
		return c.Status(404).SendString("not found")
	}
	output, err := json.Marshal(endpointStatus)
	if err != nil {
		log.Printf("[api][EndpointStatus] Unable to marshal object to JSON: %s", err.Error())
		return c.Status(500).SendString("unable to marshal object to JSON")
	}
	c.Set("Content-Type", "application/json")
	return c.Status(200).Send(output)
}

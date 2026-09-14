package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/gatus/v5/storage/store"
	"github.com/TwiN/gatus/v5/storage/store/common"
	"github.com/TwiN/gatus/v5/storage/store/common/paging"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

// EndpointStatuses handles requests to retrieve all EndpointStatus
// Due to how intensive this operation can be on the storage, this function leverages a cache.
func EndpointStatuses(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, cfg.Storage.MaximumNumberOfResults)
		value, exists := cache.Get(fmt.Sprintf("endpoint-status-%d-%d", page, pageSize))
		var data []byte
		if !exists {
			endpointStatuses, err := store.Get().GetAllEndpointStatuses(paging.NewEndpointStatusParams().WithResults(page, pageSize))
			if err != nil {
				logr.Errorf("[api.EndpointStatuses] Failed to retrieve endpoint statuses: %s", err.Error())
				return c.Status(500).SendString(err.Error())
			}
			// Annotate endpoint statuses with their current suspended state from the configuration, and add
			// placeholder statuses for suspended endpoints that don't have any persisted results yet, so that
			// they're still visible (and filterable) in the UI.
			endpointStatuses = applySuspendedStateToEndpointStatuses(cfg, endpointStatuses)
			// ALPHA: Retrieve endpoint statuses from remote instances
			if endpointStatusesFromRemote, err := getEndpointStatusesFromRemoteInstances(cfg.Remote); err != nil {
				logr.Errorf("[handler.EndpointStatuses] Silently failed to retrieve endpoint statuses from remote: %s", err.Error())
			} else if endpointStatusesFromRemote != nil {
				endpointStatuses = append(endpointStatuses, endpointStatusesFromRemote...)
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

// applySuspendedStateToEndpointStatuses sets the Suspended field on every endpoint.Status that has a matching
// suspended endpoint (or external endpoint) in the configuration, and appends a placeholder endpoint.Status for
// any suspended endpoint that doesn't have one yet (e.g. it was suspended before ever being checked).
func applySuspendedStateToEndpointStatuses(cfg *config.Config, endpointStatuses []*endpoint.Status) []*endpoint.Status {
	suspendedKeys := suspendedEndpointKeys(cfg)
	if len(suspendedKeys) == 0 {
		return endpointStatuses
	}
	seenKeys := make(map[string]bool, len(endpointStatuses))
	for _, status := range endpointStatuses {
		seenKeys[status.Key] = true
		status.Suspended = suspendedKeys[status.Key]
	}
	for _, ep := range cfg.Endpoints {
		if ep.IsSuspended() && !seenKeys[ep.Key()] {
			placeholder := endpoint.NewStatus(ep.Group, ep.Name)
			placeholder.Suspended = true
			endpointStatuses = append(endpointStatuses, placeholder)
			seenKeys[ep.Key()] = true
		}
	}
	for _, ee := range cfg.ExternalEndpoints {
		if ee.IsSuspended() && !seenKeys[ee.Key()] {
			placeholder := endpoint.NewStatus(ee.Group, ee.Name)
			placeholder.Suspended = true
			endpointStatuses = append(endpointStatuses, placeholder)
			seenKeys[ee.Key()] = true
		}
	}
	return endpointStatuses
}

// suspendedEndpointKeys returns the set of keys of all suspended endpoints and external endpoints in the
// configuration.
func suspendedEndpointKeys(cfg *config.Config) map[string]bool {
	suspendedKeys := make(map[string]bool)
	for _, ep := range cfg.Endpoints {
		if ep.IsSuspended() {
			suspendedKeys[ep.Key()] = true
		}
	}
	for _, ee := range cfg.ExternalEndpoints {
		if ee.IsSuspended() {
			suspendedKeys[ee.Key()] = true
		}
	}
	return suspendedKeys
}

// findSuspendedEndpointInConfig looks up a suspended endpoint or external endpoint by key, returning its group and
// name if found.
func findSuspendedEndpointInConfig(cfg *config.Config, key string) (group, name string, found bool) {
	for _, ep := range cfg.Endpoints {
		if ep.IsSuspended() && ep.Key() == key {
			return ep.Group, ep.Name, true
		}
	}
	for _, ee := range cfg.ExternalEndpoints {
		if ee.IsSuspended() && ee.Key() == key {
			return ee.Group, ee.Name, true
		}
	}
	return "", "", false
}

func getEndpointStatusesFromRemoteInstances(remoteConfig *remote.Config) ([]*endpoint.Status, error) {
	if remoteConfig == nil || len(remoteConfig.Instances) == 0 {
		return nil, nil
	}
	var endpointStatusesFromAllRemotes []*endpoint.Status
	httpClient := client.GetHTTPClient(remoteConfig.ClientConfig)
	for _, instance := range remoteConfig.Instances {
		response, err := httpClient.Get(instance.URL)
		if err != nil {
			// Log the error but continue with other instances
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to retrieve endpoint statuses from %s: %s", instance.URL, err.Error())
			continue
		}
		var endpointStatuses []*endpoint.Status
		if err = json.NewDecoder(response.Body).Decode(&endpointStatuses); err != nil {
			_ = response.Body.Close()
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to decode endpoint statuses from %s: %s", instance.URL, err.Error())
			continue
		}
		_ = response.Body.Close()
		for _, endpointStatus := range endpointStatuses {
			endpointStatus.Name = instance.EndpointPrefix + endpointStatus.Name
		}
		endpointStatusesFromAllRemotes = append(endpointStatusesFromAllRemotes, endpointStatuses...)
	}
	// Only return nil, error if no remote instances were successfully processed
	if len(endpointStatusesFromAllRemotes) == 0 && remoteConfig.Instances != nil {
		return nil, fmt.Errorf("failed to retrieve endpoint statuses from all remote instances")
	}
	return endpointStatusesFromAllRemotes, nil
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
				// The endpoint may not have any persisted results yet because it's suspended: fall back to a
				// placeholder built from the configuration so it's still reachable from the UI.
				if group, name, found := findSuspendedEndpointInConfig(cfg, key); found {
					endpointStatus = endpoint.NewStatus(group, name)
					endpointStatus.Suspended = true
				} else {
					return c.Status(404).SendString(err.Error())
				}
			} else {
				logr.Errorf("[api.EndpointStatus] Failed to retrieve endpoint status: %s", err.Error())
				return c.Status(500).SendString(err.Error())
			}
		}
		if endpointStatus == nil { // XXX: is this check necessary?
			logr.Errorf("[api.EndpointStatus] Endpoint with key=%s not found", key)
			return c.Status(404).SendString("not found")
		}
		endpointStatus.Suspended = suspendedEndpointKeys(cfg)[endpointStatus.Key]
		output, err := json.Marshal(endpointStatus)
		if err != nil {
			logr.Errorf("[api.EndpointStatus] Unable to marshal object to JSON: %s", err.Error())
			return c.Status(500).SendString("unable to marshal object to JSON")
		}
		c.Set("Content-Type", "application/json")
		return c.Status(200).Send(output)
	}
}

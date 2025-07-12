package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"

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

// EndpointStatuses handles paginated, filtered retrieval of endpoint statuses
func EndpointStatuses(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract pagination params
		page, pageSize := extractPageAndPageSizeFromRequest(c, cfg.Storage.MaximumNumberOfResults)

		// Read filter query params
		searchTerm := strings.ToLower(c.Query("search"))
		statusFilter := strings.ToLower(c.Query("status")) // "success" or "error"

		// Get all statuses without slicing to apply filtering
		allList, err := store.Get().GetAllEndpointStatuses(
			paging.NewEndpointStatusParams().WithResults(1, math.MaxInt32),
		)
		if err != nil {
			logr.Errorf("[api.EndpointStatuses] Failed to retrieve endpoint statuses: %s", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		// Append remote statuses if any
		if remoteList, err := getEndpointStatusesFromRemoteInstances(cfg.Remote); err != nil {
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] silently failed: %s", err)
		} else if remoteList != nil {
			allList = append(allList, remoteList...)
		}

		// Apply filters
		filtered := make([]*endpoint.Status, 0, len(allList))
		for _, es := range allList {
			if searchTerm != "" && !strings.Contains(strings.ToLower(es.Name), searchTerm) {
				continue
			}
			// Determine latest check success
			latestOK := true
			if len(es.Results) > 0 {
				latestOK = es.Results[0].Success
			}
			if statusFilter == "success" && !latestOK {
				continue
			}
			if statusFilter == "error" && latestOK {
				continue
			}
			filtered = append(filtered, es)
		}

		// Compute total count after filtering
		total := len(filtered)

		// Apply pagination slice to filtered list
		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		pageSlice := filtered[start:end]

		// Set headers
		c.Set("X-Total-Count", strconv.Itoa(total))
		c.Set("Content-Type", "application/json")

		// Return paginated, filtered results
		return c.Status(fiber.StatusOK).JSON(pageSlice)
	}
}

// EndpointStatus retrieves a single endpoint.Status by key, with pagination on its events
func EndpointStatus(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, pageSize := extractPageAndPageSizeFromRequest(c, cfg.Storage.MaximumNumberOfResults)
		key, err := url.QueryUnescape(c.Params("key"))
		if err != nil {
			logr.Errorf("[api.EndpointStatus] Invalid key encoding: %s", err)
			return c.Status(fiber.StatusBadRequest).SendString("invalid key encoding")
		}

		endpointStatus, err := store.Get().GetEndpointStatusByKey(
			key,
			paging.NewEndpointStatusParams().WithResults(page, pageSize).WithEvents(1, cfg.Storage.MaximumNumberOfEvents),
		)
		if err != nil {
			if errors.Is(err, common.ErrEndpointNotFound) {
				return c.Status(fiber.StatusNotFound).SendString(err.Error())
			}
			logr.Errorf("[api.EndpointStatus] Failed to retrieve endpoint status: %s", err)
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		if endpointStatus == nil {
			logr.Errorf("[api.EndpointStatus] Endpoint with key=%s not found", key)
			return c.Status(fiber.StatusNotFound).SendString("not found")
		}

		c.Set("Content-Type", "application/json")
		return c.Status(fiber.StatusOK).JSON(endpointStatus)
	}
}

// Helper to fetch from remote instances
func getEndpointStatusesFromRemoteInstances(remoteConfig *remote.Config) ([]*endpoint.Status, error) {
	if remoteConfig == nil || len(remoteConfig.Instances) == 0 {
		return nil, nil
	}
	var all []*endpoint.Status
	client := client.GetHTTPClient(remoteConfig.ClientConfig)
	for _, inst := range remoteConfig.Instances {
		resp, err := client.Get(inst.URL)
		if err != nil {
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] %s: %s", inst.URL, err)
			continue
		}
		var list []*endpoint.Status
		if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
			_ = resp.Body.Close()
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] decode %s: %s", inst.URL, err)
			continue
		}
		_ = resp.Body.Close()
		for _, s := range list {
			s.Name = inst.EndpointPrefix + s.Name
		}
		all = append(all, list...)
	}
	if len(all) == 0 && remoteConfig.Instances != nil {
		return nil, fmt.Errorf("failed to retrieve endpoint statuses from all remote instances")
	}
	return all, nil
}

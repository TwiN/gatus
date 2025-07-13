package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
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
        page, pageSize := extractPageAndPageSizeFromRequest(c, cfg.Storage.MaximumNumberOfResults)

        searchFilter := c.Query("search", "")
        statusFilter := c.Query("status", "")

        cacheKey := fmt.Sprintf("endpoint-status-%d-%d-%s-%s",
            page, pageSize, searchFilter, statusFilter)

        //  Check cache
        if cached, ok := cache.Get(cacheKey); ok {
            if pageData, ok := cached.(struct {
                TotalCount int
                Data       []byte
            }); ok {
                c.Set("Content-Type", "application/json")
                c.Set("X-Total-Count", fmt.Sprintf("%d", pageData.TotalCount))
                return c.Status(200).Send(pageData.Data)
            }
            logr.Warn("[api.EndpointStatuses] Cache contained unexpected type")
        }

        //  Query store
        endpointStatuses, err := store.Get().GetAllEndpointStatuses(
            paging.NewEndpointStatusParams().WithResults(1, cfg.Storage.MaximumNumberOfResults),
        )
        if err != nil {
            logr.Errorf("[api.EndpointStatuses] Failed to retrieve endpoint statuses: %s", err.Error())
            return c.Status(500).SendString(err.Error())
        }

        var filtered []*endpoint.Status
        reverseResults := c.Query("reverse", "") == "true"

        for _, es := range endpointStatuses {
            if searchFilter != "" && !matchesSearch(es, searchFilter) {
                continue
            }
            if statusFilter != "" && !matchesStatus(es, statusFilter) {
                continue
            }

            if reverseResults && len(es.Results) > 1 {
                for i, j := 0, len(es.Results)-1; i < j; i, j = i+1, j-1 {
                    es.Results[i], es.Results[j] = es.Results[j], es.Results[i]
                }
            }

            filtered = append(filtered, es)
        }

        totalCount := len(filtered)

        //  Page endpoints
        start := (page - 1) * pageSize
        end := start + pageSize
        if start >= totalCount {
            filtered = []*endpoint.Status{}
        } else {
            if end > totalCount {
                end = totalCount
            }
            filtered = filtered[start:end]
        }

        //  Page results inside each endpoint
        for _, es := range filtered {
            start := 0
            end := pageSize
            if end > len(es.Results) {
                end = len(es.Results)
            }
            es.Results = es.Results[start:end]
        }

        data, err := json.Marshal(filtered)
        if err != nil {
            logr.Errorf("[api.EndpointStatuses] Unable to marshal JSON: %s", err.Error())
            return c.Status(500).SendString("unable to marshal object to JSON")
        }

        //  Cache JSON + total count
        cache.SetWithTTL(cacheKey, struct {
            TotalCount int
            Data       []byte
        }{
            TotalCount: totalCount,
            Data:       data,
        }, cacheTTL)

        c.Set("Content-Type", "application/json")
        c.Set("X-Total-Count", fmt.Sprintf("%d", totalCount))
        return c.Status(200).Send(data)
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

func matchesSearch(es *endpoint.Status, search string) bool {
    return containsInsensitive(es.Name, search) ||
           containsInsensitive(es.Group, search) ||
           containsInsensitive(es.Key, search)
}

func containsInsensitive(s, sub string) bool {
    return len(s) > 0 && len(sub) > 0 && ( // avoid empty search matching everything
        (len(s) >= len(sub)) && 
        (containsFold(s, sub)))
}

func containsFold(s, substr string) bool {
    return len(substr) == 0 || (len(s) >= len(substr) && 
        (stringContainsFold(s, substr)))
}

func stringContainsFold(s, substr string) bool {
    return len(s) > 0 && len(substr) > 0 && 
           (stringIndexFold(s, substr) >= 0)
}

func stringIndexFold(s, substr string) int {
    return strings.Index(strings.ToLower(s), strings.ToLower(substr))
}

func matchesStatus(es *endpoint.Status, status string) bool {
    if len(es.Results) == 0 {
        return false
    }
    last := es.Results[len(es.Results)-1]
    switch status {
    case "success":
        return last.Success
    case "error":
        return !last.Success
    default:
        return true
    }
}

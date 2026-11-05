package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
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

func getEndpointStatusesFromRemoteInstances(remoteConfig *remote.Config) ([]*endpoint.Status, error) {
	if remoteConfig == nil || len(remoteConfig.Instances) == 0 {
		return nil, nil
	}
	var endpointStatusesFromAllRemotes []*endpoint.Status
	httpClient := client.GetHTTPClient(remoteConfig.ClientConfig)
	remoteCount := 0
	for _, instance := range remoteConfig.Instances {
		remoteCount = remoteCount + 1
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
			endpointStatus.Key = fmt.Sprint("_gatus_remote_", remoteCount, "_", endpointStatus.Key)
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

		if strings.HasPrefix(key, "_gatus_remote_") {
			r := regexp.MustCompile("_gatus_remote_(\\d+)_(.*)")
			regexResult := r.FindStringSubmatch(key)
			if regexResult == nil {
				return c.Status(400).SendString(fmt.Sprint("invalid remote key: ", key))
			}

			remoteNr, err := strconv.Atoi(regexResult[1])
			if err != nil {
				return c.Status(400).SendString(fmt.Sprint("invalid non-numeric requested remote: ", regexResult[1]))
			}
			remoteKey := regexResult[2]

			if remoteNr < 1 || remoteNr > len(cfg.Remote.Instances) {
				return c.Status(404).SendString(fmt.Sprint("requested remote does not exist: ", remoteNr))
			}

			return EndpointStatusFromRemoteInstance(c, cfg, remoteNr, remoteKey, page, pageSize)
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

// EndpointStatusFromRemoteInstance retrieves a single remote endpoint.Status endpoint name
func EndpointStatusFromRemoteInstance(c *fiber.Ctx, cfg *config.Config, remoteNr int, key string, page int, pageSize int) error {
	instance := cfg.Remote.Instances[remoteNr-1]
	remoteConfig := cfg.Remote
	httpClient := client.GetHTTPClient(remoteConfig.ClientConfig)

	// TODO: better integration
	newUrl := fmt.Sprint("/endpoints/", key, "/statuses?page=", page, "&pageSize=", pageSize)
	url := strings.Replace(instance.URL, "/endpoints/statuses", newUrl, 1)

	response, err := httpClient.Get(url)
	if err != nil {
		// Log the error but continue with other instances
		msg := fmt.Sprintf("[api.getEndpointStatusFromRemoteInstance] Failed to retrieve endpoint status for %s from %s: %s", key, url, err.Error())
		return c.Status(500).SendString(msg)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	defer response.Body.Close()

	// If the remote returned an error, return it as-is
	if response.StatusCode != 200 {
		logr.Debugf("Remote endpoint URL %s returned error code %d", url, response.StatusCode)
		return c.Status(response.StatusCode).Send(buf.Bytes())
	}

	var endpointStatus endpoint.Status
	if err = json.NewDecoder(buf).Decode(&endpointStatus); err != nil {
		msg := fmt.Sprintf("[api.getEndpointStatusFromRemoteInstance] Failed to decode endpoint status for %s from %s: %s", key, url, err.Error())
		return c.Status(500).SendString(msg)
	}
	_ = response.Body.Close()

	endpointStatus.Name = instance.EndpointPrefix + endpointStatus.Name
	endpointStatus.Key = fmt.Sprint("_gatus_remote_", remoteNr, "_", endpointStatus.Key)

	output, err := json.Marshal(endpointStatus)
	if err != nil {
		logr.Errorf("[api.EndpointStatusFromRemoteInstance] Unable to marshal object to JSON: %s", err.Error())
		return c.Status(500).SendString("unable to marshal object to JSON")
	}
	c.Set("Content-Type", "application/json")
	return c.Status(200).Send(output)
}

package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

type remoteEndpointTarget struct {
	instanceIndex int
	instance      remote.Instance
	remoteKey     string
}

func forwardHeadersFromContext(c *fiber.Ctx) http.Header {
	headers := make(http.Header)
	if cookie := string(c.Request().Header.Peek("Cookie")); len(cookie) > 0 {
		headers.Set("Cookie", cookie)
	}
	if authorization := string(c.Request().Header.Peek("Authorization")); len(authorization) > 0 {
		headers.Set("Authorization", authorization)
	}
	return headers
}

func remoteTargetsForKey(remoteConfig *remote.Config, key string) []remoteEndpointTarget {
	if remoteConfig == nil || len(remoteConfig.Instances) == 0 {
		return nil
	}
	if instanceIndex, remoteKey, ok := remote.ParsePrefixedKey(key); ok {
		if instanceIndex >= len(remoteConfig.Instances) {
			return nil
		}
		return []remoteEndpointTarget{{
			instanceIndex: instanceIndex,
			instance:      remoteConfig.Instances[instanceIndex],
			remoteKey:     remoteKey,
		}}
	}
	targets := make([]remoteEndpointTarget, 0, len(remoteConfig.Instances))
	for instanceIndex, instance := range remoteConfig.Instances {
		targets = append(targets, remoteEndpointTarget{
			instanceIndex: instanceIndex,
			instance:      instance,
			remoteKey:     key,
		})
	}
	return targets
}

func appendQueryString(requestURL, queryString string) string {
	if len(queryString) == 0 {
		return requestURL
	}
	if strings.Contains(requestURL, "?") {
		return requestURL + "&" + queryString
	}
	return requestURL + "?" + queryString
}

func logRemoteHTTPFailure(scope, requestURL string, response *http.Response, body []byte) {
	if response == nil {
		return
	}
	if response.StatusCode == http.StatusOK {
		return
	}
	snippet := strings.TrimSpace(string(body))
	if len(snippet) > 256 {
		snippet = snippet[:256] + "..."
	}
	logr.Errorf("[%s] Remote request to %s returned %d: %s", scope, requestURL, response.StatusCode, snippet)
}

func proxyRemoteEndpoint(c *fiber.Ctx, cfg *config.Config, key, subPath string) (bool, error) {
	targets := remoteTargetsForKey(cfg.Remote, key)
	if len(targets) == 0 {
		return false, nil
	}

	httpClient := client.GetHTTPClient(cfg.Remote.ClientConfig)
	queryString := string(c.Context().URI().QueryString())
	forwardHeaders := forwardHeadersFromContext(c)

	for _, target := range targets {
		requestURL := appendQueryString(target.instance.BuildEndpointURL(target.remoteKey, subPath), queryString)
		request, err := http.NewRequestWithContext(c.Context(), http.MethodGet, requestURL, http.NoBody)
		if err != nil {
			logr.Errorf("[api.proxyRemoteEndpoint] Failed to create request for %s: %s", requestURL, err.Error())
			continue
		}
		target.instance.ApplyRequestHeaders(request, forwardHeaders)
		response, err := httpClient.Do(request)
		if err != nil {
			logr.Errorf("[api.proxyRemoteEndpoint] Failed to retrieve %s: %s", requestURL, err.Error())
			continue
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			logr.Errorf("[api.proxyRemoteEndpoint] Failed to read response from %s: %s", requestURL, err.Error())
			continue
		}
		if response.StatusCode == http.StatusNotFound {
			continue
		}
		logRemoteHTTPFailure("api.proxyRemoteEndpoint", requestURL, response, body)

		for headerKey, headerValues := range response.Header {
			if len(headerValues) == 0 {
				continue
			}
			switch strings.ToLower(headerKey) {
			case "content-type", "cache-control", "expires":
				c.Set(headerKey, headerValues[0])
			}
		}
		return true, c.Status(response.StatusCode).Send(body)
	}
	return false, nil
}

func applyRemoteEndpointPresentation(status *endpoint.Status, instance remote.Instance, instanceIndex int, requestKey string) {
	status.Name = instance.EndpointPrefix + status.Name
	if remote.IsPrefixedKey(requestKey) {
		status.Key = remote.PrefixedKey(instanceIndex, status.Key)
	}
}

func getEndpointStatusesFromRemoteInstances(remoteConfig *remote.Config, forwardHeaders http.Header) ([]*endpoint.Status, error) {
	if remoteConfig == nil || len(remoteConfig.Instances) == 0 {
		return nil, nil
	}
	var endpointStatusesFromAllRemotes []*endpoint.Status
	httpClient := client.GetHTTPClient(remoteConfig.ClientConfig)
	for instanceIndex, instance := range remoteConfig.Instances {
		request, err := http.NewRequest(http.MethodGet, instance.URL, http.NoBody)
		if err != nil {
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to create request for %s: %s", instance.URL, err.Error())
			continue
		}
		instance.ApplyRequestHeaders(request, forwardHeaders)
		response, err := httpClient.Do(request)
		if err != nil {
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to retrieve endpoint statuses from %s: %s", instance.URL, err.Error())
			continue
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to read response from %s: %s", instance.URL, err.Error())
			continue
		}
		if response.StatusCode != http.StatusOK {
			logRemoteHTTPFailure("api.getEndpointStatusesFromRemoteInstances", instance.URL, response, body)
			continue
		}
		var endpointStatuses []*endpoint.Status
		if err = json.Unmarshal(body, &endpointStatuses); err != nil {
			if len(body) > 0 && body[0] == '<' {
				logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to decode endpoint statuses from %s: response is HTML, not JSON — remote.instances.url must end with /api/v1/endpoints/statuses (not the UI base path)", instance.URL)
			} else {
				logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to decode endpoint statuses from %s: %s", instance.URL, err.Error())
			}
			continue
		}
		for _, endpointStatus := range endpointStatuses {
			applyRemoteEndpointPresentation(endpointStatus, instance, instanceIndex, remote.PrefixedKey(instanceIndex, endpointStatus.Key))
		}
		endpointStatusesFromAllRemotes = append(endpointStatusesFromAllRemotes, endpointStatuses...)
	}
	if len(endpointStatusesFromAllRemotes) == 0 && len(remoteConfig.Instances) > 0 {
		return nil, fmt.Errorf("failed to retrieve endpoint statuses from all remote instances")
	}
	return endpointStatusesFromAllRemotes, nil
}

func proxyRemoteEndpointStatus(c *fiber.Ctx, cfg *config.Config, key string) (bool, error) {
	targets := remoteTargetsForKey(cfg.Remote, key)
	if len(targets) == 0 {
		return false, nil
	}

	httpClient := client.GetHTTPClient(cfg.Remote.ClientConfig)
	queryString := string(c.Context().URI().QueryString())
	forwardHeaders := forwardHeadersFromContext(c)

	for _, target := range targets {
		requestURL := appendQueryString(target.instance.BuildEndpointURL(target.remoteKey, "/statuses"), queryString)
		request, err := http.NewRequestWithContext(c.Context(), http.MethodGet, requestURL, http.NoBody)
		if err != nil {
			logr.Errorf("[api.proxyRemoteEndpointStatus] Failed to create request for %s: %s", requestURL, err.Error())
			continue
		}
		target.instance.ApplyRequestHeaders(request, forwardHeaders)
		response, err := httpClient.Do(request)
		if err != nil {
			logr.Errorf("[api.proxyRemoteEndpointStatus] Failed to retrieve %s: %s", requestURL, err.Error())
			continue
		}
		body, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			logr.Errorf("[api.proxyRemoteEndpointStatus] Failed to read response from %s: %s", requestURL, err.Error())
			continue
		}
		if response.StatusCode == http.StatusNotFound {
			continue
		}
		if response.StatusCode != http.StatusOK {
			logRemoteHTTPFailure("api.proxyRemoteEndpointStatus", requestURL, response, body)
			c.Set("Content-Type", "application/json")
			return true, c.Status(response.StatusCode).Send(body)
		}

		var endpointStatus endpoint.Status
		if err = json.Unmarshal(body, &endpointStatus); err != nil {
			logr.Errorf("[api.proxyRemoteEndpointStatus] Failed to decode endpoint status from %s: %s", requestURL, err.Error())
			continue
		}
		applyRemoteEndpointPresentation(&endpointStatus, target.instance, target.instanceIndex, key)
		output, err := json.Marshal(endpointStatus)
		if err != nil {
			logr.Errorf("[api.proxyRemoteEndpointStatus] Unable to marshal object to JSON: %s", err.Error())
			return true, c.Status(500).SendString("unable to marshal object to JSON")
		}
		c.Set("Content-Type", "application/json")
		return true, c.Status(200).Send(output)
	}
	return false, nil
}

package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/gatus/v5/config/endpoint"
	"github.com/TwiN/gatus/v5/config/remote"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

var errResponseBodyTooLarge = errors.New("remote response body too large")

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
	instanceIndex, remoteKey, ok := remote.ParsePrefixedKey(key)
	if !ok {
		return nil
	}
	if instanceIndex >= len(remoteConfig.Instances) {
		return nil
	}
	return []remoteEndpointTarget{{
		instanceIndex: instanceIndex,
		instance:      remoteConfig.Instances[instanceIndex],
		remoteKey:     remoteKey,
	}}
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

func readBoundedBody(body io.ReadCloser, max int64) ([]byte, error) {
	limited := io.LimitReader(body, max+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, errResponseBodyTooLarge
	}
	return data, nil
}

func closeResponseBody(body io.ReadCloser, scope string) {
	if err := body.Close(); err != nil {
		logr.Debugf("[%s] failed to close response body: %s", scope, err.Error())
	}
}

func contentTypeForSubPath(subPath string) string {
	lower := strings.ToLower(subPath)
	switch {
	case strings.HasSuffix(lower, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lower, ".shields"):
		return "application/json"
	case strings.HasSuffix(lower, "/history"):
		return "application/json"
	default:
		return "text/plain"
	}
}

func isTimeoutError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func sendRemoteGatewayError(c *fiber.Ctx, err error) (bool, error) {
	if isTimeoutError(err) {
		return true, c.Status(http.StatusGatewayTimeout).SendString("remote instance timed out")
	}
	return true, c.Status(http.StatusBadGateway).SendString("failed to retrieve data from remote instance")
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

	maxBody := cfg.Remote.MaxResponseBodySize()
	httpClient := client.GetHTTPClient(cfg.Remote.ClientConfig)
	queryString := string(c.Context().URI().QueryString())
	forwardHeaders := forwardHeadersFromContext(c)

	var sawNotFound bool
	var lastTransportErr error

	for _, target := range targets {
		if isRemoteCircuitOpen(target.instance.URL, cfg.Remote.CircuitBreaker) {
			logr.Debugf("[api.proxyRemoteEndpoint] Skipping %s because its circuit breaker is open", target.instance.URL)
			return true, c.Status(http.StatusServiceUnavailable).SendString("remote instance circuit breaker is open")
		}

		requestURL := appendQueryString(target.instance.BuildEndpointURL(target.remoteKey, subPath), queryString)
		request, err := http.NewRequestWithContext(c.Context(), http.MethodGet, requestURL, http.NoBody)
		if err != nil {
			logr.Errorf("[api.proxyRemoteEndpoint] Failed to create request for %s: %s", requestURL, err.Error())
			continue
		}
		target.instance.ApplyRequestHeaders(request, forwardHeaders)
		response, err := httpClient.Do(request)
		if err != nil {
			recordRemoteCircuitFailure(target.instance.URL, cfg.Remote.CircuitBreaker)
			logr.Errorf("[api.proxyRemoteEndpoint] Failed to retrieve %s: %s", requestURL, err.Error())
			lastTransportErr = err
			continue
		}
		body, err := readBoundedBody(response.Body, maxBody)
		closeResponseBody(response.Body, "api.proxyRemoteEndpoint")
		if errors.Is(err, errResponseBodyTooLarge) {
			recordRemoteCircuitFailure(target.instance.URL, cfg.Remote.CircuitBreaker)
			logr.Errorf("[api.proxyRemoteEndpoint] Response from %s exceeded %d bytes", requestURL, maxBody)
			return true, c.Status(502).SendString("remote response body too large")
		}
		if err != nil {
			recordRemoteCircuitFailure(target.instance.URL, cfg.Remote.CircuitBreaker)
			logr.Errorf("[api.proxyRemoteEndpoint] Failed to read response from %s: %s", requestURL, err.Error())
			lastTransportErr = err
			continue
		}
		recordRemoteCircuitSuccess(target.instance.URL)
		if response.StatusCode == http.StatusNotFound {
			sawNotFound = true
			continue
		}
		logRemoteHTTPFailure("api.proxyRemoteEndpoint", requestURL, response, body)

		// Do not forward Set-Cookie or Content-Type from remote instances.
		for headerKey, headerValues := range response.Header {
			if len(headerValues) == 0 {
				continue
			}
			switch strings.ToLower(headerKey) {
			case "cache-control", "expires":
				c.Set(headerKey, headerValues[0])
			}
		}
		c.Set("Content-Type", contentTypeForSubPath(subPath))
		return true, c.Status(response.StatusCode).Send(body)
	}
	if lastTransportErr != nil {
		return sendRemoteGatewayError(c, lastTransportErr)
	}
	if sawNotFound {
		return false, nil
	}
	return false, nil
}

func applyRemoteEndpointPresentation(status *endpoint.Status, instance remote.Instance, instanceIndex int, requestKey string) {
	status.Name = instance.EndpointPrefix + status.Name
	if remote.IsPrefixedKey(requestKey) {
		status.Key = remote.PrefixedKey(instanceIndex, status.Key)
	}
}

func fetchEndpointStatusesFromRemoteInstance(remoteConfig *remote.Config, instance remote.Instance, instanceIndex int, forwardHeaders http.Header, httpClient *http.Client) []*endpoint.Status {
	if isRemoteCircuitOpen(instance.URL, remoteConfig.CircuitBreaker) {
		logr.Debugf("[api.getEndpointStatusesFromRemoteInstances] Skipping %s because its circuit breaker is open", instance.URL)
		return nil
	}

	maxBody := remoteConfig.MaxResponseBodySize()
	request, err := http.NewRequest(http.MethodGet, instance.URL, http.NoBody)
	if err != nil {
		logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to create request for %s: %s", instance.URL, err.Error())
		return nil
	}
	instance.ApplyRequestHeaders(request, forwardHeaders)
	response, err := httpClient.Do(request)
	if err != nil {
		recordRemoteCircuitFailure(instance.URL, remoteConfig.CircuitBreaker)
		logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to retrieve endpoint statuses from %s: %s", instance.URL, err.Error())
		return nil
	}
	body, err := readBoundedBody(response.Body, maxBody)
	closeResponseBody(response.Body, "api.getEndpointStatusesFromRemoteInstances")
	if errors.Is(err, errResponseBodyTooLarge) {
		recordRemoteCircuitFailure(instance.URL, remoteConfig.CircuitBreaker)
		logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Response from %s exceeded %d bytes", instance.URL, maxBody)
		return nil
	}
	if err != nil {
		recordRemoteCircuitFailure(instance.URL, remoteConfig.CircuitBreaker)
		logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to read response from %s: %s", instance.URL, err.Error())
		return nil
	}
	if response.StatusCode != http.StatusOK {
		logRemoteHTTPFailure("api.getEndpointStatusesFromRemoteInstances", instance.URL, response, body)
		return nil
	}
	recordRemoteCircuitSuccess(instance.URL)
	var endpointStatuses []*endpoint.Status
	if err = json.Unmarshal(body, &endpointStatuses); err != nil {
		if len(body) > 0 && body[0] == '<' {
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to decode endpoint statuses from %s: response is HTML, not JSON — remote.instances.url must end with /api/v1/endpoints/statuses (not the UI base path)", instance.URL)
		} else {
			logr.Errorf("[api.getEndpointStatusesFromRemoteInstances] Failed to decode endpoint statuses from %s: %s", instance.URL, err.Error())
		}
		return nil
	}
	for _, endpointStatus := range endpointStatuses {
		applyRemoteEndpointPresentation(endpointStatus, instance, instanceIndex, remote.PrefixedKey(instanceIndex, endpointStatus.Key))
	}
	return endpointStatuses
}

func getEndpointStatusesFromRemoteInstances(remoteConfig *remote.Config, forwardHeaders http.Header) ([]*endpoint.Status, error) {
	if remoteConfig == nil || len(remoteConfig.Instances) == 0 {
		return nil, nil
	}

	httpClient := client.GetHTTPClient(remoteConfig.ClientConfig)
	endpointStatusesFromAllRemotes := make([]*endpoint.Status, 0)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for instanceIndex, instance := range remoteConfig.Instances {
		wg.Add(1)
		go func(instanceIndex int, instance remote.Instance) {
			defer wg.Done()
			endpointStatuses := fetchEndpointStatusesFromRemoteInstance(remoteConfig, instance, instanceIndex, forwardHeaders, httpClient)
			if len(endpointStatuses) == 0 {
				return
			}
			mu.Lock()
			endpointStatusesFromAllRemotes = append(endpointStatusesFromAllRemotes, endpointStatuses...)
			mu.Unlock()
		}(instanceIndex, instance)
	}
	wg.Wait()

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

	maxBody := cfg.Remote.MaxResponseBodySize()
	httpClient := client.GetHTTPClient(cfg.Remote.ClientConfig)
	queryString := string(c.Context().URI().QueryString())
	forwardHeaders := forwardHeadersFromContext(c)

	var sawNotFound bool
	var lastTransportErr error

	for _, target := range targets {
		if isRemoteCircuitOpen(target.instance.URL, cfg.Remote.CircuitBreaker) {
			logr.Debugf("[api.proxyRemoteEndpointStatus] Skipping %s because its circuit breaker is open", target.instance.URL)
			return true, c.Status(http.StatusServiceUnavailable).SendString("remote instance circuit breaker is open")
		}

		requestURL := appendQueryString(target.instance.BuildEndpointURL(target.remoteKey, "/statuses"), queryString)
		request, err := http.NewRequestWithContext(c.Context(), http.MethodGet, requestURL, http.NoBody)
		if err != nil {
			logr.Errorf("[api.proxyRemoteEndpointStatus] Failed to create request for %s: %s", requestURL, err.Error())
			continue
		}
		target.instance.ApplyRequestHeaders(request, forwardHeaders)
		response, err := httpClient.Do(request)
		if err != nil {
			recordRemoteCircuitFailure(target.instance.URL, cfg.Remote.CircuitBreaker)
			logr.Errorf("[api.proxyRemoteEndpointStatus] Failed to retrieve %s: %s", requestURL, err.Error())
			lastTransportErr = err
			continue
		}
		body, err := readBoundedBody(response.Body, maxBody)
		closeResponseBody(response.Body, "api.proxyRemoteEndpointStatus")
		if errors.Is(err, errResponseBodyTooLarge) {
			recordRemoteCircuitFailure(target.instance.URL, cfg.Remote.CircuitBreaker)
			logr.Errorf("[api.proxyRemoteEndpointStatus] Response from %s exceeded %d bytes", requestURL, maxBody)
			return true, c.Status(502).SendString("remote response body too large")
		}
		if err != nil {
			recordRemoteCircuitFailure(target.instance.URL, cfg.Remote.CircuitBreaker)
			logr.Errorf("[api.proxyRemoteEndpointStatus] Failed to read response from %s: %s", requestURL, err.Error())
			lastTransportErr = err
			continue
		}
		recordRemoteCircuitSuccess(target.instance.URL)
		if response.StatusCode == http.StatusNotFound {
			sawNotFound = true
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
	if lastTransportErr != nil {
		return sendRemoteGatewayError(c, lastTransportErr)
	}
	if sawNotFound {
		return false, nil
	}
	return false, nil
}

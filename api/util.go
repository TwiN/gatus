package api

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/TwiN/gatus/v5/client"
	"github.com/TwiN/gatus/v5/config"
	"github.com/TwiN/logr"
	"github.com/gofiber/fiber/v2"
)

const (
	// DefaultPage is the default page to use if none is specified or an invalid value is provided
	DefaultPage = 1

	// DefaultPageSize is the default page size to use if none is specified or an invalid value is provided
	DefaultPageSize = 50
)

func extractPageAndPageSizeFromRequest(c *fiber.Ctx, maximumNumberOfResults int) (page, pageSize int) {
	var err error
	if pageParameter := c.Query("page"); len(pageParameter) == 0 {
		page = DefaultPage
	} else {
		page, err = strconv.Atoi(pageParameter)
		if err != nil {
			page = DefaultPage
		}
		if page < 1 {
			page = DefaultPage
		}
	}
	if pageSizeParameter := c.Query("pageSize"); len(pageSizeParameter) == 0 {
		pageSize = DefaultPageSize
	} else {
		pageSize, err = strconv.Atoi(pageSizeParameter)
		if err != nil {
			pageSize = DefaultPageSize
		}
	}
	if page == 1 && pageSize > maximumNumberOfResults {
		// If the page is 1 and the page size is greater than the maximum number of results, return
		// no more than the maximum number of results
		pageSize = maximumNumberOfResults
	} else if pageSize < 1 {
		pageSize = DefaultPageSize
	}
	return
}

func queryRemoteHandler(c *fiber.Ctx, cfg *config.Config, logName string, key string) error {
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

	instance := cfg.Remote.Instances[remoteNr-1]
	remoteKey = strings.Replace(remoteKey, instance.EndpointPrefix, "", 1)
	remoteConfig := cfg.Remote
	httpClient := client.GetHTTPClient(remoteConfig.ClientConfig)

	// TODO: better integration
	remoteApi, err := url.Parse(instance.URL)
	if err != nil {
		return c.Status(400).SendString(fmt.Sprint("Failed to parse remote URL: ", instance.URL))
	}

	// Assume scheme is identical which may be a problem if accessed from the client in the future
	newUrlStr := fmt.Sprint(remoteApi.Scheme, "://", remoteApi.Host)
	newUrl, err := url.Parse(newUrlStr)
	if err != nil {
		return c.Status(400).SendString(fmt.Sprint("Failed to parse remote base URL: ", newUrlStr))
	}

	originalUrl, err := url.Parse(c.OriginalURL())
	if err != nil {
		return c.Status(400).SendString(fmt.Sprint("Failed to parse original URL: ", c.OriginalURL()))
	}

	newUrl.Path = strings.Replace(originalUrl.Path, key, remoteKey, 1)
	newUrl.RawQuery = originalUrl.RawQuery

	logr.Infof("Querying remote endpoint %s", newUrl.String())
	response, err := httpClient.Get(newUrl.String())
	if err != nil {
		msg := fmt.Sprintf("[api.%s/helper] Failed to query %s from remote %s: %s", logName, key, newUrl, err.Error())
		return c.Status(500).SendString(msg)
	}

	contentType := response.Header.Get("Content-Type")
	output, err := io.ReadAll(response.Body)
	if err != nil {
		msg := fmt.Sprintf("[api.%s] Failed to read remote response for %s from %s: %s", logName, key, newUrl, err.Error())
		return c.Status(500).SendString(msg)
	}
	_ = response.Body.Close()

	c.Set("Content-Type", contentType)
	c.Set("Gatus-Remote", newUrlStr)
	return c.Status(response.StatusCode).Send(output)
}

func queryRemoteOrLocalHandler(cfg *config.Config, localHandler func(*fiber.Ctx) error, logName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key, err := url.QueryUnescape(c.Params("key"))
		if err != nil {
			return c.Status(400).SendString("invalid key encoding")
		}

		if !strings.HasPrefix(key, "_gatus_remote_") {
			// Not querying a remote key, passing to the local handler
			return localHandler(c)
		}

		return queryRemoteHandler(c, cfg, logName, key)
	}
}

func queryRemoteOrLocalHandlerWithCfg(cfg *config.Config, localHandler func(*config.Config) fiber.Handler, logName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key, err := url.QueryUnescape(c.Params("key"))
		if err != nil {
			return c.Status(400).SendString("invalid key encoding")
		}

		if !strings.HasPrefix(key, "_gatus_remote_") {
			// Not querying a remote key, passing to the local handler
			localHandler(cfg)(c)
			return nil
		}

		return queryRemoteHandler(c, cfg, logName, key)
	}
}

package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/TwiN/gatus/v5/config/web"
	"github.com/TwiN/logr"
)

const healthCheckTimeout = 5 * time.Second

// runHealthCheck queries the /health endpoint of the Gatus instance described by the configuration and returns the
// process exit code: 0 if the instance is healthy, 1 otherwise.
//
// This is meant to be used as a container health check (e.g. `HEALTHCHECK CMD ["/gatus", "healthcheck"]`), since the
// Docker image is built from scratch and therefore has no shell, curl or wget.
func runHealthCheck() int {
	// Only print errors, so that the configuration loading logs don't pollute the health check output
	logr.SetThreshold(logr.LevelError)
	cfg, err := loadConfiguration()
	if err != nil {
		fmt.Printf("unhealthy: failed to load configuration: %s\n", err.Error())
		return 1
	}
	if err := checkHealth(healthCheckURL(cfg.Web)); err != nil {
		fmt.Printf("unhealthy: %s\n", err.Error())
		return 1
	}
	fmt.Println("healthy")
	return 0
}

// healthCheckURL returns the URL of the /health endpoint of the server described by the web configuration.
// If the server listens on all interfaces, the loopback address is used instead.
func healthCheckURL(webConfig *web.Config) string {
	scheme := "http"
	if webConfig.HasTLS() {
		scheme = "https"
	}
	host := webConfig.Address
	if ip := net.ParseIP(host); host == "" || (ip != nil && ip.IsUnspecified()) {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s://%s/health", scheme, net.JoinHostPort(host, strconv.Itoa(webConfig.Port)))
}

// checkHealth sends a GET request to the given URL and returns an error unless the response status is 200
func checkHealth(url string) error {
	client := &http.Client{
		Timeout: healthCheckTimeout,
		Transport: &http.Transport{
			// The certificate is issued for the public hostname rather than the address we're connecting to,
			// and we're only checking our own server, so there's no point in verifying it.
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned status code %d", url, response.StatusCode)
	}
	return nil
}

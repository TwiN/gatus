package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var (
	secureHTTPClient   *http.Client
	insecureHTTPClient *http.Client
)

// GetHTTPClient returns the shared HTTP client
func GetHTTPClient(insecure bool) *http.Client {
	if insecure {
		if insecureHTTPClient == nil {
			insecureHTTPClient = &http.Client{
				Timeout: time.Second * 10,
				Transport: &http.Transport{
					MaxIdleConns:        100,
					MaxIdleConnsPerHost: 20,
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}
		return insecureHTTPClient
	}
	if secureHTTPClient == nil {
		secureHTTPClient = &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
			},
		}
	}
	return secureHTTPClient
}

// CanCreateConnectionToTCPService checks whether a connection can be established with a TCP service
func CanCreateConnectionToTCPService(address string) bool {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

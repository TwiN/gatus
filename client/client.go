package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

var (
	secureHttpClient   *http.Client
	insecureHttpClient *http.Client
)

// GetHttpClient returns the shared HTTP client
func GetHttpClient(insecure bool) *http.Client {
	if insecure {
		if insecureHttpClient == nil {
			insecureHttpClient = &http.Client{
				Timeout: time.Second * 10,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
				},
			}
		}
		return insecureHttpClient
	} else {
		if secureHttpClient == nil {
			secureHttpClient = &http.Client{
				Timeout: time.Second * 10,
			}
		}
		return secureHttpClient
	}
}

// CanCreateConnectionToTcpService checks whether a connection can be established with a TCP service
func CanCreateConnectionToTcpService(address string) bool {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}
